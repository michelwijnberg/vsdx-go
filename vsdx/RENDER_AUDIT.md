# SVG Renderer Pipeline — Production Status

**Last reviewed**: 2026-05-29

## Architecture Overview

```
Parsed Model (VSDX XML)
  │
  ├─► Effective Style Resolution      [Shape.ComputeEffectiveStyle()]
  │     └─ local → master → stylesheet → theme → defaults
  │     └─ ✓ PRODUCTION
  │
  ├─► Geometry Resolution             [GeometryResolver]
  │     └─ all 15 GeometryRowTypes resolved to SVG paths
  │     └─ NURBS / B-spline → Bezier conversion
  │     └─ arrow setback applied in points (÷72×96 → SVG units)
  │     └─ ✓ PRODUCTION
  │
  ├─► World Transform Resolution      [RenderTreeBuilder]
  │     └─ PinX/PinY, LocPinX/LocPinY, Angle
  │     └─ FlipX / FlipY (geometry mirror, text upright)
  │     └─ parent transforms cascade to children
  │     └─ ✓ PRODUCTION
  │
  ├─► Text Resolution                 [resolveText / wrapTextLines]
  │     └─ multiline word-wrap (hyphen-aware splitWordsForWrap)
  │     └─ alignment, margins, rotation
  │     └─ vertical-rl writing-mode for cardinal angles (±π/2)
  │     └─ tspan-per-line emission for multiline
  │     └─ ✓ PRODUCTION
  │
  ├─► Render Tree Construction        [RenderTreeBuilder.BuildWithScale()]
  │     └─ OrderIndex-based z-order (per MS-VSDX spec)
  │     └─ visibility filtering (NoShow, layer membership)
  │     └─ ✓ PRODUCTION
  │
  └─► SVG Emission                    [SVGEmitter.Emit()]
        └─ pure serialization only — no logic, no lookups
        └─ marker definitions (45+ arrow types)
        └─ filter definitions (drop shadow, soft edges)
        └─ pattern definitions (8×8 bitmaps voor 2-9 + 25-26)
        └─ <a xlink:href> wrapping voor hyperlinks
        └─ deterministic output (sorted maps)
        └─ ✓ PRODUCTION
```

## Public API

| Function | Description | Status |
|----------|-------------|--------|
| `ShapeToSVG(shape, opts...)` | Canonical entry — RenderTree path | ✓ Production |
| `ShapeToSVGWithSize` / `WithPrecision` | Option helpers | ✓ Production |
| Result struct (`SVG`, `BrandColor`, etc.) | Full output with metadata | ✓ Production |
| `renderpage.Render(page, w, h)` | Page-level SVG (internal/renderpage) | ✓ Production |

The legacy renderer is removed; the RenderTree path is the only path.

## Validation Tools

