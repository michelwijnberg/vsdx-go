// batch-fixture-gen builds ONE comprehensive VSDX that exercises every
// mutator we've fixed in Phase 1 + Phase 2 + every closed EC-ID. Each
// mutation gets its own labelled cell on a grid layout. The user opens the
// resulting file in Microsoft Visio, exports a single SVG, and we can diff
// every mutation against Visio's reference in one pass.
//
// Grid layout (Visio Y-up, page 11"×8.5" landscape):
//
//	col 0       col 1       col 2       col 3
//	+-----------+-----------+-----------+-----------+
//	| SetWidth  | SetHeight | SetFill   | SetLine   |  row 0  (top, y=7.0)
//	|   2x      |   2x      |   red     |   blue    |
//	+-----------+-----------+-----------+-----------+
//	| LineWt    | FlipX     | FlipY     | TxtAngle  |  row 1
//	|  thick    | mirror    | mirror    |  vertical |
//	+-----------+-----------+-----------+-----------+
//	| EndArrow  | SetText   | SetCharSz | Rounding  |  row 2
//	|  connect  | overwrite |   bigger  |  corners  |
//	+-----------+-----------+-----------+-----------+
//	|         Move connector (spans two cells)       |  row 3  (bottom)
//	+-----------------------+-----------------------+
//
// The connector + its two endpoints exercise Move's BeginX/EndX coupling
// (EC-002 root-cause fix from the audit).
package main

import (
	"fmt"
	"math"
	"os"

	"wijnberg.net/vsdx-go/vsdx"
)

const (
	pageW = 11.0
	pageH = 8.5

	cols     = 4
	cellW    = 2.4
	cellH    = 1.4
	xMargin  = 0.5
	cellGapX = 0.1
	cellGapY = 0.6
	rowTopY  = 7.3 // Visio Y-up: row 0 sits highest
)

func cellCenter(col, row int) (float64, float64) {
	x := xMargin + float64(col)*(cellW+cellGapX) + cellW/2
	y := rowTopY - float64(row)*(cellH+cellGapY)
	return x, y
}

// addLabeledShape drops a baseline shape at the given grid cell, sets a
// neutral fill + border, writes the label, then runs the caller-supplied
// mutator. Returns the shape so callers can chain extra ops (e.g. connect
// to another).
func addLabeledShape(page *vsdx.Page, col, row int, label string, fill string, mutate func(*vsdx.Shape)) *vsdx.Shape {
	cx, cy := cellCenter(col, row)
	s := page.AddShape()
	s.SetX(cx)
	s.SetY(cy)
	s.SetWidth(cellW * 0.65)
	s.SetHeight(cellH * 0.65)
	s.SetLocX(cellW * 0.65 * 0.5)
	s.SetLocY(cellH * 0.65 * 0.5)
	s.AddGeometryRect()
	if fill == "" {
		fill = "#e8f0ff"
	}
	s.SetFillColor(fill)
	s.SetLineColor("#222222")
	s.SetLineWeight(0.013) // ~0.92 pt
	s.SetText(label)
	if mutate != nil {
		mutate(s)
	}
	return s
}

