# EDIT_CONTRACTS — mutatie-laag van vsdx-go

> Dit document beschrijft wat élke mutator (Set*, Add*, Move, Connect*, …) in vsdx-go contractueel MOET doen. Het is ontstaan uit een audit (Fase 1 + 2, 2026-05) die 6 bugs in de mutatie-laag aan het licht bracht binnen één werksessie. Het document gaat náást [RENDER_AUDIT.md](RENDER_AUDIT.md) en [DIVERGENCE_STATUS.md](DIVERGENCE_STATUS.md): die catalogiseren rendering-divergenties, dit catalogiseert mutatie-correctheid.
>
> **Status**: living document. Per PR die een mutator toevoegt of wijzigt: update de tabel in §10 en (waar nodig) de contracten in §3–9.

---

## 1. Scope

De **mutatie-laag** is het oppervlak van public methods die VSDX-state veranderen:

- Cell-setters (`(*Shape).SetX`, `SetWidth`, `SetFillColor`, `SetCellValue`, `SetCellFormula`, …)
- Geometry-setters (`(*Geometry).Move`, `SetMoveTo`, `AddLineTo`, …)
- Shape-operaties (`(*Shape).Move`, `Remove`, `FindReplace`, `ApplyTextFilter`, …)
- Page-operaties (`(*Page).AddShape`, `GroupShapes`, `AutoSize`, `AutoRouteConnectors`, …)
- File-operaties (`(*VisioFile).AddPage`, `ConnectShapes`, `ConnectShapesWithStyle`, `CopyShape`, …)
- Style/Theme (`SetLineStyleID`, `SetFillStyleID`, `SetQuickStyle`, `ApplyVariant`, …)
- Data (`AddDataProperty`, `LinkToData`, `AddRecord`, …)

Wat **niet** in deze doc valt:

- Read-only properties (X(), Width(), CellValue()). Hun correctheid staat in [doc.go](doc.go) en de Go-docs.
- Rendering (zie RENDER_AUDIT.md).
- ZIP/OPC-laag (zie [vsdxfile.go](vsdxfile.go) commentaar).

---

## 2. Vocabulaire

| Term | Definitie |
|---|---|
| **Cell** | `<Cell N="X" V="..." F="..." U="..." E="..."/>` XML-element met optionele attributen V (value), F (formula), U (units), E (error). Spec: MS-VSDX §2.3.4.2.5. |
| **F="Inh"** | Marker dat de cel-waarde geërfd is van de master. Spec: MS-VSDX §2.2.5.4.6. |
| **THEMEGUARD** | No-op formule die de cel-waarde "vastpint" tegen theme-overschrijvingen. Spec: MS-VSDX §2.5.3.157. |
| **Master** | Een sjabloon-shape gedeeld door instances. Loaded via `loadMasterPages` als `*Page`. |
| **Instance** | Een shape op een page met `Master` of `MasterShape` XML-attribuut. Zie [shape.go:171](shape.go) (`MasterShape()`). |
| **Subshape / ChildShape** | Een geneste shape binnen een group of compound master (`<Shape>` binnen `<Shapes>` binnen `<Shape>`). Zie `(*Shape).ChildShapes()`. |
| **Local cell** | Een cell waarvan `c.xml.Parent()` gelijk is aan `s.xml`. De waarde is op deze shape gedefinieerd, niet geërfd. |
| **Aliased geometry** | Een `*Geometry` waarvan `g.xml.Parent() != g.shape.xml`. Pointer naar master XML; mutaties moeten eerst localiseren. |
| **Localize** | Het deep-copyen van master XML naar de instance shape, zodat mutaties lokaal landen. Implementatie: `(*Geometry).localize()` in [geometry.go](geometry.go). |

---

## 3. De drie mutatie-lagen

De meeste bugs in de Fase 1 audit ontstonden bij de overgang tussen lagen. Élke mutator moet expliciet declareren in welke laag hij opereert:

### Laag A — Stomp

Pure XML-attribuut writes zonder propagatie. Geen side effects.

Voorbeelden:
- `(*Cell).SetValue(v)` schrijft V-attribuut.
- `(*Cell).SetFormula(f)` schrijft F-attribuut.
- `(*Shape).SetLineStyleID(id)` schrijft een attribuut op `<Shape>`.

