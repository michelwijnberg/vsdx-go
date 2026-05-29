# vsdx-go

Go library voor het lezen, bewerken en schrijven van Microsoft Visio (.vsdx) bestanden.
Port van de Python [vsdx](https://github.com/dave-howard/vsdx) library (v0.6.1).

## Go Package Structuur

```
vsdx-go/
├── go.mod
├── vsdx/                       # Alle library code in één package (40 source + 30 test files)
│   ├── doc.go                  # Package-level documentatie (74 lines)
│   │
│   │── # Core types
│   ├── vsdxfile.go             # VisioFile: Open/Close/Save, page management, doc props,
│   │                           #   refresh hooks (palette/fonts/HLinks/triggers) (2086 lines)
│   ├── page.go                 # Page: shapes, search, connects, dimensions, layers,
│   │                           #   backgrounds (609 lines)
│   ├── shape.go                # Shape: positie, tekst, stijl, cellen, hiërarchie,
│   │                           #   3D effects, locks, Char/Para, Txt-frame (2825 lines)
│   ├── cell.go                 # Cell: name/value/formula/unit/error (84 lines)
│   ├── connect.go              # Connect: from/to shape relaties (66 lines)
│   ├── data_property.go        # DataProperty: custom shape properties met master
│   │                           #   inheritance (123 lines)
│   │
│   │── # Geometry
│   ├── geometry.go             # Geometry, GeometryRow, GeometryCell: shape paden
│   │                           #   + builders (1018 lines)
│   ├── geometry_resolve.go     # GeometryResolver: NURBS→Bezier, arc conversie,
│   │                           #   arrow setbacks (1771 lines)
│   │
│   │── # SVG Rendering
│   ├── svg.go                  # ShapeToSVG: per-shape SVG renderer met arrows,
│   │                           #   text, line patterns (818 lines)
│   ├── svg_emit.go             # SVG emitter: clean SVG generatie met marker
│   │                           #   definitions, shadow/soft-edges filters (957 lines)
│   ├── svg_parse.go            # SVG parser: element extractie voor vergelijking (92 lines)
│   ├── render_tree.go          # RenderTree: hiërarchische render tree met transform
│   │                           #   propagatie, FlipX/Y, hyphen-aware text wrap (958 lines)
│   ├── render_validate.go      # RenderValidator: transforms, connectors, z-order
│   │                           #   validatie (322 lines)
│   ├── transform.go            # Transform: 2D affine transformaties en matrix
│   │                           #   operaties (270 lines)
│   ├── gradient.go             # Gradient: fill gradients (linear + radial), CW-from-X
│   │                           #   angle convention (266 lines)
│   ├── fillpattern.go          # 8×8 bitmap fill patterns 2-9 + 25-26 (277 lines)
│   ├── shadow.go               # Shadow: drop shadow effecten + feDropShadow filter (127 lines)
│   ├── effective_style.go      # EffectiveStyle: computed style met theme/master/
│   │                           #   stylesheet inheritance (945 lines)
│   │
│   │── # Features
│   ├── foreign.go              # AddImage, AddShape, GroupShapes (met Angle/Flip
│   │                           #   defaults), SetForeignData (461 lines)
│   ├── template.go             # RenderTemplate: Jinja2-achtige directives (490 lines)
│   ├── diff.go                 # VisioFileDiff: twee .vsdx bestanden vergelijken (241 lines)
│   ├── media.go                # Media: embedded template shapes voor connectors (67 lines)
│   ├── formula.go              # FormulaEvaluator: volledige formule-evaluatie,
│   │                           #   175+ functies (2454 lines)
│   ├── routing.go              # Router: A* pathfinding voor auto-routing
│   │                           #   connectors (414 lines)
│   ├── export.go               # ExportPNG, ExportPDF: raster/vector export via
│   │                           #   externe tools (318 lines)
│   ├── validate.go             # Validate: schema validation en error recovery (703 lines)
│   │
│   │── # Stencils & Masters
│   ├── master.go               # CreateMaster, DeleteMaster, DuplicateMaster (322 lines)
│   ├── stencil.go              # Stencil: .vssx stencil bestanden (357 lines)
│   ├── theme.go                # Theme: document themes, effects, variants,
│   │                           #   QuickStyle (1388 lines)
│   ├── styles.go               # StyleSheet: style inheritance en toepassing (398 lines)
│   │
│   │── # Comments & Data Links
│   ├── comments.go             # Comments: document/shape comments + authors (423 lines)
│   ├── linegradient.go         # LineGradient: stroke gradients + Reviewer/
│   │                           #   Annotation (462 lines)
│   ├── datalink.go             # DataLink: DataConnections, DataRecordSets (356 lines)
│   │
│   │── # Support
│   ├── cellname.go             # CellName constants: 70+ cel definities incl.
│   │                           #   3D/effects (143 lines)
│   ├── compat.go               # Markup Compatibility: mc:AlternateContent,
│   │                           #   mc:Ignorable (213 lines)
│   ├── errors.go               # Sentinel errors: ErrInvalidFileType, FileError (40 lines)
│   ├── types.go                # Result structs: Point, Rect (24 lines)
│   ├── namespace.go            # XML namespace constants incl. McCompatNS (17 lines)
│   ├── util.go                 # writeFile helper (26 lines)
│   │
│   ├── vsdx_test.go            # Main test suite (7795 lines)
│   ├── foreign_test.go         # Image/group tests (726 lines)
│   ├── svg_test.go             # Per-shape SVG tests (673 lines)
│   ├── svg_emit_test.go        # Full SVG emit tests (356 lines)
│   ├── golden_test.go          # Golden file tests voor SVG rendering (393 lines)
│   ├── transform_test.go       # Transform tests (185 lines)
│   ├── effective_style_test.go # Style tests (180 lines)
│   ├── contracts_helpers_test.go         # Shared contract-test helpers (384 lines)
│   ├── *_contracts_test.go     # 12 mutator-contract test files
│   ├── ec*_test.go             # 7 EC (edge-case) regression test files
│   ├── master_isolation_test.go          # Master isolation tests (131 lines)
│   ├── master_save_test.go     # Master save tests (110 lines)
│   └── style_setters_f_policy_test.go    # Style setter F-attribute policy (210 lines)
│
├── internal/renderpage/        # Page-level SVG assembler (shared by render-compare
│                               # and mutation-corpus-gen)
├── cmd/render-compare/         # Vergelijkt library SVG met Visio golden exports
├── cmd/render-audit/           # Validatie: transforms, connectors, z-order, arrows
├── cmd/text-compare/           # Vergelijkt tekst posities tussen SVGs
├── cmd/stencil-diag/           # Diagnostic tool voor stencil bestanden
├── cmd/batch-fixture-gen/      # Genereert per-thema mutation fixtures
├── cmd/mutation-corpus-gen/    # Bouwt mutation-render corpus uit fixture-VSDX
├── cmd/comprehensive-gen/      # Genereert comprehensive-features.vsdx (9 thema's)
├── cmd/comprehensive-compare/  # Per-thema render vergelijking Visio↔vsdx-go
├── cmd/probe-conn/             # Diagnostic voor connector geometry
├── svg-compare/cmd/            # 10 reverse-engineering CLI's voor SVG-diff
├── testdata/golden/            # Golden test fixtures voor SVG rendering
├── tests/                      # Test fixture .vsdx bestanden (20+ files)
├── vsdx-svg/                   # Visio golden SVG corpus voor SSIM benchmarks
└── docs/MS-VSDX.pdf            # Microsoft VSDX format specificatie (468 pagina's)
```

## Architectuur

### Data Flow

**Openen:**
```
.vsdx (ZIP) → map[string][]byte (in-memory) → etree XML parse
  → VisioFile.Pages []*Page       (vanuit visio/pages/)
  → Page.shapes() []*Shape        (vanuit <Shapes> elementen)
  → Shape.Cells, Geometry, etc.   (vanuit child XML elementen)
  → VisioFile.MasterPages []*Page (vanuit visio/masters/)
```

**Opslaan:**
```
SaveVsdxBytes() canonical-form normalisaties:
  → refreshDocumentColorPalette()   (uniek #RRGGBB → ColorEntry)
  → refreshFaceNames()              (Char.Font → FaceName registratie)
  → refreshAppXMLHLinks()           (Hyperlink sections → HLinks vector)
  → refreshPageRecalcTriggers()     (per page een RecalcColor Trigger)
  → normalizeTextFormatMarkers()    (Section Character/Paragraph → cp/pp markers)
  → stripWindowsChildren()          (Visio strip Window state on save)
Gewijzigde etree XML → serialize naar []byte
  → update map[string][]byte entries
  → schrijf nieuw ZIP-archief naar disk
```

### Shape Property Resolution

Properties worden opgelost via master-inheritance chain:
```
shape.CellValue("PinX")
  → check eigen <Cell N="PinX">
  → zo niet, check MasterShape().CellValue("PinX")
  → zo niet, return ""
```

Dit geldt voor cells, text, data properties, en geometry.

### Key Types

| Type | Bestand | Verantwoordelijkheid |
|------|---------|---------------------|
| `VisioFile` | `vsdxfile.go` | Hoofd-entrypoint: ZIP openen/opslaan, pagina-beheer, save-time normalisatie |
| `Page` | `page.go` | Pagina of master-pagina: shapes, connects, afmetingen, layers, backgrounds |
| `Shape` | `shape.go` | Shape of groep: tekst, positie, stijl, cellen, hiërarchie, protection |
| `ShapeParent` | `shape.go` | Interface voor Shape.Parent (`*Page` of `*Shape`) |
| `Cell` | `cell.go` | Naam/waarde/formule paar uit XML Cell element |
| `DataProperty` | `data_property.go` | Custom properties met master inheritance |
| `Connect` | `connect.go` | Verbinding tussen twee shapes |
| `Geometry` | `geometry.go` | Shape pad-definitie + builders (MoveTo, LineTo, ArcTo, etc.) |
| `GeometryResolver` | `geometry_resolve.go` | NURBS→Bezier conversie, arc→path, arrow setbacks |
| `RenderTree` | `render_tree.go` | Hiërarchische shape tree met transform propagatie |
| `EffectiveStyle` | `effective_style.go` | Computed style met theme/master/stylesheet inheritance |
| `Transform` | `transform.go` | 2D affine transformatie matrix |
| `Gradient` | `gradient.go` | Fill gradient met stops en angle |
| `Shadow` | `shadow.go` | Drop shadow met offset, blur, kleur |
| `Theme` | `theme.go` | Document theme met kleuren en fonts |
| `Stencil` | `stencil.go` | .vssx stencil bestand met masters |
| `Router` | `routing.go` | A* pathfinding voor connector routing |
| `ValidationResult` | `validate.go` | Schema validation resultaten |
| `Comment`, `Author` | `comments.go` | Document/shape comments met authors |
| `LineGradient` | `linegradient.go` | Stroke gradient met stops |
| `Reviewer`, `Annotation` | `linegradient.go` | Review markup |
| `DataConnection` | `datalink.go` | External data source connection |
| `DataRecordSet` | `datalink.go` | Data records gelinkt aan shapes |
| `Point`, `Rect` | `types.go` | Gestructureerde return waarden |
| `CellName` | `cellname.go` | Type alias + 70+ constants voor cell namen |
| `FileError` | `errors.go` | Error type met path en wrapping |

### Interfaces

- **`ShapeParent`** - Unexported method interface, geïmplementeerd door `*Page` en `*Shape`. Maakt `Shape.Remove()` type-safe.
- **`GeometryCellParent`** - Marker interface voor `*Geometry` en `*GeometryRow`.

### Shape Secties (XML Section types)

De library leest en schrijft de volgende VSDX shape secties:

| Sectie | Lezen | Schrijven | Methods |
|--------|-------|-----------|---------|
| **Character** | ✓ | ✓ | `SetCharBold`, `SetCharItalic`, `SetCharUnderline`, `SetCharSize`, `SetCharFont`, `SetTextColor` |
| **Paragraph** | ✓ | ✓ | `SetParagraphAlign` (AlignLeft/Center/Right/Justify) |
| **Geometry** | ✓ | ✓ | `AddGeometry`, `AddGeometryRect`, `AddMoveTo/LineTo/RelMoveTo/RelLineTo/ArcTo` |
| **Property** | ✓ | ✓ | `DataProperties`, `AddDataProperty`, `SetValue`, `GetAttribute` |
| **Hyperlink** | ✓ | ✓ | `AddHyperlink(address, description)` met Description/SubAddress/NewWindow/SortKey |
| **Connection** | ✓ | ✓ | `AddConnectionPoint(x, y)` met T='Connection', AutoGen + Prompt cells |
| **Layer** | ✓ | ✓ | `Page.AddLayer(name)` (canonical 11-cell row), `Shape.SetLayerMember("0;1")` |
| **Protection** | partial | partial | `SetLockMove`/`SetLockSize`/etc. — schrijven nu als direct shape cells, niet in Protection section |
| **User** | ✓ | ✓ | `AddUserCell(name, value)`, `UserCellValue(name)` |
| **ForeignData** | ✓ | ✓ | `AddImage`, `SetForeignData` |
| **Scratch** | ✓ | ✓ | `ScratchCells()`, `AddScratchCell(x, y, a, b, c, d)` |
| **Actions** | ✓ | ✓ | `Actions()`, `AddAction(name, menu, action)` |
| **Field** | ✓ | ✓ | `Fields()`, `AddField(type, value, format)` |
| **Control** | ✓ | ✓ | `Controls()`, `AddControl(name, x, y, tip)` |
| **Tabs** | ✓ | ✓ | `TabStops()`, `AddTabStop(position, alignment)` |
| **FillGradient** | ✓ | ✓ | `FillGradient()`, `SetFillGradient(angle, stops)` |
| **LineGradient** | ✓ | ✓ | `LineGradient()`, `SetLineGradient(angle, stops)` |
| **Reviewer** | ✓ | ✓ | `Reviewers()`, `AddReviewer(name, initials, color)`, `DeleteReviewer(id)` |
| **Annotation** | ✓ | ✓ | `Annotations()`, `AddAnnotation(x, y, reviewerID, comment)`, `DeleteAnnotation(id)` |
| **SmartTag** | ✓ | ✓ | `SmartTags()`, `AddSmartTag(name, x, y, description)` |
| **ActionTag** | ✓ | ✓ | `ActionTags()`, `AddActionTag(name, x, y, tagName, description)` |
| **ConnectionABCD** | ✓ | ✓ | `AddConnectionABCD(x, y, dirX, dirY, connType)` — canonical: rows met `T='ConnectionABCD'` in een `Section N='Connection'`, met X/Y/A/B/C/D cells |

## VSDX Bestandsformaat

Een `.vsdx` bestand is een ZIP-archief met XML-bestanden:

```
_rels/.rels                      Package relationships (root rels)
[Content_Types].xml              Content type mappings (Override per page + parts)
docProps/app.xml                 Extended properties (pagina-telling, HLinks)
docProps/core.xml                Core properties (titel, auteur, datum)
docProps/custom.xml              Custom properties (user-defined)
visio/document.xml               Stijlen/stylesheets, FaceNames, ColorEntry
visio/pages/pages.xml            Paginadefinities (namen, IDs, RecalcColor triggers)
visio/pages/page1.xml            Pagina-inhoud (shapes, connects)
visio/pages/_rels/pages.xml.rels Per-page rels (master refs)
visio/masters/masters.xml        Master shape definities
visio/masters/master1.xml        Individuele master shapes
visio/theme/theme1.xml           Theme definities (kleuren, fonts, effects)
visio/windows.xml                Session state (gestript op save)
```

XML namespace: `http://schemas.microsoft.com/office/visio/2012/main`
Extended-types namespace: `http://schemas.openxmlformats.org/officeDocument/2006/docPropsVTypes` (prefix `vt:`)

### Shape XML structuur

```xml
<Shape ID="1" MasterShape="2" Master="3">
  <Cell N="PinX" V="3.5"/>
  <Cell N="Width" V="2.0"/>
  <Text><cp IX="0"/>Hello World</Text>
  <Section N="Property">
    <Row N="Status"><Cell N="Value" V="Active"/></Row>
  </Section>
  <Section N="Geometry1">
    <Row T="MoveTo" IX="1"><Cell N="X" V="0"/><Cell N="Y" V="0"/></Row>
    <Row T="LineTo" IX="2"><Cell N="X" V="1"/><Cell N="Y" V="1"/></Row>
  </Section>
  <Shapes><!-- child shapes --></Shapes>
</Shape>
```

## Commando's

```bash
# Go tests
cd /home/michel/vsdx-go && go test ./vsdx/... -v

# Enkele test
cd /home/michel/vsdx-go && go test ./vsdx/... -run TestName -v

# Test coverage rapport
go test ./vsdx/... -cover -count=1

# SVG render comparison (Visio gouden corpus → SSIM ~0.987)
go run ./cmd/render-compare/...
cd render-compare-output && python3 compute_ssim.py

# Comprehensive corpus regen + compare (9 thema's → SSIM ~0.959)
go run ./cmd/comprehensive-gen/...
go run ./cmd/comprehensive-compare/...
cd render-compare-output-comprehensive && python3 compute_ssim.py
```

## Development Tools

### Reverse Engineering Tools

| Tool | Locatie | Doel |
|------|---------|------|
| `render-compare` | `cmd/render-compare/` | Library-SVG vs Visio's golden SVG exports |
| `render-audit` | `cmd/render-audit/` | Validatie: transforms, connectors, z-order, arrows, render tree |
| `text-compare` | `cmd/text-compare/` | Tekst posities tussen golden en rendered SVG |
| `stencil-diag` | `cmd/stencil-diag/` | Diagnostische tool voor stencil (.vssx) bestanden |
| `batch-fixture-gen` | `cmd/batch-fixture-gen/` | Genereert per-thema mutation fixtures |
| `mutation-corpus-gen` | `cmd/mutation-corpus-gen/` | Bouwt mutation-render corpus |
| `comprehensive-gen` | `cmd/comprehensive-gen/` | Genereert `comprehensive-features.vsdx` (9 thema's, 170+ shapes) |
| `comprehensive-compare` | `cmd/comprehensive-compare/` | Per-thema render vergelijking vs Visio's resave |
| `probe-conn` | `cmd/probe-conn/` | Diagnostic voor connector geometry |
| `svg-compare/cmd/*` | `svg-compare/cmd/` | 10 reverse-engineering CLI's (shape-inspect, bbox-compare, master-inspect, path-analyze, debug-nurbs, …) |

**render-compare workflow:**
1. Leest `.vsdx` bestanden uit `vsdx-svg/`
2. Zoekt bijbehorende golden `.svg` exports (door Visio gegenereerd)
3. Parseert de golden SVG om te bepalen welke pagina gerenderd moet worden (bijv. "Page-2")
4. Rendert dezelfde pagina met de library
5. Schrijft beide SVGs naar `render-compare-output/` voor vergelijking
6. Genereert `compare.html` voor side-by-side visuele inspectie

**comprehensive-gen workflow:**
1. Bouwt een 9-pagina test-VSDX met alle features per thema
2. Pagina's: Shapes, Fills, Lines, Arrows, Text, Transforms, Effects, Connectors, Data
3. Schrijft naar `vsdx-svg/comprehensive/comprehensive-features.vsdx`
4. Visio's resave (handmatig gemaakt) staat als `comprehensive-features-visio-saved.vsdx`
5. Per-thema XML byte-diff tegen Visio's resave is de writer-canonical regressietest

**VSDX inspectie met Python:**
```bash
# Extract VSDX en inspecteer XML structuur
unzip -d /tmp/extract file.vsdx
cat /tmp/extract/visio/pages/page1.xml | xmllint --format -

# Parse shapes met Python
python3 -c "
import xml.etree.ElementTree as ET
tree = ET.parse('/tmp/extract/visio/pages/page1.xml')
ns = {'ns': 'http://schemas.microsoft.com/office/visio/2012/main'}
for shape in tree.findall('.//ns:Shape', ns):
    print(f'Shape {shape.get(\"ID\")}: {shape.get(\"Type\")}')"
```

### Golden SVG Exports

Golden SVGs zijn referentie-exports gemaakt door Microsoft Visio zelf. Ze dienen als ground truth voor rendering validatie.

**Locatie:** `vsdx-svg/*.svg` (naast de bijbehorende `.vsdx` bestanden)

**Hoe te genereren:**
1. Open `.vsdx` in Microsoft Visio
2. File → Export → Change File Type → SVG
3. Sla op met dezelfde basename als het `.vsdx` bestand

**Kenmerken van Visio SVG exports:**
- XML comment met pagina-info: `<!-- Generated by Microsoft Visio, SVG Export filename.svg Page-2 -->`
- Visio-specifieke namespace: `xmlns:v="http://schemas.microsoft.com/visio/2003/SVGExtensions/"`
- Verbose structuur met geneste `<g>` elementen en metadata
- CSS classes voor styling (`.st1`, `.st2`, etc.)
- Marker definitions met `<use xlink:href="#lend5">` patronen
- Shapes met `id="shape123-45"` en `v:mID="123"`

**Huidige golden corpus:**
```
vsdx-svg/
├── ad-hoc-exploration.vsdx + .svg          # Page-2, Azure architectuur
├── logical-architecture.vsdx + .svg         # Page-1, logische architectuur
├── physical-architecture-*.vsdx + .svg      # Page-1, diverse Azure topologieën
├── reference-architecture.vsdx + .svg       # Page-1, referentie architectuur
└── comprehensive/
    ├── comprehensive-features.vsdx          # vsdx-go-gegenereerd (9 thema's)
    ├── comprehensive-features-visio-saved.vsdx  # Visio's resave (byte-diff baseline)
    └── comprehensive-features-*.svg         # Per-thema Visio SVG exports
```

**Vergelijking output:** `render-compare-output/`, `render-compare-output-comprehensive/`
- `*_golden.svg` / `*_visio.svg` - Kopie van Visio's export
- `*_rendered.svg` / `*_vsdxgo.svg` - Library's rendering
- `compare.html` - Side-by-side HTML vergelijking
- `SSIM_REPORT.md` - SSIM scores per file/thema

## Afhankelijkheden

- `github.com/beevik/etree` v1.4.1 - XML parsing met XPath-achtige navigatie
- Go 1.21+

**Optioneel voor SSIM benchmarks:**
- `rsvg-convert` (librsvg) — SVG → PNG rasterisatie
- Python 3 + `numpy` + `scikit-image` + `Pillow` — SSIM-berekening

## Referentie Documentatie

### MS-VSDX spec
- `docs/MS-VSDX.pdf` - Officiële Microsoft VSDX format specificatie (468 pagina's)
  - §2.2.5.3.3.1 Cell Default Values
  - §2.2.5.4 Inheritance - 5 types (wij ondersteunen master-to-shape)
  - §2.2.7.3 Effects (Bevel, Glow, Reflection, Soft Edges, Sketch, Rotation 3D)
  - §2.2.7.4.3 QuickStyle slices (7 matrices + Type/Variation)
  - §2.2.10 Markup Compatibility (mc:AlternateContent, mc:Ignorable)
  - §2.2.11.2 Formulas - volledige formule grammatica
  - §2.4.2 GeometryRowTypes - 15 types (alle 15 ondersteund)
  - §2.4.4 Cells - complete catalogus van cel definities

### ECMA-376 / OOXML
- `docProps/app.xml` gebruikt extended-properties + variant-types (vt:) namespaces
- HLinks vector structuur per §15.2.12.10

### Interne audit-documenten
- `vsdx/UNSUPPORTED_FEATURES.md` — Model / API / Render support matrix
- `vsdx/DIVERGENCE_STATUS.md` — Per-divergence resolutie met evidence
- `vsdx/RENDER_AUDIT.md` — Render pipeline architectuur
- `vsdx/WRITER_AUDIT.md` — Writer canonical-form audit
- `vsdx/EDIT_CONTRACTS.md` — Mutator contract-tests overzicht
- `vsdx/TEXT_DIVERGENCE_REPORT.md` — Tekst-positie analyse
- `ROADMAP.md` — Historische ontwikkelfases (alles ✓)

## Huidige Status

**Code & tests** (na writer canonicalization sweep, commit a5976f9):
- 40 Go source bestanden, ~23,000 LOC code + ~13,600 LOC tests
- 400 top-level test functions, 609 inclusief subtests (alle passing)
- ~59.5% code coverage (vooral effective_style, gradient, geometry_resolve, render_tree)
- 9 cmd/ tools + 10 svg-compare/cmd/ helpers

**Spec dekking:**
- **100% MS-VSDX section coverage** (21+ sections: alle Char/Para/Geometry/Layer/Hyperlink/
  Connection/User/Property/Action/Field/Control/Tabs/Scratch/ForeignData/Protection/Reviewer/
  Annotation/SmartTag/ActionTag/ConnectionABCD/FillGradient/LineGradient secties)
- **15/15 Geometry row types** (incl. NURBSTo via B-spline → Bezier conversie)
- **175+ formule functies** (volledige formule-evaluatie engine)
- **Volledige style/theme support** (themes, variants, QuickStyle, 7 matrix slices)

**Round-trip met Visio's resave** (comprehensive corpus byte-diff):
- 7/9 pagina's identiek (Shapes, Fills, Lines, Arrows, Transforms, Connectors, Data)
- 2 pagina's met minimale cosmetische deltas:
  - Text page: 30 cells short op één rich-text shape (Visio's Char-row expansie)
  - Effects page: 1 cell artifact (stale Visio-save baseline)
- `Content_Types`, `app.xml`, `document.xml`, `pages.xml(.rels)`, `windows.xml` — identiek

**SSIM render baselines:**
- Visio gouden corpus (8 files): gemiddelde **0.987**
  - 5 files boven 0.99, 3 in 0.97-0.98 (logical/reference/vsdx-test)
- Comprehensive corpus (9 thema's): gemiddelde **0.959**
  - shapes 0.985, transforms 0.980, data 0.978, effects 0.977, lines 0.970,
    connectors 0.966, arrows 0.949, text 0.934, fills 0.889

**SVG Rendering:**
- Line patterns (24 types via stroke-dasharray)
- Arrow markers (45+ types met markerUnits="strokeWidth", setback in points)
- Gradient fills (linear + radial, CW-from-X angle convention)
- Drop shadows (feDropShadow filter)
- Soft edges (feGaussianBlur filter)
- Fill bitmap patterns 2-9 + 25-26 (8×8 pixel-grid rect)
- NURBS/B-spline → Bezier conversie voor connector curves
- Shape rotation + FlipX/Y met proper coordinate transforms
- Render tree met hiërarchische transform propagatie + group inheritance
- Hyphen-aware text wrap voor multiline text
- Hyperlink `<a xlink:href>` wrappers in output SVG

**Authoring features:**
- Master shapes aanmaken/verwijderen/dupliceren, isolatie tests
- Stencils (.vssx), themes, variants
- Background pages (write + render)

**Advanced features:**
- Auto-routing connectors (A* pathfinding, `Router`)
- PNG/PDF export via externe tools
- Schema validation + error recovery
- `TheCel`/`Sheet.N!` formula references

**Data features:**
- Comments/annotations (read+write)
- Data links/recordsets
- Reviewers (read+write)
- Hyperlinks met SortKey/NewWindow

**Package features:**
- Root relationships, core/custom document properties
- Cell U/E attributes (units, errors)
- `vt:` namespace voor extended-properties variant types
- HLinks auto-refresh in app.xml
- FaceName auto-registratie in document.xml
- Color palette auto-refresh

**Section types:**
- SmartTag, ActionTag, ConnectionABCD canonical formaat
- Alle originele 21 Section types

**3D Effect cells** (MS-VSDX §2.2.7.3):
- BevelEffect (13 cells), GlowEffect (3), ReflectionEffect (4), SketchEffect (6),
  Rotation3DEffect (7), SoftEdgesSize
- Model + API support volledig; render-side: drop shadow + soft edges actief,
  Bevel/Glow/Reflection/Sketch/Rotation3D nog niet visueel

**Idiomatisch Go:**
- Cell constants (70+), sentinel errors, typed interfaces, result structs
- Mutator contract-tests met shared helpers (12 contract test files)
- EC (edge-case) regression tests (7 files voor specific defect classes)
