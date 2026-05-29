# Comprehensive Feature Reference

Doel: één feature-rijk Visio-document dat dient als ground truth voor vsdx-go's render-, parse- en round-trip pipeline. Door dit bestand in Visio 2021 te openen, op te slaan en als SVG te exporteren krijgen we **drie orthogonale signalen** uit één round-trip.

## Bestanden

- `comprehensive-features.vsdx` — door vsdx-go gegenereerd (170 shapes, 9 pagina's, elke shape heeft zijn feature-naam als text). **Dit is het bestand dat jij in Visio opent.**

## Wat jij doet (10-15 min in Visio)

1. Open `comprehensive-features.vsdx` in **Microsoft Visio 2021 Pro**.
2. Loop door alle 9 pagina's heen. Check globaal of het er redelijk uitziet. Sommige features die vsdx-go niet kan renderen (pattern fills, 3D effects, etc.) zullen in Visio mogelijk anders verschijnen dan in vsdx-go's eigen output — dat is precies wat we willen meten.
3. **Save as** `comprehensive-features-visio.vsdx` in deze directory (`vsdx-svg/comprehensive/`). Visio normaliseert de XML naar zijn canonical form bij het opslaan — dat is waardevol voor byte-diff analyse.
4. Voor élke pagina: **File → Export → Change File Type → SVG**. Visio exporteert één pagina per SVG, met namen zoals `comprehensive-features.svg`, `comprehensive-features-2.svg`, etc. Drop al die SVGs in deze directory.

## Pagina-overzicht

| # | Pagina | Shapes | Features |
|---|---|---|---|
| 1 | Shapes | 18 | Rect, ellipse, polygon (3-8 zijden), star (5+6 punten), diamond, multi-geom icon, group shape, block arrow, cross, lightning |
| 2 | Fills | 24 | Solid (4 kleuren), linear gradient (2/3/4 stops, verschillende hoeken), radial gradient, patterns 2/3/4/5/6/7/8/9/25/26, transparency (25/50/75%), foreground+background combinatie |
| 3 | Lines | 20 | Patterns 1-10, weights (0.25-8pt), caps (round/square/extended), line gradient, custom color |
| 4 | Arrows | 38 | Arrow types 1-24 op end-side, 6 both-end combos, sizes 0-5, curved (NURBS) arrows |
| 5 | Text | 25 | Plain, multi-line, fonts (Arial/Times/Courier), sizes (8-36pt), bold/italic/underline, color, rotation (30/45/90/135/180/270°), alignment, **rich text met cp markers** |
| 6 | Transforms | 9 | Rotation 30/45/90/135/180°, FlipX/FlipY/FlipX+Y (op L-vormige geometry voor visibility) |
| 7 | Effects | 9 | Drop shadow (default + colored), soft edges, glow, bevel (circle + cross), reflection, 3D rotation, sketch |
| 8 | Connectors | 14 | Static straight, bidirectional met label, dashed, thick colored, shape met 4 connection points, shape met ABCD directional points |
| 9 | Data | 13 | 5 custom data properties (verschillende types), external + internal hyperlinks, user cells, layer membership (A/A+B/C/none), locks (Move/Size/Delete/Rotate), comment |

## Wat ik daarna doe

Zodra Visio's `.vsdx` resave + SVG-exports binnen zijn:

1. **`cmd/feature-coverage`** (nieuw te bouwen): leest elke shape, plakt zijn text als feature-naam, render via vsdx-go, vergelijkt SVG-fragment met Visio's golden, rapporteert per feature een SSIM-score.
2. **Byte-diff** tussen `comprehensive-features.vsdx` (mijn output) en `comprehensive-features-visio.vsdx` (Visio's resave): waar wijkt mijn writer af van Visio's canonical XML?
3. **Round-trip test**: vsdx-go opent Visio's resave → render → vergelijk met Visio's SVG export. Test of mijn reader Visio's XML correct interpreteert.
4. Output: `vsdx-svg/comprehensive/COVERAGE.md` met tabel "feature → SSIM → status" + per-categorie samenvatting.

## Generator

Het `.vsdx` bestand is geproduceerd door `cmd/comprehensive-gen/main.go`. Regenereer met:
```bash
go run ./cmd/comprehensive-gen
```

De generator gebruikt vsdx-go's eigen API waar mogelijk; voor features zonder publieke API (fill gradients, rich text cp markers, shadow cells) worden cells direct geschreven volgens MS-VSDX spec.