Contract:
- **Idempotent**: zelfde call meermaals → zelfde state.
- **Geen propagatie**: geen andere cellen / shapes geraakt.
- **Caller-verantwoordelijkheid**: consistentie tussen V/F/E moet de caller bewaken.

### Laag B — Cell + local

Setters die één semantische property zetten via `(*Shape).SetCellValue` of `SetCellFormula`. Triggeren `ensureCell`, die master-attributen kopieert behalve het te zetten attribuut.

Voorbeelden:
- `SetX`, `SetY`, `SetAngle`, `SetWidth*`, `SetHeight*`
- `SetLineWeight`, `SetBeginArrow`, `SetEndArrow`
- `SetLineColor`, `SetFillColor` (* zien §4)

Contract:
- **Lokaliseert**: na de call heeft de instance een local cell voor deze property.
- **Inherit-aware**: voor cellen waar de master een F-formule had (THEMEGUARD, Inh, GUARD), zie §4 voor de policy.
- **Side effects expliciet**: setter doc-comment vermeldt élke cell die hij raakt.

(*) `SetWidth` en `SetHeight` lijken laag B maar zitten in C — zie hieronder.

### Laag C — Cascade

Setters die meerdere cellen, geometry, of andere shapes raken. Hier ontstaan de meeste bugs omdat side effects niet visueel zijn in de naam.

Voorbeelden:
- `(*Shape).SetWidth` — raakt CellWidth + LocPinX + geometry rows + child shapes.
- `(*Shape).SetHeight` — idem op Y-axis.
- `(*Shape).SetStartAndFinish` — Begin/End/X/Y + MoveTo/LineTo rows + TxtPin + control + roept SetWidth + SetHeight.
- `(*VisioFile).ConnectShapesWithStyle` — copy connector master, remap formules, SetMasterPageID, SetStartAndFinish-cascade.
- `(*Shape).SetForeignData` — vervangt Geometry-section, set ForeignData element, set ImgWidth/Height formules.
- `(*VisioFile).RenderTemplate` — scant alle pages, mutateert in masse via SetX/Y/Width/Height/SetText.

Contract:
- **Doc-comment lijst álle side effects** met file:line referentie.
- **Order semantics**: documenteer of de cascade orde-onafhankelijk is of niet.
- **Composability**: documenteer interactie met andere cascading setters.

> **Vuistregel**: een setter die meer dan twee cellen raakt, of die in een loop andere setters aanroept, hoort in Laag C en moet contract-tests hebben voor SideEffects + Composition.

---

## 4. Cel-mutatie contract

### V vs F atomic

Per MS-VSDX §2.3.4.2.5: V wordt gebruikt **totdat een formula-evaluatie van F getriggerd wordt**. Visio re-evalueert F bij file-open en cell-dependency-changes. Vsdx-go's reader gebruikt alleen V (geen runtime formula-engine, behalve op rendering).

**Implicatie**: een cell met inconsistente V/F is geldig per spec maar gevaarlijk in praktijk — bij Visio re-open verdwijnt de V.

### `ensureCell` policy

`(*Shape).ensureCell` ([shape.go:236](shape.go)) creëert een lokale cell bij eerste mutatie en kopieert master-cell attributen behálve het te zetten attribuut. Dit is een keuze, geen bug:

- Bewust: behoudt U-attribuut (units), E-attribuut (error), en F (formula) wanneer alleen V wijzigt.
- Risico: voor cellen waarvan F een theme-binding is (THEMEGUARD, Inh), produceert dit een V/F-inconsistentie.

### Color-override policy (huidige implementatie)

`SetFillColor` en `SetLineColor` ([shape.go:580+](shape.go)) clearen F na het zetten van V. Rationale: de gebruiker zegt "deze kleur" — die verwachting moet de Visio re-open overleven.

```go
func (s *Shape) SetFillColor(v string) {
    s.SetCellValue(CellFillForegnd, v)
    s.clearCellFormula(CellFillForegnd)
}
```

Gebruikers die theme-tracking willen behouden roepen na `SetFillColor` expliciet `SetCellFormula(CellFillForegnd, "THEMEGUARD(RGB(...))")` aan.

### Uitbreidingsplan voor andere color/style setters

Per audit Agent C is dezelfde override-policy nog wenselijk voor:

| Setter | F-policy | Status |
|---|---|---|
| `SetFillColor` | clear F | ✅ gefixt |
| `SetLineColor` | clear F | ✅ gefixt |
| `SetTextColor` | clear F | ⏳ open |
| `SetLineWeight` | behoud F als Inh, anders clear | ⏳ open |
| `SetCharSize` | behoud F als Inh, anders clear | ⏳ open |
| `SetFillPattern` | clear F | ⏳ open |

