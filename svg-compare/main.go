package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"wijnberg.net/vsdx-go/vsdx"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: svg-compare <vsdx-file>")
		os.Exit(1)
	}

	vsdxPath := os.Args[1]

	// Open the VSDX file
	vf, err := vsdx.Open(vsdxPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening %s: %v\n", vsdxPath, err)
		os.Exit(1)
	}
	defer vf.Close()

	// Get output directory
	base := filepath.Base(vsdxPath)
	baseName := strings.TrimSuffix(base, ".vsdx")
	outDir := filepath.Join(filepath.Dir(vsdxPath), baseName+"-generated")
	os.MkdirAll(outDir, 0755)

	// Process each page
	for _, page := range vf.Pages {
		fmt.Printf("\n=== Page: %s ===\n", page.Name())

		pageWidth := page.Width()
		pageHeight := page.Height()
		fmt.Printf("Page dimensions: %.2f x %.2f inches\n", pageWidth, pageHeight)

		// Get all shapes on the page
		shapes := page.ChildShapes()
		fmt.Printf("Found %d top-level shapes\n", len(shapes))

		for _, shape := range shapes {
			shapeID := shape.ID
			masterRef := ""
			if shape.MasterPageID != "" {
				masterRef = fmt.Sprintf(" (master: %s)", shape.MasterPageID)
			}

			// Get shape dimensions and position
			w := shape.Width()
			h := shape.Height()
			pinX := shape.X()
			pinY := shape.Y()
			locPinX := shape.LocX()
			locPinY := shape.LocY()

			isConnector := strings.Contains(strings.ToLower(shape.ShapeName), "connector") || shape.ShapeType == "Connector"

			fmt.Printf("\nShape %s: %s%s\n", shapeID, shape.ShapeName, masterRef)
			fmt.Printf("  Type: %s, Connector: %v\n", shape.ShapeType, isConnector)
			fmt.Printf("  Size: %.3f x %.3f inches\n", w, h)
			fmt.Printf("  Pin: (%.3f, %.3f), LocPin: (%.3f, %.3f)\n", pinX, pinY, locPinX, locPinY)

			// Check if it's a connector and show begin/end
			if shape.HasBeginX() {
				fmt.Printf("  Begin: (%.3f, %.3f), End: (%.3f, %.3f)\n",
					shape.BeginX(), shape.BeginY(), shape.EndX(), shape.EndY())
			}

			// Get geometry
			geoms := shape.Geometries
			fmt.Printf("  Geometries: %d\n", len(geoms))
			for i, geom := range geoms {
				rows := geom.SortedRows()
				fmt.Printf("    Geom%d: %d rows\n", i+1, len(rows))
				for j, row := range rows {
					if j < 5 || j == len(rows)-1 {
						fmt.Printf("      Row %s: %s X=%.3f Y=%.3f\n", row.Index(), row.RowType(), row.X(), row.Y())
					} else if j == 5 {
						fmt.Printf("      ... (%d more rows)\n", len(rows)-6)
					}
				}
			}

			// Try to render to SVG
			result, err := vsdx.ShapeToSVG(shape, vsdx.WithSize(200, 200))
			if err != nil {
				fmt.Printf("  SVG Error: %v\n", err)
				continue
			}

			// Save the SVG
			svgPath := filepath.Join(outDir, fmt.Sprintf("shape-%s.svg", shapeID))
			if err := os.WriteFile(svgPath, result.SVG, 0644); err != nil {
				fmt.Printf("  Write error: %v\n", err)
				continue
			}
			fmt.Printf("  Wrote: %s\n", svgPath)
		}
	}

	fmt.Printf("\n\nGenerated SVGs in: %s\n", outDir)
}
