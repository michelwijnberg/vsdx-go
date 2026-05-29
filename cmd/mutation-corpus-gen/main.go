// mutation-corpus-gen generates a set of post-mutation render fixtures.
//
// For each recipe in the registry below:
//  1. Open a source VSDX
//  2. Apply a programmed mutation (SetWidth, Move, SetFillColor, ...)
//  3. Save the mutated VSDX to vsdx-svg-mutations/<recipe>.vsdx
//  4. Render that mutated VSDX with the shared renderpage.Render and freeze
//     the result as vsdx-svg-mutations/<recipe>.svg
//
// The "golden" written here is vsdx-go's OWN render at this moment — not a
// Visio export. That makes the corpus a pure regression baseline: SSIM 1.0
// on first commit, dips on any future change to the mutation+render path.
//
// To upgrade a fixture to "fidelity-vs-Visio" later, replace its .svg with
// a hand-exported Visio SVG of the same mutated state. The render-compare
// tool doesn't care where the golden comes from.
package main

import (
	"flag"
	"fmt"
	"math"
	"os"
	"path/filepath"

	"wijnberg.net/vsdx-go/internal/renderpage"
	"wijnberg.net/vsdx-go/vsdx"
)

// recipe describes a single mutation case. The mutate function receives the
// open VisioFile and is expected to mutate in place. The find function
// returns the source VSDX path relative to the repo root.
type recipe struct {
	name     string
	source   string
	mutate   func(*vsdx.VisioFile) error
	pageIdx  int
}

// registry is the active set of mutation fixtures. Each entry exercises ONE
// primary mutation so a baseline SSIM dip points at a specific code path.
// All source paths resolve from the vsdx-go repo root so the corpus is
// self-contained.
var registry = []recipe{
	{
		name:    "setwidth-shape-doubled",
		source:  "tests/test_master.vsdx",
		pageIdx: 0,
		mutate: func(v *vsdx.VisioFile) error {
			// Pick the first shape on the page with a master and a finite
			// width. Doubling Width exercises the geometry + LocPin +
			// child-shape scaling cascade.
			s := firstShapeWithMaster(v)
			if s == nil {
				return fmt.Errorf("no shape-with-master found")
			}
			s.SetWidth(s.Width() * 2.0)
			return nil
		},
	},
	{
		name:    "setfillcolor-shape-red",
		source:  "tests/test12_colors.vsdx",
		pageIdx: 0,
		mutate: func(v *vsdx.VisioFile) error {
			s := firstShapeWithFill(v)
			if s == nil {
				return fmt.Errorf("no shape with fill found")
			}
			s.SetFillColor("#cc0000")
			return nil
		},
	},
	{
		name:    "move-connector-shifted",
		source:  "tests/test4_connectors.vsdx",
		pageIdx: 0,
		mutate: func(v *vsdx.VisioFile) error {
			// Move shifts BOTH BeginX/Y and EndX/Y for connectors — Phase 2
			// fix. Catches regression if either endpoint stops following.
			for _, s := range v.Pages[0].AllShapes() {
				if s.HasBeginX() {
					s.Move(0.5, 0.5)
					return nil
				}
			}
			return fmt.Errorf("no 1D shape with BeginX in fixture")
		},
	},
	{
		// DIVERGENCE #35: FlipX. Before the render-side fix this recipe
		// produced an SVG identical to the un-flipped one — silently wrong.
		// With the fix in place the rendered group carries
		// `scale(-1, 1)` around the right edge, mirroring the geometry.
		name:    "flipx-shape-mirrored",
		source:  "tests/test_master.vsdx",
		pageIdx: 0,
		mutate: func(v *vsdx.VisioFile) error {
			s := firstShapeWithMaster(v)
			if s == nil {
				return fmt.Errorf("no shape-with-master found")
			}
			s.SetFlipX(true)
			return nil
		},
	},
	{
		// DIVERGENCE #35: FlipY counterpart.
		name:    "flipy-shape-mirrored",
		source:  "tests/test_master.vsdx",
		pageIdx: 0,
		mutate: func(v *vsdx.VisioFile) error {
			s := firstShapeWithMaster(v)
			if s == nil {
				return fmt.Errorf("no shape-with-master found")
			}
			s.SetFlipY(true)
			return nil
		},
	},
	{
		// DIVERGENCE #31: vertical text via TxtAngle = π/2 (90°). vsdx-go
		// already emits a rotate() transform on the <text> element when
		// TxtAngle != 0 (render_tree.go:733). This recipe locks the
		// behaviour as a baseline.
		name:    "txtangle-vertical-90deg",
		source:  "tests/test_master.vsdx",
		pageIdx: 0,
		mutate: func(v *vsdx.VisioFile) error {
			s := firstShapeWithMaster(v)
			if s == nil {
				return fmt.Errorf("no shape-with-master found")
			}
			s.SetTxtAngle(math.Pi / 2)
			return nil
		},
	},
	{
		// DIVERGENCE #34: arrow setback on straight + curved connectors.
		// shortenPathStart / shortenPathEnd handle both 'L' (straight) and
		// 'C' (cubic Bezier) via de Casteljau subdivision. NURBSTo rows
		// get emitted as cubic Beziers, so this covers the curve case too.
		// vsdx-go's tests/ fixtures only carry straight connectors; this
		// recipe pins the L-path. The C-path is exercised indirectly via
		// vsdx-svg/logical-architecture.vsdx (in the static SSIM corpus).
		name:    "arrow-on-connector",
		source:  "tests/test4_connectors.vsdx",
		pageIdx: 0,
		mutate: func(v *vsdx.VisioFile) error {
			for _, s := range v.Pages[0].AllShapes() {
				if s.HasBeginX() {
					s.SetEndArrow(13)
					s.SetBeginArrow(13)
					return nil
				}
			}
			return fmt.Errorf("no 1D shape with BeginX in fixture")
		},
	},
}