---

## 5. Geometrie inheritance & localize-on-write

### Probleem

Een instance zonder lokale `<Section N="Geometry">` erft de complete geometrie van zijn master. Pre-fix deed `newShape`:

```go
s.Geometries = ms.Geometries  // pointer-alias!
```

Met als gevolg: élke mutatie via `inst.Geometries[0].XXX` muteerde master-XML, zichtbaar als data-leak naar alle sibling-instances.

### Fix: localize-on-write

[shape.go:127–154](shape.go) bouwt nu instance-owned `*Geometry` + `*GeometryRow` wrappers. Initieel verwijzen `g.xml` / `r.xml` naar master-XML (lazy share). Op de eerste mutatie wordt deep-gekloond.

Detectie:

```go
// (*Geometry).needsLocalize() returns true if g.xml lives in the master.
func (g *Geometry) needsLocalize() bool {
    return g.shape != nil && g.xml != nil && g.xml.Parent() != g.shape.xml
}
```

Mutatie-pad:

```go
// (*Geometry).localize() — deep-copies master section into instance XML
// and rebuilds Cells + Rows from the clone.
func (g *Geometry) localize() {
    if !g.needsLocalize() { return }
    cloned := g.xml.Copy()
    g.shape.xml.AddChild(cloned)
    g.xml = cloned
    // … rebuild g.Cells and g.Rows from cloned
}
```

### Verplichte localize-hooks

| Entry point | File | Hook? |
|---|---|---|
| `(*Geometry).Move` | geometry.go | ✅ |
| `(*Geometry).setRowCoords` | geometry.go | ✅ |
| `(*Geometry).addRow` | geometry.go | ✅ |
| `(*Geometry).AddArcTo` / `AddEllipse` / `AddEllipticalArcTo` / `AddRelEllipticalArcTo` / `AddRelCubBezTo` / `AddRelQuadBezTo` / `AddNURBSTo` / `AddPolylineTo` / `AddSplineStart` / `AddSplineKnot` / `AddInfiniteLine` | geometry.go | ✅ |
| `(*GeometryRow).SetX` / `SetY` | geometry.go | ✅ (via redirect-to-localized-row) |
| `scaleGeometryAxis` (intern, gebruikt door SetWidth/SetHeight) | shape.go | ✅ |

> **Verplichting**: élke nieuwe Geometry-mutator in geometry.go MOET met `g.localize()` beginnen, OF moet expliciet documenteren waarom niet.

### Row-level partial inheritance

Een shape kan een lokale `<Section N="Geometry">` hebben mét sommige rows die per IX van de master geërfd zijn (mixed-source section). `(*GeometryRow).SetX` checkt deze case via `r.geometry.needsLocalize()` en redirect naar de lokaal-equivalente row na localize. Voor `setRowCoords` worden inherited rows nog steeds silent-geskipt — dit is bewust om geen onbedoelde sectie-localization te triggeren bij iteratie.

---

## 6. Pin / LocPin / Width / Height contract

### Invariant

Voor elke shape met geldige bbox:

```
bbox.left   = Pin.X  - LocPin.X
bbox.bottom = Pin.Y  - LocPin.Y
bbox.right  = Pin.X  + Width  - LocPin.X
bbox.top    = Pin.Y  + Height - LocPin.Y   (Visio Y-up, dus top > bottom)
```

Alle mutators moeten deze invariant respecteren of expliciet documenteren waar ze afwijken.

### SetWidth / SetHeight cascade

`(*Shape).SetWidth(v)` ([shape.go:348](shape.go)) doet:

1. `SetCellValue(CellWidth, v)` — creëert lokale Width-cel.
2. Bereken `scale := v / old`.
3. `scaleGeometryAxis(s.Geometry, "X", scale)` — schaalt absolute X-cellen in alle non-Rel rows. Localiseert eerst.
4. `SetLocX(LocX() * scale)` — schaalt LocPin proportioneel.
5. Per child shape: `scaleChildShapeAxis(child, "X", scale)` — recursief op de hele subtree.

Gevolg voor de invariant: `bbox.left` blijft constant (LocPin scaleert mee met Width). Idempotent (v==old skipt).

### SetX / SetY

