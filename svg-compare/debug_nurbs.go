package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"wijnberg.net/vsdx-go/vsdx"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: debug_nurbs <vsdx-file>")
		os.Exit(1)
	}

	vsdxPath := os.Args[1]

	vf, err := vsdx.Open(vsdxPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening %s: %v\n", vsdxPath, err)
		os.Exit(1)
	}
	defer vf.Close()

	for _, page := range vf.Pages {
		shapes := page.ChildShapes()
		for _, shape := range shapes {
			// Focus on connectors
			if !strings.Contains(strings.ToLower(shape.ShapeName), "connector") {
				continue
			}

			fmt.Printf("\n=== Shape %s: %s ===\n", shape.ID, shape.ShapeName)
			fmt.Printf("Width: %.6f inches\n", shape.Width())
			fmt.Printf("Height: %.6f inches\n", shape.Height())

			// Get the geometry
			for i, geom := range shape.Geometries {
				fmt.Printf("\nGeometry %d:\n", i)
				rows := geom.SortedRows()
				for _, row := range rows {
					fmt.Printf("  Row %s (%s): X=%.6f Y=%.6f\n",
						row.Index(), row.RowType(), row.X(), row.Y())

					// For NURBSTo, look at E cell
					if strings.ToLower(row.RowType()) == "nurbsto" {
						if eCell := row.Cells["E"]; eCell != nil {
							formula := eCell.Formula()
							if formula == "" {
								formula = eCell.Value()
							}
							fmt.Printf("    E formula: %s\n", formula)
							parseAndShowNURBS(formula, shape.Width(), shape.Height())
						}
					}
				}
			}
		}
	}
}

func parseAndShowNURBS(formula string, width, height float64) {
	upper := strings.ToUpper(formula)
	if !strings.HasPrefix(upper, "NURBS(") || !strings.HasSuffix(formula, ")") {
		fmt.Printf("    Not a valid NURBS formula\n")
		return
	}

	inner := formula[6 : len(formula)-1]
	parts := strings.Split(inner, ",")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}

	if len(parts) < 4 {
		fmt.Printf("    Not enough parameters\n")
		return
	}

	knotLast, _ := strconv.ParseFloat(parts[0], 64)
	degree, _ := strconv.ParseInt(parts[1], 10, 32)
	xType, _ := strconv.ParseInt(parts[2], 10, 32)
	yType, _ := strconv.ParseInt(parts[3], 10, 32)

	fmt.Printf("    Parsed NURBS: knotLast=%.6f, degree=%d, xType=%d, yType=%d\n",
		knotLast, degree, xType, yType)

	// Parse control points (groups of 4: x, y, knot, weight)
	cpData := parts[4:]
	numCPs := len(cpData) / 4
	fmt.Printf("    Control points (%d):\n", numCPs)

	for i := 0; i < numCPs; i++ {
		x, _ := strconv.ParseFloat(cpData[i*4], 64)
		y, _ := strconv.ParseFloat(cpData[i*4+1], 64)
		knot, _ := strconv.ParseFloat(cpData[i*4+2], 64)
		weight, _ := strconv.ParseFloat(cpData[i*4+3], 64)

		// Convert to absolute based on xType/yType
		var absX, absY float64
		if xType == 0 {
			absX = x * width
		} else {
			absX = x
		}
		if yType == 0 {
			absY = y * height
		} else {
			absY = y
		}

		fmt.Printf("      CP%d: raw(%.6f, %.6f) knot=%.2f weight=%.2f → absolute(%.6f, %.6f)\n",
			i+1, x, y, knot, weight, absX, absY)
	}
}
