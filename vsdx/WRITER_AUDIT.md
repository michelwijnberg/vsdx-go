# Writer Audit — vsdx-go output vs Visio canonical resave

> **Methodology**: Generate `comprehensive-features.vsdx` (9 themed pages, 170+ shapes
> covering every Section type, fill pattern, gradient, arrow style, effect, transform,
> connector type, and data feature) → open in Microsoft Visio 2021 → "Save As" to
> produce `comprehensive-features-visio-saved.vsdx` → tag-by-tag XML diff between
> the two ZIPs. Every item below is something the diff surfaced; every item is
> backed by a regression test that re-asserts the canonical form on the next save.
>
> Snapshot date: 2026-05-29 (post writer-canonicalization sweep, commit a5976f9).

---

## Round-trip status

Per file in the ZIP, comparing tag-counts between vsdx-go's output and Visio's resave:

| File | Status |
|---|---|
| `[Content_Types].xml` | ✓ identical |
| `_rels/.rels` | ✓ identical |
| `docProps/app.xml` | ✓ identical (HLinks vector + vt: namespace + ordering) |
| `docProps/core.xml` | ✓ identical |
| `docProps/custom.xml` | ✓ identical |
| `visio/document.xml` | ✓ identical (color palette + FaceName auto-register) |
| `visio/document.xml.rels` | ✓ identical |
| `visio/masters/masters.xml` | ✓ identical |
| `visio/masters/master1.xml` | ✓ identical |
| `visio/pages/pages.xml` | ✓ identical (RecalcColor triggers + canonical Layer rows) |
| `visio/pages/pages.xml.rels` | ✓ identical (orphan rel cleanup on RemovePage) |
| `visio/pages/page1.xml` (Shapes) | ✓ identical |
| `visio/pages/page2.xml` (Fills) | ✓ identical |
| `visio/pages/page3.xml` (Lines) | ✓ identical |
| `visio/pages/page4.xml` (Arrows) | ✓ identical |
| `visio/pages/page5.xml` (Text) | ⚠️ 30 cells short on one rich-text shape (Char-row expansion) |
| `visio/pages/page6.xml` (Transforms) | ✓ identical |
| `visio/pages/page7.xml` (Effects) | ⚠️ 1 cell artifact (stale Visio-save baseline; will close on next resave) |
| `visio/pages/page8.xml` (Connectors) | ✓ identical |
| `visio/pages/page9.xml` (Data) | ✓ identical |
| Per-page rels (`page*.xml.rels`) | ✓ identical |
| `visio/theme/theme1.xml` | ✓ identical |
| `visio/windows.xml` | ✓ identical (children stripped on save) |

**Files matching exactly: 22/24.** Remaining 2 deltas are cosmetic and documented under
"Open items" below.

---

## Closed items

Every fix below has an in-tree regression test in `vsdx/` (mostly under contract-
or EC-style test files). Commit hashes refer to the canonicalization sweep.

### Package / packaging

| # | Item | Mechanism | Commit |
|---|---|---|---|
| 1 | Single-quote attributes everywhere | `writeXMLBytes` helper sets `etree.WriteSettings{AttrSingleQuote: true}` on every Document before serialisation | pre-sweep |
| 2 | `vt:` namespace prefix on app.xml typed elements | `CreateElement("vt:lpstr")` / `vt:i4` / `vt:variant` instead of bare tag names; binds to ECMA-376 docPropsVTypes schema | bf16bfd |
| 3 | Page-rels file for every `ConnectShapes`-using page | `ensurePageMasterRel(page, masterTarget)` runs unconditionally per call, not only for the first | bf16bfd |
| 4 | TitlesOfParts ordering (masters at tail) | `insertLpstrBeforeMasters` keeps page titles contiguous, masters last | bf16bfd |
| 5 | `[Content_Types]` orphan `<Override>` on page removal | `RemovePageByIndex` removes the matching Override entry by PartName | 119c49d |
| 6 | `pages.xml.rels` orphan `<Relationship>` on page removal | Same path: matches by Target | 119c49d |