Stompe Laag B setters. Raken alleen `PinX` / `PinY`. Combinatie met SetWidth is order-onafhankelijk binnen Visio-inches, getest in `TestSetWidthContract_OrderWithSetX`.

### Non-spec aanname

Vsdx-go's "eager scaling" in scaleGeometryAxis schaalt alle absolute X-cellen van non-Rel rows. Per MS-VSDX hoeft dat niet — Visio's runtime ShapeSheet-evaluator zou Width-bound formules ([F="Width*0.5"]) automatisch herschalen, terwijl truly-fixed cells niet meeschalen. Vsdx-go heeft geen runtime engine, dus we kiezen voor "alles meeschalen" als pragmatisch compromis. Bekend gevolg: shapes met formule `F="0.5"` (truly fixed) worden ten onrechte mee-geschaald.

---

## 7. Connector contract (1D shapes)

### Definitie

Een shape is 1D als `s.HasBeginX()` true is (i.e. heeft een BeginX-cell met value). Spec: MS-VSDX §2.2.3.1.1.

### Move contract

`(*Shape).Move(dx, dy)` MOET voor 1D shapes BÁÁD endpoints transleren:

```go
if s.HasBeginX() {
    s.SetBeginX(s.BeginX() + dx)
    s.SetBeginY(s.BeginY() + dy)
    if s.CellValue(CellEndX) != "" {
        s.SetEndX(s.EndX() + dx)
        s.SetEndY(s.EndY() + dy)
    }
}
```

Pre-fix verplaatste alleen Begin → connector uitrekken bij Move. ✅ gefixt in [shape.go:1382](shape.go), regression: `TestMoveContract_ConnectorBothEndpointsMove`.

### Width / Height formule-coupling

Voor de meeste 1D shapes geldt:

- `CellWidth.F = "GUARD(EndX-BeginX)"`
- `CellHeight.F = "GUARD(EndY-BeginY)"`

Dus `SetWidth` op een connector zou theoretisch herrekend moeten worden via SetEndX. In de huidige implementatie zet SetWidth alleen V — en SetStartAndFinish is het canonieke pad om een connector te resizen.

---

## 8. Master isolation contract

### Statement

Elke mutator op een instance shape MOET de master XML byte-voor-byte ongewijzigd laten.

### Detectie

`AssertMasterIsolation` ([contracts_helpers_test.go](contracts_helpers_test.go)) doet:

1. Snapshot master XML hash (SHA-256 van `m.xml.WriteToBytes()`).
2. Snapshot sibling instances (W, H, LocX, LocY, FillColor, …).
3. Mutate target instance.
4. Verifieer master hash ongewijzigd.
5. Verifieer sibling snapshots ongewijzigd.

### Toegestane mutators op master

Expliciete master-mutators (via `MasterPages[i].FindShapeByID(...).Set*`) zijn legitiem en propageren bij Visio-open naar alle instances. Die zijn niet onderdeel van het isolation-contract — alleen instance-mutators.

---

## 9b. Mutation-render baseline corpus

Naast de contract-tests (§10) en de bestaande Visio-fidelity SSIM-corpus
(`vsdx-svg/`) is er een **post-mutation render-baseline**: `vsdx-svg-mutations/`.

### Doel
Vangt regressies in het pad **mutatie + render**. Contract-tests (§10) testen mutatie-correctheid op cell-niveau (V/F/master-isolation/etc.). Visio-fidelity SSIM (`vsdx-svg/`) test rendering van rust-state documenten. Geen van beide vangt "vsdx-go past de mutatie correct toe, maar rendert de resulterende staat verkeerd". Deze corpus sluit dat gat.

### Werking
`cmd/mutation-corpus-gen` houdt een Go-recipe-registry. Elke recipe:
1. Opent een vsdx-go-tests fixture als source
2. Past één gerichte mutatie toe (SetWidth, SetFillColor, Move, ...)
3. Slaat de gemuteerde VSDX op als `vsdx-svg-mutations/<recipe>.vsdx`
4. Rendert via `internal/renderpage.Render` en bevriest het resultaat als `vsdx-svg-mutations/<recipe>.svg`

De golden SVG is dus vsdx-go's **eigen output op het moment van bevriezen** — geen Visio-export. Day-1 SSIM = 1.000. Elke render-regressie op gemuteerde documenten verschijnt als SSIM-dip.