func firstShapeWithMaster(v *vsdx.VisioFile) *vsdx.Shape {
	for _, p := range v.Pages {
		for _, s := range p.AllShapes() {
			if s.MasterPageID != "" && s.Width() > 0 {
				return s
			}
		}
	}
	return nil
}

func firstShapeWithFill(v *vsdx.VisioFile) *vsdx.Shape {
	for _, p := range v.Pages {
		for _, s := range p.AllShapes() {
			if s.FillColor() != "" {
				return s
			}
		}
	}
	return nil
}

func findFirstByName(v *vsdx.VisioFile, name string) *vsdx.Shape {
	for _, p := range v.Pages {
		for _, s := range p.AllShapes() {
			if s.ShapeName == name {
				return s
			}
		}
	}
	return nil
}

func main() {
	outputDir := flag.String("output", "vsdx-svg-mutations", "directory to write mutation fixtures")
	repoRoot := flag.String("repo", ".", "vsdx-go repo root (sources resolved from here)")
	update := flag.Bool("update", false, "overwrite existing fixtures; without this flag, existing fixtures are skipped")
	flag.Parse()

	if err := os.MkdirAll(*outputDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "mkdir output: %v\n", err)
		os.Exit(1)
	}

	for _, r := range registry {
		outVSDX := filepath.Join(*outputDir, r.name+".vsdx")
		outSVG := filepath.Join(*outputDir, r.name+".svg")
		if !*update {
			if _, err := os.Stat(outVSDX); err == nil {
				fmt.Printf("Skipping %s (exists; use -update to overwrite)\n", r.name)
				continue
			}
		}

		src := filepath.Join(*repoRoot, r.source)
		data, err := os.ReadFile(src)
		if err != nil {
			// Try inside the repo's tests directory as a fallback.
			alt := filepath.Join(*repoRoot, "tests", filepath.Base(r.source))
			data, err = os.ReadFile(alt)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Source %s not found (tried %s and %s)\n", r.source, src, alt)
				continue
			}
		}

		v, err := vsdx.OpenBytes(data)
		if err != nil {
			fmt.Fprintf(os.Stderr, "open %s: %v\n", r.name, err)
			continue
		}

		if err := r.mutate(v); err != nil {
			fmt.Fprintf(os.Stderr, "mutate %s: %v\n", r.name, err)
			v.Close()
			continue
		}

		// Render BEFORE saving so we capture the in-memory mutated state
		// (Save+reopen would also work but adds a parse round-trip).
		if r.pageIdx >= len(v.Pages) {
			fmt.Fprintf(os.Stderr, "%s: page index %d out of range\n", r.name, r.pageIdx)
			v.Close()
			continue
		}
		page := v.Pages[r.pageIdx]
		pageW := page.Width()
		pageH := page.Height()
		if pageW == 0 {
			pageW = 800
		}
		if pageH == 0 {
			pageH = 600
		}
		svg, err := renderpage.Render(page, pageW, pageH)
		if err != nil {
			fmt.Fprintf(os.Stderr, "render %s: %v\n", r.name, err)
			v.Close()
			continue
		}

		// Save the mutated VSDX so render-compare can re-open it and
		// regenerate the same SVG.
		out, err := v.SaveVsdxBytes()
		if err != nil {
			fmt.Fprintf(os.Stderr, "save %s: %v\n", r.name, err)
			v.Close()
			continue
		}
		v.Close()

		if err := os.WriteFile(outVSDX, out, 0644); err != nil {
			fmt.Fprintf(os.Stderr, "write vsdx %s: %v\n", r.name, err)
			continue
		}
		if err := os.WriteFile(outSVG, []byte(svg), 0644); err != nil {
			fmt.Fprintf(os.Stderr, "write svg %s: %v\n", r.name, err)
			continue
		}
		fmt.Printf("Generated %s (vsdx + golden svg)\n", r.name)
	}
}