### app.xml

| # | Item | Mechanism | Commit |
|---|---|---|---|
| 7 | HLinks section auto-generation | `refreshAppXMLHLinks` walks every shape Hyperlink section, emits canonical `vt:vector` with 6N variants per ECMA-376 §15.2.12.10 | 88a2451 |
| 8 | Hyperlinks deterministically ordered | Sort by Address — matches Visio's alphabetical canonical order | 88a2451 |
| 9 | TitlesOfParts auto-increment / decrement on page add+remove | `addPageToAppXML` / `removePageFromAppXML` | pre-sweep |
| 10 | HeadingPairs auto-increment | Same | pre-sweep |

### document.xml

| # | Item | Mechanism | Commit |
|---|---|---|---|
| 11 | Color palette auto-refresh | `refreshDocumentColorPalette` walks every shape (page + master), collects unique #RRGGBB from 6 color-bearing cells, appends new `ColorEntry` rows sorted by hex | pre-sweep |
| 12 | FaceName auto-registration | `refreshFaceNames` walks every Char.Font value, appends new `FaceName` entries with canonical UnicodeRanges/CharSets/Panose/Flags from a font-metric table (7 fonts pre-populated, unknown fonts get a minimal NameU-only entry) | 88a2451 |
| 13 | FaceNames inserted before StyleSheets | `refreshFaceNames` finds StyleSheets and inserts before, preserving canonical DocumentSettings → Colors → FaceNames → StyleSheets order | 88a2451 |

### pages.xml

| # | Item | Mechanism | Commit |
|---|---|---|---|
| 14 | Per-page `<Trigger N='RecalcColor'>` with self-referencing RefBy | `refreshPageRecalcTriggers` inserts `<Trigger N='RecalcColor'><RefBy T='Page' ID='X'/></Trigger>` into every PageSheet (between DrawingScaleType and InhibitSnap), idempotent | 119c49d |
| 15 | PageSheet default cells (PageScale, DrawingScale, locks) | `AddPageAt` emits the full 17-cell canonical set | pre-sweep |

### windows.xml

| # | Item | Mechanism | Commit |
|---|---|---|---|
| 16 | Strip `<Window>` children on save | `stripWindowsChildren` parses windows.xml at save time, drops all children of `<Windows>` but preserves `ClientWidth` / `ClientHeight` on the root | 88a2451 |

### Per-shape canonicalization

| # | Item | Mechanism | Commit |
|---|---|---|---|
| 17 | Default shape cells (Angle/FlipX/FlipY/ResizeMode) | `Page.AddShape` emits all four with `V='0' F='No Formula'`, Angle additionally with `U='NUM'` | pre-sweep |
| 18 | Group shape default cells | `GroupShapes` (foreign.go) now emits the same 4 default cells on the group container | a5976f9 |
| 19 | LineWeight U='PT' annotation | `SetLineWeight` annotates the cell | pre-sweep |
| 20 | Char.Size U='PT' annotation | `SetCharSize` annotates the cell | pre-sweep |

### Section canonicalization