### Workflow
- **Nieuwe recipe toevoegen**: edit `cmd/mutation-corpus-gen/main.go`, run `go run ./cmd/mutation-corpus-gen` (skip bestaande, gebruik `-update` om te overschrijven).
- **Regressie-detectie**: `go run ./cmd/render-compare -input vsdx-svg-mutations -output render-compare-output-mutations`, dan `python3 compute_ssim.py`. Verwacht SSIM = 1.000.
- **Bewuste golden-update**: na een spec-conforme render-wijziging, regenereer met `-update` flag en commit de nieuwe goldens.

### Wat het wel/niet vangt
| Vraag | Vangt deze corpus? |
|---|---|
| Render-bug op gemuteerd document | ✅ |
| Render-bug op rust-state document | ❌ — daar dient `vsdx-svg/` voor |
| Mutatie-correctheid (V/F/master) | ❌ — daar dienen contract-tests voor |
| Fidelity vs. Microsoft Visio | ❌ — golden is vsdx-go's eigen output |

Voor het laatste punt: een individuele recipe kan worden ge-upgrade naar Visio-fidelity door de `.svg` te vervangen door een handmatige Visio-export van dezelfde gemuteerde staat. De rest van de pipeline blijft hetzelfde.

### Sensitivity-validatie
Verificatie tijdens framework-build: één regel `ppi := 72.0` naar `73.0` in `internal/renderpage/renderpage.go` veroorzaakt SSIM-drop van 1.000 → 0.99994. Het signaal is dus gevoelig genoeg om kleine geometry-wijzigingen op te pikken.

---

## 9. Round-trip persistence contract

### Statement

Élke `(*VisioFile).SaveVsdxBytes` MOET alle in-memory etree-mutaties serialiseren naar de ZIP-bytes.

### Implementatie

[vsdxfile.go:1204–1290](vsdxfile.go) serialiseert sequentieel:

1. `pagesXML` + `pagesXMLRels`
2. Elke `Page.xml` + `Page.RelsXML`
3. **Elke master `Page.xml` + `RelsXML`** (gefixt Fase 1)
4. `mastersXML` index (via `updateMastersXMLInZip`)
5. `rootRelsXML`, `contentTypesXML`, `appXML`, `coreXML`, `customXML`, `documentXML`, `documentXMLRels`

Pre-fix sloeg #3 + #4 over — in-memory master-mutaties werden silent gedropt bij save. Regression-tests: `TestMasterSave_*` in [master_save_test.go](master_save_test.go).

### Wat ontbreekt nog

- Geen serialisatie van per-master rels die niet in `MasterPages[i].RelsXMLFile` zitten (edge case, niet getest).
- Geen validatie dat de gegenereerde ZIP OPC-conform is na een sequentie van mutaties.

---

## 10. Contract-test coverage matrix

Per mutator: welke contracten zijn al door tests beschermd? Status na Fase 2.

Legenda: ✅ test bestaat • ⏳ open • — niet van toepassing

