# Writer Audit — vsdx-go output vs Visio canonical resave

> Comparison: `vsdx-svg/comprehensive/comprehensive-features.vsdx` (vsdx-go's output, 9 themed pages, 170 shapes) byte-diffed against `vsdx-svg/comprehensive/comprehensive-features-visio-saved.vsdx` (Visio 2021's resave of the same file).
>
> **Bottom line**: Visio successfully opens and resaves every vsdx-go-generated file. None of the deltas below break Visio interop — they're all writer-style differences (cell defaults Visio explicitly writes, normalisations on resave, etc.). They DO matter when:
> 1. Diffing two .vsdx files for content changes (our omissions look like content)
> 2. Re-opening in older Visio versions (some defaults may be required)
> 3. Producing a "round-trip clean" file where no Visio resave changes anything
>
> Snapshot date: 2026-05-29.

---

## Summary by category

| # | Area | vsdx-go | Visio canonical | Severity |
|---|---|---|---|---|
| 1 | Attribute quote style | `"` (double) | `'` (single) | Cosmetic |
| 2 | Shape default cells | Omitted | Explicit `V='0' F='No Formula'` | Low (resave adds) |
| 3 | Geometry section defaults | Only NoFill/NoLine | + NoShow, NoSnap, NoQuickDrag | Low (resave adds) |
| 4 | Unit attribute (U) | Omitted on numerics | `U='NUM'`, `U='PT'`, `U='MM'` | Low (resave normalises) |
| 5 | PageSheet defaults | Omitted | + PageScale, DrawingScale, PageLockReplace, PageLockDuplicate | Low |
| 6 | Document color palette | Static, no shape colors added | ColorEntry rows for every unique RGB | Medium — palette grows on resave |
| 7 | Cell ordering | Insertion order | Visio's canonical order (Pin/Loc/Width/H/Angle/FlipX/Y/Resize…) | Cosmetic |
| 8 | Page file renumbering | `page2…page10.xml` (with leading blank removed leaves gap) | Sequential `page1…page9.xml` | Low |
| 9 | windows.xml | Inherits source state | Empties to `<Windows .../>` | None — Visio strips on save |
| 10 | DocumentSettings TopPage | `0` | Last-viewed page index | None |

---

## §1. Quote style

vsdx-go writer (via etree):
```xml
<Cell N="PinX" V="1.4"/>
<Section N="Geometry" IX="0">
```

Visio canonical:
```xml
<Cell N='PinX' V='1.4'/>
<Section N='Geometry' IX='0'>
```

Both well-formed per XML 1.0. Visio's resave normalises everything to single-quote — purely cosmetic.

## §2. Shape default cells

Every Visio-written shape carries 7 cells that vsdx-go never emits because they're at their default value (0):

```xml
<Cell N='Angle' V='0' U='NUM' F='No Formula'/>
<Cell N='FlipX' V='0' F='No Formula'/>
<Cell N='FlipY' V='0' F='No Formula'/>
<Cell N='ResizeMode' V='0' F='No Formula'/>
<Cell N='NoShow' V='0' F='No Formula'/>   (shape-level)
<Cell N='NoSnap' V='0' F='No Formula'/>
<Cell N='NoQuickDrag' V='0' F='No Formula'/>
```

The `F='No Formula'` literal is Visio's way of distinguishing "explicitly set to 0" from "inherited/derived". For files that downstream tools diff cell-by-cell, our omission can look like missing data; for vsdx-go itself the inheritance fallback returns 0 either way.

## §3. Geometry section defaults

vsdx-go's `AddGeometry()` and `AddGeometryRect()` emit only the cells they explicitly set (NoFill, NoLine). Visio writes the full default set inside each Geometry section:

```xml
<Section N='Geometry' IX='0'>
  <Cell N='NoFill' V='0' F='No Formula'/>
  <Cell N='NoLine' V='0' F='No Formula'/>
  <Cell N='NoShow' V='0' F='No Formula'/>
  <Cell N='NoSnap' V='0' F='No Formula'/>
  <Cell N='NoQuickDrag' V='0' F='No Formula'/>
  …
```

Same effect on Visio. The `F='No Formula'` annotation is the noticeable difference.

## §4. Unit attribute

Visio annotates numeric cells with their unit:

```xml
<Cell N='LineWeight' V='0.04166666666666666' U='PT'/>
<Cell N='Angle' V='0' U='NUM' F='No Formula'/>
<Cell N='Height' V='-1.181102362204724' U='MM' F='GUARD(EndY-BeginY)'/>
```