| Tool | Purpose | Location |
|------|---------|----------|
| `cmd/render-compare` | Compare library SVG vs Visio golden exports (8-file corpus) | CLI |
| `cmd/render-audit` | Validate transforms, connectors, z-order, arrows | CLI |
| `cmd/text-compare` | Compare text positions between SVGs | CLI |
| `cmd/comprehensive-compare` | Per-theme render comparison vs Visio resave (9 thema's) | CLI |
| `vsdx/render_validate.go` | Programmatic RenderValidator | Library |
| `TestGoldenFixtures` (`golden_test.go`) | Golden file regression tests | Test suite |

## Golden Test Fixtures

Located in `testdata/golden/`:

| Fixture | Categories | Description |
|---------|------------|-------------|
| simple_rect | geometry, text | Basic rectangle |
| filled_rect | geometry, fill, text | Filled shape |
| house_group | geometry, group, transform | Nested geometries |
| connector_arrow | geometry, markers | Arrow markers |

Plus the broader regression corpus in `vsdx-svg/` (8 architecture diagrams
re-exported from Visio) and `vsdx-svg/comprehensive/` (9-theme feature
matrix re-exported per page).

## Divergence Status (zie DIVERGENCE_STATUS.md)

| Category | Total | Fixed | Intentional | Needs Work |
|----------|-------|-------|-------------|------------|
| path | 14 | 11 | 3 | 0 |
| bounds | 8 | 8 | 0 | 0 |
| text | 88 | 83 | 1 | 4 (edge cases) |
| fill | 50 | 0 | 50 | 0 |
| **Total** | **160** | **102** | **54** | **4** |

De 4 resterende text-divergenties zijn allemaal edge cases met H=0 of
negative-height connectors waarvan de visuele impact minimaal is.

## Known Intentional Divergences

### 1. Arrow Setback Unit Conversion
**RenderTree** converts BeginArrowSize / EndArrowSize from points to SVG
units (÷72×96).
**Legacy code** treated points as SVG units directly (bug).
**Delta**: ~0.91 SVG units per connector.
**Status**: ✓ RenderTree is correct per MS-VSDX §2.2.5.3.3.1.

### 2. Multi-Geometry Fill Colors
**RenderTree** uses actual shape data colors per spec.
**Legacy code** applied an undocumented heuristic that inverted dark fills
to white for secondary geometries of icon shapes (House, Router, Switch).
**Status**: ✓ RenderTree is correct; Visio files should set explicit
colors when intended.

### 3. Z-Order Determination
**RenderTree** uses the OrderIndex property per spec.
**Legacy code** used geometry-count heuristics.
**Status**: ✓ RenderTree follows MS-VSDX §2.2.5.3.3.1.

## SSIM Baselines

Run met `rsvg-convert` + Python `skimage.metrics.structural_similarity`:

| Corpus | Files | Mean | Range |
|---|---|---|---|
| Visio gouden | 8 | 0.987 | 0.969 – 0.997 |
| Comprehensive (9 thema's) | 9 | 0.959 | 0.889 – 0.985 |

Per-thema in comprehensive corpus:

| Thema | SSIM |
|---|---|
| Shapes | 0.985 |
| Transforms | 0.980 |
| Data | 0.978 |
| Effects | 0.977 |
| Lines | 0.970 |
| Connectors | 0.966 |
| Arrows | 0.949 |
| Text | 0.934 |
| Fills | 0.889 |

## Guarantees

### Determinism
- Multiple renders van dezelfde shape produceren byte-identical SVG
- Map iteration order beïnvloedt output niet (gesorteerd in emitter)
- Verified by `TestDeterministicOutput`

### Correctness
- All geometry resolved before emission (no last-minute computation)
- All transforms pre-computed in RenderTree
- No heuristics in SVG emitter (pure serialisatie)
- All style lookups via EffectiveStyle (consistent inheritance)

### Reproducibility
- Golden fixtures pinnen verwachte output
- Regression tests draaien op elke build
- `cmd/render-compare` + SSIM-script geven kwantitatieve regressie-detectie

## Unsupported Features (render-side)

Volledige lijst en classificatie in `UNSUPPORTED_FEATURES.md`. Hoofdpunten:

- **3D effects** (Bevel, Glow, Reflection, Sketch, Rotation3D): cellen
  worden gelezen en geschreven, maar de renderer emitteert nog geen
  visueel effect. Drop shadow en soft edges zijn wel gerenderd.
- **Hatch patterns 10-24**: gerenderd als solid; patterns 2-9 + 25-26
  hebben een 8×8 bitmap implementatie
- **Image / texture / picture fills**: cellen geparsed, render skipped
- **Strikethru / DblUnderline / Overline**: cellen in model, geen API/render
- **Subscript / superscript** (Char.Pos)
- **Letter spacing** rendering (Char.Letterspace cell wel ondersteund)
- **Tab stops** in tekst-layout (Tabs section model + API klaar, render ontbreekt)
- **Bullet lists** (Para.Bullet cells, geen API/render)
- **CJK / double-byte layout** (AsianFont cells in model)
- **Custom LineJoin** per row
- **Clipping paths** (ClipPath property)
- **Skew transforms**
- **Layer-based z-order** + layer-specifieke styles
- **Custom arrow definitions**
- **Compound arrow heads** / **compound lines**
- **Embossed / debossed effects**
- **Connector jumps / bridges** (line crossings visual)

## Test Commands

```bash
# Run all renderer tests
go test ./vsdx/... -v

# Run golden tests
go test ./vsdx/... -run "TestGoldenFixtures" -v

# Regenerate golden fixtures
GENERATE_GOLDEN=1 go test ./vsdx/... -run "TestGenerateGoldenFixtures" -v

# Test determinism
go test ./vsdx/... -run "TestDeterministicOutput" -v

# Compare SVG output tegen Visio gouden corpus + SSIM
go run ./cmd/render-compare/...
cd render-compare-output && python3 compute_ssim.py

# Comprehensive corpus regen + per-thema vergelijking
go run ./cmd/comprehensive-gen/...
go run ./cmd/comprehensive-compare/...
cd render-compare-output-comprehensive && python3 compute_ssim.py
```

## Architecture Compliance

| Requirement | Status |
|-------------|--------|
| No geometry mutation during emission | ✓ |
| No transform computation during emission | ✓ |
| No style resolution during emission | ✓ |
| No connector heuristics | ✓ |
| No ShapeSheet lookups in emitter | ✓ |
| Deterministic serialization | ✓ |
| Golden test coverage | ✓ |
| Documented unsupported features | ✓ |
