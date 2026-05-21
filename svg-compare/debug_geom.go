package main

import (
	"fmt"
	"os"

	"wijnberg.net/vsdx-go/vsdx"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: debug_geom <vsdx-file>")
		os.Exit(1)
	}

	vf, err := vsdx.Open(os.Args[1])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	defer vf.Close()

	for _, page := range vf.Pages {
		fmt.Printf("\n=== Page: %s ===\n", page.Name())
		for _, shape := range page.ChildShapes() {
			dumpShape(shape, 0)
		}
	}
}

func dumpShape(shape *vsdx.Shape, indent int) {
	prefix := ""
	for i := 0; i < indent; i++ {
		prefix += "  "
	}

	fmt.Printf("%sShape %s: %s (Master=%s)\n", prefix, shape.ID, shape.ShapeName, shape.MasterPageID)
	fmt.Printf("%s  Position: PinX=%.4f PinY=%.4f LocPinX=%.4f LocPinY=%.4f\n",
		prefix, shape.X(), shape.Y(), shape.LocX(), shape.LocY())
	fmt.Printf("%s  Size: Width=%.4f Height=%.4f\n", prefix, shape.Width(), shape.Height())

	// Show geometry
	for i, geom := range shape.Geometries {
		noShow := false
		noFill := false
		noLine := false
		for _, c := range geom.Cells {
			if c.Name() == "NoShow" && c.Value() == "1" {
				noShow = true
			}
			if c.Name() == "NoFill" && c.Value() == "1" {
				noFill = true
			}
			if c.Name() == "NoLine" && c.Value() == "1" {
				noLine = true
			}
		}
		fmt.Printf("%s  Geometry%d: NoShow=%v NoFill=%v NoLine=%v\n", prefix, i+1, noShow, noFill, noLine)

		rows := geom.SortedRows()
		for _, row := range rows {
			x := row.X()
			y := row.Y()

			// Get additional cells
			var extra string
			for name, cell := range row.Cells {
				if name != "X" && name != "Y" {
					extra += fmt.Sprintf(" %s=%s", name, cell.Value())
				}
			}

			fmt.Printf("%s    Row[%s] %s: X=%.4f Y=%.4f%s\n",
				prefix, row.Index(), row.RowType(), x, y, extra)
		}
	}

	// Dump children
	for _, child := range shape.ChildShapes() {
		dumpShape(child, indent+1)
	}
}