| Mutator | Laag | SetAndRead | SideEffects | RoundTripXML | Idempotent | OrderIndep | MasterIso | Special |
|---|:---:|:---:|:---:|:---:|:---:|:---:|:---:|:---|
| `SetX` / `SetY` | B | ✅¹ | ⏳ | ⏳ | ⏳ | ✅² | ⏳ | — |
| `SetWidth` | C | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | `CreatesLocalGeometry` ✅ |
| `SetHeight` | C | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | `NegativeHeightSignPreserved` ✅ |
| `SetAngle` | B | ⏳ | ⏳ | ⏳ | ⏳ | ⏳ | ⏳ | — |
| `SetLocX` / `SetLocY` | B | ⏳ | ⏳ | ⏳ | ⏳ | ⏳ | ⏳ | — |
| `SetBeginX/Y` / `SetEndX/Y` | B | ⏳ | ⏳ | ⏳ | ⏳ | ⏳ | ⏳ | — |
| `SetFillColor` | B | ✅ | — | ✅ | ✅ | ⏳ | ✅ | `FAttributePolicy` ✅ |
| `SetLineColor` | B | ✅ | — | ✅ | ⏳ | ⏳ | ✅ | `FAttributePolicy` ✅ |
| `SetTextColor` | B | ⏳ | — | ⏳ | ⏳ | ⏳ | ⏳ | F-policy: gaat via `ensureCharacterCell` (geen F-preservatie) |
| `SetLineWeight` / `SetLinePattern` / `SetLineCap` / `SetBeginArrow` / `SetEndArrow` / `SetRounding` / `SetFillPattern` / `SetFillTransparency` / `SetFillBkgndColor` / `SetFillBkgndTransparency` | B | — | — | — | — | — | — | `EC011_*_ClearsFormula` ✅ (10 subtests + synthetic) |
| `SetText` | C | ✅ | — | ✅ | ✅ | — | ✅ | `EmptyClearsText` ✅, `EC007_PreservesFormatMarkerElements` ✅ (pin) |
| `Move` (Shape) | C | ✅ | ✅ | ✅ | ✅³ | — | ✅ | `ConnectorBothEndpoints` ✅, `Additive` ✅ |
| `(*Geometry).Move` | C | ✅ | — | — | — | — | ✅⁴ | `ArcTo`, `EllipticalArcTo`, `Ellipse`, `NURBSTo`, `InfiniteLine`, `RelLineToUnchanged`, `MixedPathAllTranslate` — alle ✅ |
| `(*Geometry).AddLineTo` (+ varianten) | C | ⏳ | ⏳ | ⏳ | — | — | ✅⁴ | — |
| `(*GeometryRow).SetX` / `SetY` | B | ⏳ | — | ⏳ | ⏳ | ⏳ | ✅⁴ | — |
| `Remove` | C | ✅ | ✅⁷ | ✅ | — | — | — | `OrphanConnectsAreRemoved` ✅, `RemovingConnectorCleansItsConnects` ✅ |
| `Page.AddShape` | C | ⏳ | — | ⏳ | — | — | — | — |
| `Page.GroupShapes` | C | ✅ | — | ✅ | — | — | — | `BboxCenteredPins` ✅, `BboxOffCenterPin_PinsCurrentDrift` ✅ (pin), `ChildIDsPreserved` ✅, `EmptyReturnsNil` ✅ |
| `ConnectShapes` / `ConnectShapesWithStyle` | C | ✅ | ✅⁵ | ✅⁶ | — | — | — | `NoMasterCollision` ✅, `GeometryIsLineShape` ✅, `StyleAffectsGeometryNotMaster` ✅, `InvalidStyleErrors` ✅ |
| `SetStartAndFinish` | C | ✅ | — | ✅ | — | — | ✅ | `EndpointsMatch` ✅, `WidthHeightAreDeltas` ✅, `PinFollowsFormulaOrStart` ✅, `GeometryReset` ✅, `NonOneDShapeIsNoOp` ✅ |
| `SetForeignData` | C | ⏳ | ⏳ | ⏳ | — | — | — | — |
| `AddImage` | C | ⏳ | — | ⏳ | — | — | — | — |
| `AddDataProperty` | C | ⏳ | — | ⏳ | — | — | — | — |
| `LinkToData` | B | ⏳ | — | ⏳ | — | — | — | — |

Voetnoten:
1. Via bestaand `TestSetPositionAndSize` ([vsdx_test.go:1320](vsdx_test.go)) — niet expliciet als contract-test.
2. Via `TestSetWidthContract_OrderWithSetX`.
3. Via `TestMoveContract_ZeroDeltaIsNoOp` en `TestMoveContract_Additive`.
4. Via `TestMasterIsolation_GeometryRowMutationDoesNotLeakAcrossInstances` ([master_isolation_test.go](master_isolation_test.go)).
5. Via `TestConnectShapesContract_SourceShapesUnchanged`.
6. Via `TestConnectShapesContract_RoundTripPersistMaster`.
7. Via `TestRemoveContract_DoesNotAffectSiblings`.

---

## 11. Bekende mutatie-gaps

Niet-uitputtende lijst van gedocumenteerde-maar-niet-gefixte issues.

