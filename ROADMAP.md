# vsdx-go Roadmap

Doel: een Go library die volledig compatibel is met Microsoft Visio voor
zowel het lezen / bewerken / schrijven van .vsdx als het renderen naar SVG.

---

## Huidige Status

**Last reviewed**: 2026-05-29

| Metric | Waarde |
|---|---|
| Source files | 40 (vsdx/*.go non-test) |
| Test files | 30 |
| Source LOC | ~23,000 |
| Test LOC | ~13,600 |
| Test functions | 400 top-level, 609 incl. subtests |
| Coverage | 59.5% |
| MS-VSDX section types | 21/21 (100%) |
| Geometry row types | 15/15 (100%) |
| Formula functions | 175+ |
| Visio SSIM (gouden corpus) | 0.987 gemiddelde |
| Comprehensive SSIM (9 thema's) | 0.959 gemiddelde |
| Visio resave byte-diff | 22/24 files identiek |

Zie `CLAUDE.md` voor het volledige feature overzicht en architectuur,
`vsdx/UNSUPPORTED_FEATURES.md` voor het Model/API/Render support-matrix.

---

## Voltooide fases

### Fase 1: Rendering Completeness ✓

Alle SVG-render fundamentals: line patterns (24 types), arrow markers (45+
types met `markerUnits="strokeWidth"`), text block positionering, ellipse
geometry, NURBS/B-spline → Bezier conversie, shape rotation, hiërarchische
transforms via RenderTree.

### Fase 2: Visual Polish ✓

Gradient fills (linear + radial, CW-from-X angle conventie), drop shadows
(feDropShadow filter), soft edges (feGaussianBlur), 8×8 bitmap hatch
patterns voor fill patterns 2-9 + 25-26, background pages (write + render).

### Fase 3: Authoring ✓

Master shape CRUD, isolation tests, stencils (.vssx) met master-extractie,
themes + variants + QuickStyle (alle 7 matrix slices + FontColor + Type +
Variation), StyleSheet inheritance.

### Fase 4: Advanced Features ✓

Auto-routing connectors via A* pathfinding (`Router`), PNG/PDF export via
externe tools, schema validation + error recovery, formula evaluator met
175+ functies incl. `TheCel`/`Sheet.N!` cross-shape references.

### Fase 5: Geometry Completion ✓

Alle 15 GeometryRowTypes: MoveTo, LineTo, RelMoveTo, RelLineTo, ArcTo,
EllipticalArcTo, RelEllipticalArcTo, RelCubBezTo, RelQuadBezTo, NURBSTo,
PolylineTo, SplineStart, SplineKnot, InfiniteLine, Ellipse.

### Fase 6: Writer Canonicalization ✓ (2026-05)

Tag-voor-tag byte-diff van vsdx-go's output tegen Visio 2021's resave van
hetzelfde bestand. 22/24 ZIP-files nu identiek; details in
`vsdx/WRITER_AUDIT.md`. Hoogtepunten:

- **38 closed items** verspreid over packaging (vt: namespace, page-rels,
  TitlesOfParts ordering, orphan cleanup op page removal), app.xml (HLinks
  auto-generate, HeadingPairs), document.xml (color palette + FaceName
  auto-refresh), pages.xml (RecalcColor triggers), windows.xml (children
  strip op save), section canonicalisatie (Layer rows, Connection rows,
  Hyperlink rows, ConnectionABCD), per-shape (Group defaults, TxtAngle
  placeholders), en default-stripping (LinePattern=1, LineCap=0,
  arrow-sizes, gradient defaults, alle 3D effect zero-cells).
- **2 open items** (cosmetisch): rich-text Character row expansion op één
  Text-shape (Visio expandeert tot 17 default cells per styled run), en
  één stale-baseline cel op Effects page.

---

## Huidige Focus & Vervolg

Drie open sporen, in volgorde van mijn aanbeveling:

### 1. Round-trip verificatie (lage moeite, hoog signaal)

Open de net byte-matching `comprehensive-features.vsdx` opnieuw in Visio,
sla op, hervergelijk. Als alle 22 files nog steeds matchen en de twee
cosmetische deltas verdwijnen, sluit dat Fase 6 definitief af. Als er
nieuwe verschillen verschijnen wijst dat op writer-bugs in code-paden
die we nog niet via diff hebben gevangen.

**Effort**: ~30 min (handmatige Visio-actie + her-run van het diff script).

### 2. Render fidelity op zwakste thema's

SSIM-baselines tonen waar de renderer nog niet voldoet:

| Thema | SSIM | Vermoedelijke oorzaak |
|---|---|---|
| Fills | 0.889 | Hatch patterns 10-24 nog niet geïmplementeerd; gradient angle convention edge cases |
| Text | 0.934 | Font-metric variaties (rsvg-convert ↔ browser), rich-text formatting runs |
| Arrows | 0.949 | Enkele arrow setbacks + size-multipliers nog niet exact |
| Connectors | 0.966 | A* paths met meerdere knikken/dynamische re-routing |

**Effort**: per thema 1-3 sessies. Fills heeft de grootste delta en is
het meest visueel zichtbaar.

### 3. 3D Effects rendering (write→render gap)

`SetBevelEffect` / `SetGlowEffect` / `SetReflectionEffect` /
`SetSketchEffect` / `SetRotation3DEffect` schrijven nu de cellen
canonical naar VSDX, maar de SVG-renderer negeert ze (alleen
`Shadow` en `SoftEdges` produceren SVG-filters). Implementatie hiervan
zou ons formeel "BETTER than Visio's SVG export" maken — Visio's eigen
SVG-export negeert ook deze effecten.

**Effort**: ~4-6 sessies. Bevel is het meest complex (multi-layer
filter compositie); Glow en Reflection zijn simpeler (feGaussianBlur +
positionering). Sketch vergt path-perturbatie. Rotation3D vergt
perspective-projection wiskunde.

### 4. Long-tail items uit UNSUPPORTED_FEATURES.md

Lage prioriteit, maar wel concrete spec-conform features:

- Hatch patterns 10-24 (nu rendered as solid)
- Image / texture / picture fills (cells geparsed, render skipped)
- Text decorations Strikethru / DblUnderline / Overline (cells in model,
  API ontbreekt)
- Subscript/superscript via Char.Pos
- Letter spacing rendering
- Tab stops in tekst-layout (model+API klaar, render ontbreekt)
- Bullet lists (cells in model, geen API/render)
- Vertical text orientation (TxtAngle werkt; tekst-baseline rotation +
  geheugen ontbreekt)
- CJK / double-byte layout (AsianFont cells in model)
- Custom LineJoin per row
- Clipping paths (ClipPath property)
- Skew transforms
- Layer-based z-order + layer-specifieke styles
- Custom arrow definitions
- Compound arrow heads
- Compound lines (DoubleLine, ParaLine)
- Embossed / debossed effects
- Connector jumps / bridges (line crossings visual)
- Page headers / footers
- Print areas

---

## Test Strategy

### Unit tests
- Elke nieuwe functie krijgt tests
- Edge cases voor rendering
- Round-trip tests (create → save → load → verify)

### Contract tests
- `*_contracts_test.go`: mutator contracten (12 files, shared helpers in
  `contracts_helpers_test.go`)
- `ec*_test.go`: edge-case regressie tests per specific defect class

### Visual tests
- `golden_test.go`: golden file vergelijking tegen `testdata/golden/`
- `cmd/render-compare/`: SVG vs Visio golden export (8 file corpus)
- `cmd/comprehensive-compare/`: per-thema vergelijking (9 thema's)

### Writer round-trip
- `cmd/comprehensive-gen` produceert canonical .vsdx
- Visio 2021 resave produceert baseline
- Python tag-count diff per ZIP-file (zie WRITER_AUDIT.md)

### Performance
- Geen formele benchmarks momenteel
- Manual: comprehensive corpus genereert in <1s op moderne hardware

---

## Dependencies

- `github.com/beevik/etree` v1.4.1 — XML parsing met XPath-achtige navigatie
- Go 1.21+

**Optioneel voor SSIM benchmarks:**
- `rsvg-convert` (librsvg) — SVG → PNG rasterisatie
- Python 3 + `numpy` + `scikit-image` + `Pillow`

---

## Documentatie

### Interne audit-documenten
- `CLAUDE.md` — package overzicht + huidige status (entry point)
- `vsdx/UNSUPPORTED_FEATURES.md` — Model / API / Render support-matrix
- `vsdx/DIVERGENCE_STATUS.md` — per-divergence resolutie + evidence
- `vsdx/RENDER_AUDIT.md` — render pipeline architectuur
- `vsdx/WRITER_AUDIT.md` — writer canonical-form audit (Fase 6 details)
- `vsdx/EDIT_CONTRACTS.md` — mutator contract-tests overzicht
- `vsdx/TEXT_DIVERGENCE_REPORT.md` — tekst-positie analyse

### Spec referentie
- `docs/MS-VSDX.pdf` — Officiële Microsoft VSDX format spec (468 p.)
- ECMA-376 / ISO/IEC 29500 — OOXML standard (impliciet voor app.xml /
  core.xml / vt: namespace)

---

## Conclusie

Het oorspronkelijke roadmap (fases 1-5) is volledig voltooid; Fase 6
(Writer Canonicalization) afgesloten. De library kan diagrammen lezen,
bewerken en schrijven die door Visio worden geaccepteerd zonder
informatieverlies. SVG-rendering is gemiddeld 0.987 SSIM op Visio's eigen
gouden corpus.

De drie genoemde vervolg-sporen (round-trip verificatie, render fidelity
op zwakke thema's, 3D effects rendering) zijn additionele kwaliteits- en
feature-stappen — niet noodzakelijk voor library-functionaliteit als
geheel.