vsdx-go currently writes only `V` (no `U`). Visio assumes inches (`IN`) when `U` is absent — which is correct for our values. The unit is purely descriptive on resave but matters if a user opens the shape's Properties → Geometry pane and expects "0.04 pt" instead of "0.04 in".

## §5. PageSheet defaults

Per page, Visio writes 4 cells we omit:

```xml
<Cell N='PageScale' V='1' U='IN'/>
<Cell N='DrawingScale' V='1' U='IN'/>
<Cell N='PageLockReplace' V='0' F='No Formula'/>
<Cell N='PageLockDuplicate' V='0' F='No Formula'/>
```

PageScale / DrawingScale = 1 means 1:1 (no scale). PageLockReplace / PageLockDuplicate = 0 means unlocked. All defaults, but Visio writes them explicitly.

## §6. Document color palette

This is the BIGGEST writer-side delta. vsdx-go ships the same 25-entry palette regardless of which colors the shapes use. Visio's resave appends a `ColorEntry` row for every unique color found in any shape's FillForegnd / LineColor / etc.:

```xml
<!-- vsdx-go: 25 ColorEntry rows, palette ends at IX="24" -->
<ColorEntry IX="24" RGB="#7F7F7F"/>
</Colors>

<!-- Visio resave: appended -->
<ColorEntry IX='25' RGB='#F0F4FF'/>
<ColorEntry IX='26' RGB='#333333'/>
<ColorEntry IX='27' RGB='#CFE2FF'/>
…
```

Functional impact: minimal — shapes reference colors by `#RRGGBB` literal, not by palette index. But:
- Visio's color picker UI shows recently-used colors from the palette
- A scan tool looking at the palette to enumerate "what colors are in this document" would only see the static 25 entries from our output

Suggested fix: on save, scan all shape cells for `#RRGGBB` values and append unseen ones to the document palette.

## §7. Cell ordering

vsdx-go writes cells in insertion order (whatever order setter calls happened). Visio writes a canonical order. For shape-rectangle:

vsdx-go order (per shape.go's setter calls):
```
PinX PinY Width Height LocPinX LocPinY FillForegnd LineColor LineWeight
```

Visio canonical order:
```
PinX PinY Width Height LocPinX LocPinY Angle FlipX FlipY ResizeMode
FillForegnd LineWeight LineColor   (note swap with vsdx-go's LineWeight/LineColor)
```

Both work. A `vsdx-diff` tool that compares sorted by name would not notice. A naïve textual diff sees this as noise.

## §8. Page file renumbering

vsdx-go's generator removes the blank default page via `RemovePageByIndex(0)` after adding 9 new pages. The resulting filenames in the ZIP are `page2.xml` through `page10.xml` (page1 is gone but the others keep their original numbers). Visio's resave renumbers to `page1.xml` through `page9.xml`. Pages.xml internal references are correct in both.

vsdx-go's behaviour is intentional — renumbering on remove would require rewriting every rel file. Easier left as-is. Visio cleans it up on resave.

## §9. windows.xml

vsdx-go preserves the `Windows` element from the source `blank.vsdx`, including ClientWidth, ClientHeight, the active Window's geometry, ShowRulers/ShowGrid/ShowGuides settings, SnapSettings, etc. Visio's resave strips the entire body and emits an empty `<Windows .../>`. Visio re-adds window state on next open as needed.

## §10. DocumentSettings TopPage

`TopPage` records the index of the page that's currently active. vsdx-go writes `TopPage='0'` (first page); Visio's resave writes whichever page the user was viewing when they saved. Pure UI-state.

---

## What's actionable

If we want byte-clean round-trip (open in Visio, save, no diff):

1. **Single-quote attribute style** — low effort, ~1 day to thread through etree's output.
2. **Default cells on shape and geometry** — write the 7 shape defaults and 3 geometry defaults explicitly with `F='No Formula'`. Medium effort.
3. **U attribute on numerics** — add unit-aware setters (`SetLineWeightPt`, `SetAngleRad` → write `U='PT'` / `U='NUM'`). Medium effort, requires per-cell unit knowledge.
4. **Color palette extension on save** — scan shape cells, append unseen RGBs to `<Colors>` block. Low effort.
5. **PageSheet defaults** — write 4 default cells. Trivial.

None of these block Visio compatibility — Visio opens our files and resaves them with all the additions. They're polish for tools that diff or audit .vsdx files at byte / structural level.

---

## What's NOT a writer issue

These differences are NOT in the writer's control:
- Page file renumbering on Visio resave (Visio's choice, not ours)
- windows.xml strip-and-rewrite (Visio's choice)
- TopPage update (Visio tracks UI state)
- ColorEntry additions for runtime palette (Visio refreshes on edit)
