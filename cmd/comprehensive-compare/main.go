// comprehensive-compare drives the systematic per-theme comparison between
// Visio's SVG exports and vsdx-go's rendered output for the comprehensive
// corpus (one .vsdx with multiple themed pages, each with its own Visio SVG).
//
// Usage:
//
//	go run ./cmd/comprehensive-compare           # all pages
//	go run ./cmd/comprehensive-compare Shapes    # one page by name
//
// Output goes to render-compare-output-comprehensive/:
//   - <theme>_visio.svg     — Visio's golden export
//   - <theme>_vsdxgo.svg    — vsdx-go's rendered version
//   - compare.html          — side-by-side viewer with shape-count summary
//
// The tool also writes a short stdout report: per-page shape counts,
// renderpage errors if any, and which themes are missing a Visio SVG.
package main

import (
	"fmt"
	"html"
	"os"
	"path/filepath"
	"strings"

	"wijnberg.net/vsdx-go/internal/renderpage"
	"wijnberg.net/vsdx-go/vsdx"
)

const (
	vsdxPath  = "vsdx-svg/comprehensive/comprehensive-features.vsdx"
	visioDir  = "vsdx-svg/comprehensive"
	outputDir = "render-compare-output-comprehensive"
)

type pageResult struct {
	Name         string
	VisioPath    string // Visio's SVG, "" if missing
	VsdxgoSVG    string // rendered SVG
	VsdxgoErr    string
	VsdxgoShapes int // shape count (counting groups as 1 like Visio does)
	VisioFeats   int // number of <desc> in Visio's SVG (= features Visio acknowledges)
}