func main() {
	out := "/home/michel/vsdx-go/vsdx-svg-mutations/comprehensive-batch.vsdx"
	if len(os.Args) > 1 {
		out = os.Args[1]
	}

	// Start from blank.vsdx (already in tests/) — it gives us a clean page
	// without any pre-existing shapes that would clutter the diff.
	source := "/home/michel/vsdx-go/tests/blank.vsdx"
	data, err := os.ReadFile(source)
	if err != nil {
		fmt.Fprintf(os.Stderr, "read %s: %v\n", source, err)
		os.Exit(1)
	}
	v, err := vsdx.OpenBytes(data)
	if err != nil {
		fmt.Fprintf(os.Stderr, "open: %v\n", err)
		os.Exit(1)
	}
	defer v.Close()

	page := v.Pages[0]

	// ---- row 0: dimension mutations ----
	addLabeledShape(page, 0, 0, "SetWidth 2x", "#cfe2ff", func(s *vsdx.Shape) {
		s.SetWidth(s.Width() * 2.0)
	})
	addLabeledShape(page, 1, 0, "SetHeight 2x", "#cfeebd", func(s *vsdx.Shape) {
		s.SetHeight(s.Height() * 2.0)
	})
	addLabeledShape(page, 2, 0, "Fill red", "#ffcccc", func(s *vsdx.Shape) {
		s.SetFillColor("#cc0000")
	})
	addLabeledShape(page, 3, 0, "Line blue", "#ffffff", func(s *vsdx.Shape) {
		s.SetLineColor("#0044cc")
		s.SetLineWeight(0.04)
	})

	// ---- row 1: transform / line-style ----
	addLabeledShape(page, 0, 1, "LineWeight thick", "#ffffff", func(s *vsdx.Shape) {
		s.SetLineWeight(0.08)
	})
	addLabeledShape(page, 1, 1, "FlipX", "#fff0c0", func(s *vsdx.Shape) {
		s.SetFlipX(true)
	})
	addLabeledShape(page, 2, 1, "FlipY", "#fff0c0", func(s *vsdx.Shape) {
		s.SetFlipY(true)
	})
	addLabeledShape(page, 3, 1, "Vertical", "#e6e0ff", func(s *vsdx.Shape) {
		s.SetTxtAngle(math.Pi / 2)
	})

	// ---- row 2: text + style ----
	addLabeledShape(page, 0, 2, "Rounded", "#ffffff", func(s *vsdx.Shape) {
		s.SetRounding(0.15)
	})
	addLabeledShape(page, 1, 2, "Overwritten", "#fff8d0", func(s *vsdx.Shape) {
		// SetText after the baseline label: exercises EC-007 pin
		// (cp/pp/fld element preservation policy).
		s.SetText("Overwritten")
	})
	addLabeledShape(page, 2, 2, "Big text", "#ffffff", func(s *vsdx.Shape) {
		s.SetCharSize(18) // 18pt → ensureCharacterCell + EC-006 row copy
	})
	addLabeledShape(page, 3, 2, "FillBkgnd", "#ffeeff", func(s *vsdx.Shape) {
		s.SetFillBkgndColor("#ddaadd")
		s.SetFillPattern(2) // simple hatch — exercises EC-011 hook
	})

	// ---- row 3: connector + move (full-width span) ----
	// Two anchor shapes + one ConnectShapes between them. We translate the
	// connector by Move(0.7, 0) so its BeginX/EndX must both follow (EC-002
	// fix in shape.go:1382). The anchor shapes themselves are NOT moved so
	// the connector now sits OFF-axis relative to its sources — that's the
	// visible "Move both endpoints" proof.
	mkAnchor := func(col int, label string) *vsdx.Shape {
		cx, cy := cellCenter(col, 3)
		s := page.AddShape()
		s.SetX(cx)
		s.SetY(cy)
		w, h := cellW*0.4, cellH*0.55
		s.SetWidth(w)
		s.SetHeight(h)
		s.SetLocX(w / 2)
		s.SetLocY(h / 2)
		s.AddGeometryRect()
		s.SetFillColor("#cccccc")
		s.SetLineColor("#222222")
		s.SetLineWeight(0.013)
		s.SetText(label)
		return s
	}
	anchorA := mkAnchor(0, "A")
	anchorB := mkAnchor(3, "B (Move connector 0.7 right)")

	conn, err := v.ConnectShapes(page, anchorA, anchorB)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ConnectShapes: %v\n", err)
		os.Exit(1)
	}
	conn.SetEndArrow(13)
	conn.SetBeginArrow(13)
	conn.Move(0.7, 0)

	bytes, err := v.SaveVsdxBytes()
	if err != nil {
		fmt.Fprintf(os.Stderr, "save: %v\n", err)
		os.Exit(1)
	}
	if err := os.WriteFile(out, bytes, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "write %s: %v\n", out, err)
		os.Exit(1)
	}
	fmt.Printf("Wrote %s\n", out)
	fmt.Println()
	fmt.Println("Cell map (column, row → mutation):")
	fmt.Println("  (0,0) SetWidth 2x         (1,0) SetHeight 2x       (2,0) Fill red          (3,0) Line blue")
	fmt.Println("  (0,1) LineWeight thick    (1,1) FlipX              (2,1) FlipY             (3,1) Vertical text")
	fmt.Println("  (0,2) Rounded corners     (1,2) SetText overwrite  (2,2) Big char size     (3,2) FillBkgnd+pattern")
	fmt.Println("  row 3: connector A→B with both arrows, then Move(+0.7, 0)")
}