| ID | Beschrijving | Locatie | Risico | Fix-strategie |
|---|---|---|---|---|
| ~~EC-001~~ | ~~`Geometry.Move` handelt alleen MoveTo/LineTo.~~ | ~~[geometry.go:117](geometry.go)~~ | — | ✅ Gefixt via `moveCoordCells` lookup-tabel die per row-type expliciet de X-coord en Y-coord cellen opsomt. Scalars (bow, knot, weight, ratio, angle) blijven onaangetast. Relative-types worden uitgesloten. Tests in [geometry_move_contracts_test.go](geometry_move_contracts_test.go). |
| ~~EC-002~~ | ~~`scaleGeometryAxis` `strings.Contains(f, "Width")` is te grof.~~ | ~~[shape.go:431](shape.go)~~ | — | ✅ Vervangen door word-boundary regex `widthHeightTokenRE` in `scaleNonInhCell`. Tests in [ec002_token_matcher_test.go](ec002_token_matcher_test.go). |
| ~~EC-003~~ | ~~Master-mutaties propageren niet runtime naar instances met cached cells.~~ | ~~shape.go~~ | — | ✅ Twee invalidation hooks: `(*Shape).InvalidateInheritanceCaches()` voor één-shape, `(*VisioFile).InvalidateInstanceCachesForMaster(id)` voor alle vanaf AllShapes() bereikbare instances. Tests in [ec003_master_propagation_test.go](ec003_master_propagation_test.go). |
| ~~EC-004~~ | ~~`DataProperties` cache wordt nooit ge-invalideerd.~~ | ~~shape.go~~ | — | ✅ `AddDataProperty` invalideerde al `s.dataProperties = nil`. Toegevoegd: publieke `(*Shape).InvalidateDataPropertiesCache()` voor callers die XML direct muteren. Tests in [ec004_dataproperties_cache_test.go](ec004_dataproperties_cache_test.go). |
| ~~EC-005~~ | ~~Compound masters met meerdere geometry-secties hebben muteerbare gaten.~~ | ~~geometry.go~~ | — | ✅ `(*Shape).GeometryAt(idx)` exposes bounds-checked access to any section IX. Existing Geometry methods (SetMoveTo, AddLineTo, ...) reachable via `s.GeometryAt(N).XX(...)`. Tests in [ec005_geomindex_test.go](ec005_geomindex_test.go). |
| ~~EC-006~~ | ~~`ensureCharacterCell` creëert partial Character-rows.~~ | ~~[shape.go:604](shape.go)~~ | — | ✅ Gefixt in `ensureSectionCell`: bij eerste lokale Row worden alle master-cellen (uit dezelfde sectie/row) gekopieerd inclusief F-attributen, daarna wordt de gevraagde cel overschreven. Tests in [ec006_character_row_test.go](ec006_character_row_test.go). Werkt voor alle secties die ensureSectionCell gebruiken (Character, Paragraph, etc.). |
| EC-007 | `SetText` is half-destructief — leegt text-content van `<cp>`/`<pp>`/`<fld>` maar laat de elementen staan. Format-markers verliezen interleaving met de nieuwe text. **Pinned** via `TestSetTextContract_EC007_PreservesFormatMarkerElements`. Fix-strategie nog open. | [shape.go:1326](shape.go) | Medium voor format-rich text | (a) Verwijder children of (b) parse + behoud runs |
| ~~EC-008~~ | ~~`ConnectShapesWithStyle` had master-collision bug (gefixt commit b6f709b) maar mist regression-test.~~ | ~~[vsdxfile.go:828](vsdxfile.go)~~ | ~~Medium voor regressie~~ | ✅ Gefixt: `TestConnectShapesContract_NoMasterCollision` + 6 begeleidende contract-tests in [connect_shapes_contracts_test.go](connect_shapes_contracts_test.go) |
| EC-009 | Geen public `DisconnectShapes` / `RemoveConnect` als standalone API. Orphan-cleanup ZIT al in `Remove()` (zie `TestRemoveContract_OrphanConnectsAreRemoved`). Open: een explicit "verbreek deze connectie, laat de connector staan" actie. | n.v.t. | Laag | Nieuwe public API bovenop bestaande removeOrphanConnects helper |
| ~~EC-010~~ | ~~Geen `RemoveCell` / `RevertToInherited` om naar master te de-overriden.~~ | ~~n.v.t.~~ | — | ✅ Twee setters toegevoegd: `(*Shape).RemoveCell(name)` haalt local cell weg, `RevertToInherited(name)` is semantische alias. Tests in [ec010_remove_cell_test.go](ec010_remove_cell_test.go). |
| EC-010 | Geen `RemoveCell` / `RevertToInherited`. | n.v.t. | Laag-medium voor stencil-bewerking | Nieuwe public API |
| ~~EC-011~~ | ~~F-attribute clear-policy nog niet uitgebreid naar overige setters.~~ | ~~shape.go~~ | — | ✅ Gefixt voor 10 setters: `SetLineWeight`, `SetLinePattern`, `SetLineCap`, `SetBeginArrow`, `SetEndArrow`, `SetRounding`, `SetFillPattern`, `SetFillTransparency`, `SetFillBkgndColor`, `SetFillBkgndTransparency`. Tests in [style_setters_f_policy_test.go](style_setters_f_policy_test.go). `SetTextColor` / `SetCharSize` gaan via `ensureCharacterCell` en kennen het probleem niet, maar hebben EC-006 (partial Character row) als gerelateerd issue. |
| ~~EC-012~~ | ~~Geen formula-engine voor cell-dependency-cascade.~~ | ~~formula.go~~ | — | ✅ Scoped MVP: `(*Shape).RecalculateDependents(cellName)` walks cells on the same shape, finds those whose F-formula references the named cell as a word-boundary token, and re-evaluates them via existing `CalcValue`. One level deep, single shape. Does NOT handle transitive chains, cross-shape refs, or cycles — call recursively for transitive, or upgrade to a topological walker if cross-shape needed. Tests in [ec012_recalc_dependents_test.go](ec012_recalc_dependents_test.go). |
| EC-013 | `Page.GroupShapes` bbox-berekening gebruikt `Width()/2` ipv `LocX()` om stale V te omzeilen. Klopt voor centered-pin shapes; drift voor shapes met off-center pin. **Pinned** via `TestGroupShapesContract_BboxOffCenterPin_PinsCurrentDrift`. | [foreign.go:172](foreign.go) | Laag-medium voor stencil-bewerking met asymmetrische shapes | Effective-Pin helper die V/F resolve doet, dan switchen naar LocX() in bbox-loop |