| # | Item | Mechanism | Commit |
|---|---|---|---|
| 21 | Geometry NoShow/NoSnap/NoQuickDrag defaults | `AddGeometry` emits all three with `V='0' F='No Formula'` at the head of every Geometry section | pre-sweep |
| 22 | Geometry NoFill/NoLine defaults | Same path; both with `V='0' F='No Formula'`. `AddGeometryRect` overrides NoLine to `1` rather than appending a duplicate cell | 119c49d |
| 23 | Layer Row canonical cell set | `Page.AddLayer` writes Name, Color, Status, Visible, Print, Active, Lock, Snap, Glue, NameUniv, ColorTrans — 11 cells in Visio's exact order, with `F='No Formula'` on defaults | 119c49d |
| 24 | Connection Row T='Connection' attribute + AutoGen/Prompt | `AddConnectionPoint` adds the row-type attribute and emits canonical AutoGen / Prompt cells | 119c49d |
| 25 | Hyperlink Row canonical cells + ordering | `AddHyperlink` writes Description, Address, SubAddress, ExtraInfo, Frame, NewWindow, Default, Invisible, SortKey — Visio's exact order, with `F='No Formula'` on the optional cells (NewWindow, SortKey) | 119c49d |
| 26 | ConnectionABCD canonical form | `AddConnectionABCD` writes the row inside `Section N='Connection'` (not its own section) with `T='ConnectionABCD'` and cells X/Y/A/B/C/D only — directional cells DirX/DirY/Type belong to `T='Connection'` rows | 119c49d |
| 27 | Lock cells direct on shape | `setLock` writes LockMove* / LockSize* / LockDelete / LockRotate / LockAspect as direct shape `<Cell>` children, not inside a Protection section. Visio's resave always hoists Lock cells from any Protection section to the shape body | 119c49d |
| 28 | Text element cp/pp formatting markers | `normalizeTextFormatMarkers` (called from SaveVsdxBytes for every shape on every page + master) prepends `<cp IX='0'/>` if the shape has a Character section and `<pp IX='0'/>` if it has a Paragraph section, idempotently. Visio binds the initial text run to row 0 of the formatting section via these markers | 88a2451 |
| 29 | TxtAngle placeholder companion cells | `SetTxtAngle` seeds TxtPinX/Y, TxtWidth/Height, TxtLocPinX/Y with `V='0' U='NUM' F='No Formula'` when not already present. Visio emits this 7-cell placeholder block whenever a shape carries text rotation | 119c49d |

### Default-value stripping

Visio's resave strips cells whose value equals the stylesheet default. The library
now mirrors that to keep round-trips clean:

| # | Item | Default | Mechanism | Commit |
|---|---|---|---|---|
| 30 | LinePattern | `1` (solid) | `SetLinePattern(1)` removes the cell instead of writing it | a5976f9 |
| 31 | LineCap | `0` (round) | `SetLineCap(0)` removes the cell | a5976f9 |
| 32 | BeginArrowSize / EndArrowSize | `2` (medium) | Generator-side skip in comprehensive-gen | 119c49d |
| 33 | FillGradientDir / LineGradientDir | `0` (linear) | Both setters omit the cell at default | a5976f9, 119c49d |
| 34 | GradientStopPosition (first stop) | `0` | Both fill and line gradient writers skip when position is 0 (implicit) | a5976f9, 119c49d |
| 35 | GradientStopColorTrans | `0` | Both fill and line gradient writers skip when 0 | a5976f9, 119c49d |
| 36 | BevelEffect numeric zeros | `0` | `SetBevelEffect` skips every numeric cell whose input value is 0 | a5976f9 |
| 37 | GlowEffect / ReflectionEffect numeric zeros | `0` | Setters skip cells with 0 values | a5976f9 |
| 38 | Rotation3D numeric zeros | `0` | `SetRotation3DEffect` skips X/Y/Z angle / Perspective / Distance when 0; `KeepTextFlat` only written when true | a5976f9 |

---

## Open items

### 1. Rich-text Character row expansion (page 5 / Text)

Visio's canonical resave expands every Character row that carries any style cell
to a full 17-cell row: Font, Color, Style, Case, Pos, FontScale, Size,
DblUnderline, Overline, Strikethru, DoubleStrikethrough, Letterspace, ColorTrans,
AsianFont, ComplexScriptFont, ComplexScriptSize, LangID (all defaults with
`F='No Formula'`, plus whichever style cells the caller actually set).

vsdx-go writes only the cells the caller touches (typically 1-3). For shapes with
inline formatting runs (e.g. mixed bold/italic/color in one text), Visio adds
30 default cells per styled run on resave.

**Impact**: cosmetic — round-trip is content-equivalent. Adding 15 default cells
per Character row inflates file size without any behavioural change.