func main() {
	filter := ""
	if len(os.Args) > 1 {
		filter = os.Args[1]
	}

	if err := os.MkdirAll(outputDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "mkdir: %v\n", err)
		os.Exit(1)
	}

	v, err := vsdx.Open(vsdxPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "open vsdx: %v\n", err)
		os.Exit(1)
	}
	defer v.Close()

	var results []pageResult
	for _, p := range v.Pages {
		name := p.Name()
		if filter != "" && !strings.EqualFold(name, filter) {
			continue
		}
		r := processPage(p)
		results = append(results, r)

		if r.VsdxgoErr == "" {
			outPath := filepath.Join(outputDir, fmt.Sprintf("%s_vsdxgo.svg", strings.ToLower(name)))
			os.WriteFile(outPath, []byte(r.VsdxgoSVG), 0644)
		}
		if r.VisioPath != "" {
			data, _ := os.ReadFile(r.VisioPath)
			outPath := filepath.Join(outputDir, fmt.Sprintf("%s_visio.svg", strings.ToLower(name)))
			os.WriteFile(outPath, data, 0644)
		}
	}

	// Print per-page summary
	fmt.Println()
	fmt.Printf("%-15s %-12s %-12s %s\n", "Theme", "Visio feats", "vsdxgo top", "Status")
	fmt.Println(strings.Repeat("-", 70))
	for _, r := range results {
		status := "ok"
		if r.VisioPath == "" {
			status = "MISSING Visio SVG"
		} else if r.VsdxgoErr != "" {
			status = "RENDER ERROR: " + r.VsdxgoErr
		} else if r.VisioFeats != r.VsdxgoShapes {
			status = fmt.Sprintf("count mismatch (visio=%d vs vsdxgo=%d)", r.VisioFeats, r.VsdxgoShapes)
		}
		fmt.Printf("%-15s %-12d %-12d %s\n", r.Name, r.VisioFeats, r.VsdxgoShapes, status)
	}

	// HTML
	htmlPath := filepath.Join(outputDir, "compare.html")
	if err := writeHTML(results, htmlPath); err != nil {
		fmt.Fprintf(os.Stderr, "html: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("\nOutput: %s/\n", outputDir)
	fmt.Printf("Open in browser: file://%s\n", mustAbs(htmlPath))
}

func processPage(p *vsdx.Page) pageResult {
	r := pageResult{Name: p.Name()}

	// Locate Visio's SVG: comprehensive-features-<lowercase-name>.svg
	visioPath := filepath.Join(visioDir, fmt.Sprintf("comprehensive-features-%s.svg", strings.ToLower(p.Name())))
	if _, err := os.Stat(visioPath); err == nil {
		r.VisioPath = visioPath
		data, _ := os.ReadFile(visioPath)
		r.VisioFeats = strings.Count(string(data), "<desc>")
	}

	// Count top-level shapes (matches Visio's per-shape <desc>: one per
	// top-level group/shape, group's children are nested).
	r.VsdxgoShapes = len(p.ChildShapes())

	pageW, pageH := p.Width(), p.Height()
	svg, err := renderpage.Render(p, pageW, pageH)
	if err != nil {
		r.VsdxgoErr = err.Error()
	} else {
		r.VsdxgoSVG = svg
	}
	return r
}

func mustAbs(p string) string {
	a, err := filepath.Abs(p)
	if err != nil {
		return p
	}
	return a
}

func writeHTML(results []pageResult, path string) error {
	var sb strings.Builder
	sb.WriteString(`<!DOCTYPE html>
<html>
<head>
<meta charset="UTF-8">
<title>Comprehensive feature comparison</title>
<style>
body { font-family: -apple-system, sans-serif; margin: 0; padding: 16px; background: #f0f0f0; }
h1 { margin: 0 0 4px; }
.intro { color: #555; font-size: 13px; margin: 0 0 16px; }
.toc { background: white; border-radius: 6px; padding: 12px 16px; margin-bottom: 16px; box-shadow: 0 1px 2px rgba(0,0,0,.06); }
.toc a { margin-right: 12px; color: #0066cc; text-decoration: none; font-size: 13px; }
.theme { background: white; border-radius: 6px; padding: 16px; margin: 0 0 16px; box-shadow: 0 1px 2px rgba(0,0,0,.06); }
.theme h2 { margin: 0 0 8px; font-size: 18px; }
.meta { color: #666; font-size: 12px; margin-bottom: 12px; }
.pair { display: grid; grid-template-columns: 1fr 1fr; gap: 10px; }
.col { background: #fafafa; border: 1px solid #ddd; border-radius: 4px; padding: 8px; }
.col .h { font-weight: 600; font-size: 13px; color: #444; margin-bottom: 6px; }
.col svg, .col object { width: 100%; height: auto; max-height: 600px; display: block; background: white; }
.err { color: #c00; font-size: 12px; padding: 8px; background: #ffe0e0; border-radius: 4px; }
.warn { color: #b80; font-size: 12px; padding: 6px 10px; background: #fff4d0; border-radius: 4px; margin-bottom: 10px; }
</style>
</head>
<body>
<h1>Comprehensive feature comparison</h1>
<p class="intro">Visio's SVG export (left) vs vsdx-go's render (right), per theme page.</p>
<div class="toc">`)

	for _, r := range results {
		sb.WriteString(fmt.Sprintf(`<a href="#%s">%s</a>`, strings.ToLower(r.Name), html.EscapeString(r.Name)))
	}
	sb.WriteString(`</div>`)

	for _, r := range results {
		anchor := strings.ToLower(r.Name)
		sb.WriteString(fmt.Sprintf(`<div class="theme" id="%s">`, anchor))
		sb.WriteString(fmt.Sprintf(`<h2>%s</h2>`, html.EscapeString(r.Name)))
		sb.WriteString(fmt.Sprintf(`<div class="meta">Visio features: %d &middot; vsdx-go top-level shapes: %d</div>`, r.VisioFeats, r.VsdxgoShapes))

		if r.VisioPath == "" {
			sb.WriteString(`<div class="warn">No Visio SVG found for this theme.</div>`)
		}
		if r.VisioFeats != r.VsdxgoShapes {
			sb.WriteString(fmt.Sprintf(`<div class="warn">Shape-count mismatch: Visio sees %d, vsdx-go renders %d top-level shapes.</div>`, r.VisioFeats, r.VsdxgoShapes))
		}

		sb.WriteString(`<div class="pair">`)
		// Left: Visio's SVG via <object>
		sb.WriteString(`<div class="col"><div class="h">Visio</div>`)
		if r.VisioPath != "" {
			sb.WriteString(fmt.Sprintf(`<object type="image/svg+xml" data="%s_visio.svg"></object>`, anchor))
		} else {
			sb.WriteString(`<div class="err">no Visio SVG</div>`)
		}
		sb.WriteString(`</div>`)

		// Right: vsdx-go
		sb.WriteString(`<div class="col"><div class="h">vsdx-go</div>`)
		if r.VsdxgoErr != "" {
			sb.WriteString(fmt.Sprintf(`<div class="err">render error: %s</div>`, html.EscapeString(r.VsdxgoErr)))
		} else {
			sb.WriteString(fmt.Sprintf(`<object type="image/svg+xml" data="%s_vsdxgo.svg"></object>`, anchor))
		}
		sb.WriteString(`</div>`)

		sb.WriteString(`</div></div>`)
	}

	sb.WriteString(`</body></html>`)
	return os.WriteFile(path, []byte(sb.String()), 0644)
}