---

## 12. Hoe je een nieuwe mutator toevoegt

Checklist voor PR-auteurs:

1. **Laag bepalen** (A/B/C). Doc-comment van de setter moet de laag noemen.
2. **Side effects opsommen** in de doc-comment, met file:line referenties naar elke geraakte cell / shape / property.
3. **Contract-test schrijven** in `{Mutator}_contracts_test.go`, volgens patroon van [set_width_contracts_test.go](set_width_contracts_test.go):
   - `*_SetAndRead` — basis
   - `*_SideEffects` — gebruik `snapshotShape` + `assertOnlyTheseFieldsChanged`
   - `*_RoundTripXML` — gebruik `AssertRoundTripXML`
   - `*_Idempotent` — gebruik `AssertIdempotent`
   - `*_OrderWith{OtherSetter}` — gebruik `AssertOrderIndependent` (alleen voor laag B/C)
   - `*_MasterIsolation` — gebruik `AssertMasterIsolation`
   - Specifieke edge cases (negatieve waarden, themed cells, child shapes)
4. **Localize-hook toevoegen** als de setter naar geometry schrijft. Patroon: `g.localize()` als eerste line.
5. **F-policy declareren** voor color/style setters: clear of behoud, expliciet via `clearCellFormula` of doc-comment.
6. **SSIM-baseline runnen** (`go run ./cmd/render-compare`) om geen render-regressie te krijgen. Score moet ±0.001 van de baseline blijven.
7. **Matrix bijwerken** in §10 van deze doc.
8. **Bekende gap closen** in §11 indien van toepassing.

---

## 13. Referenties

- MS-VSDX spec: [docs/MS-VSDX.pdf](../docs/MS-VSDX.pdf), specifiek §2.2.3 (Shape geometry), §2.2.5.4 (Inheritance), §2.3.4.2.5 (Cell_Type), §2.5.3.157 (THEMEGUARD).
- Audit-rapporten (Fase 1, mei 2026): conversation log + de 4 specialist-audit transcripts.
- ROADMAP overall plan: [../ROADMAP.md](../ROADMAP.md).
- RENDER_AUDIT (render-zijde divergenties): [RENDER_AUDIT.md](RENDER_AUDIT.md).
- DIVERGENCE_STATUS: [DIVERGENCE_STATUS.md](DIVERGENCE_STATUS.md).

---

*Last updated*: 2026-05-29 (alle resterende EC's afgesloten in één pass. EC-002, EC-003, EC-004, EC-005, EC-010, EC-012 gesloten met tests. Open: EC-007 (pinned), EC-013 (pinned), EC-009 (orphan-cleanup zit in Remove, standalone API blijft open). DIVERGENCE_STATUS heeft nul NEEDS_WORK items. Mutation-render corpus 8 recipes SSIM 1.000.).