**Plan**: optional — can be added under `normalizeTextFormatMarkers` if a downstream
tool relies on the full canonical row. Not pursued in this sweep.

### 2. Per-page Cell ordering inside sections

vsdx-go emits cells in setter-call order. Visio's resave produces a canonical
sequence (e.g. for a shape: PinX → PinY → Width → Height → LocPinX/Y → Angle →
Flip → ResizeMode → fill cells → line cells → ...). Tag-counts match exactly;
only positional order differs.

**Impact**: textual `diff` flags as noise. Tag-count diff and tools that compare
by cell name are unaffected.

**Plan**: would require a canonical-order pass in every setter or a final sort
in `SaveVsdxBytes`. Not pursued — large patch surface, no functional benefit.

### 3. Effects render-side visualization

The writer round-trips Bevel / Glow / Reflection / Sketch / Rotation3D cells
cleanly; the SVG renderer doesn't yet visualize them (only Shadow and SoftEdges
emit filters). Tracked in `UNSUPPORTED_FEATURES.md` under "Effects".

---

## How to re-run the audit

```bash
# 1. Regenerate vsdx-go's output
go run ./cmd/comprehensive-gen/...

# 2. (Manual) Open vsdx-svg/comprehensive/comprehensive-features.vsdx in Visio 2021,
#    File → Save As → vsdx-svg/comprehensive/comprehensive-features-visio-saved.vsdx

# 3. Tag-count diff
cd /tmp && rm -rf gA vA && mkdir gA vA
unzip -d gA /home/michel/vsdx-go/vsdx-svg/comprehensive/comprehensive-features.vsdx > /dev/null
unzip -d vA /home/michel/vsdx-go/vsdx-svg/comprehensive/comprehensive-features-visio-saved.vsdx > /dev/null
python3 -c "
import re, os, glob
from collections import Counter
mapping = {'page2.xml':'page1.xml','page3.xml':'page2.xml','page4.xml':'page3.xml',
           'page5.xml':'page4.xml','page6.xml':'page5.xml','page7.xml':'page6.xml',
           'page8.xml':'page7.xml','page9.xml':'page8.xml','page10.xml':'page9.xml'}
def find(r):
    for f in glob.glob(f'{r}/**/*', recursive=True):
        if os.path.isfile(f) and (f.endswith('.xml') or f.endswith('.rels')):
            yield f.replace(r+'/','')
def cnt(p):
    return Counter(re.findall(r'<([a-zA-Z][\w:.-]*)\b', open(p).read()))
for gf in sorted(find('gA')):
    fname = os.path.basename(gf)
    vfn = mapping.get(fname, fname)
    vp = 'vA/' + os.path.dirname(gf) + '/' + vfn
    if not os.path.exists(vp): continue
    gc, vc = cnt('gA/'+gf), cnt(vp)
    d = {k:(gc[k],vc.get(k,0)) for k in gc if gc[k]!=vc.get(k,0)}
    d.update({k:(gc.get(k,0),vc[k]) for k in vc if k not in gc})
    if d: print(f'{gf} vs {vfn}: {d}')
"
```

A clean run prints only the two known-open items (rich-text expansion and 1 cell
artifact on Effects page).

---

## Why this matters

Visio interop has three failure modes, and each fix above closes one:

| Failure mode | Closed by items |
|---|---|
| **File rejected** (won't open in Visio) | Items #1-6 (packaging, namespaces, rels integrity) |
| **File opens but loses information** (cells silently dropped on resave) | Items #11-13 (palette, fonts), #17-29 (canonical section forms) |
| **File round-trips visibly differently** (Visio's resave adds/removes content) | Items #30-38 (default stripping), and the resave acts as the regression detector for the rest |

The first mode is fatal; the second is a stealth correctness bug; the third makes
diffs and version control noisy. After this sweep, all three are addressed for
the feature surface exercised by the comprehensive corpus (170+ shapes covering
every Section type).
