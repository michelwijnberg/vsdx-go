package main

import (
	"fmt"

	"wijnberg.net/vsdx-go/vsdx"
)

func main() {
	vf, _ := vsdx.Open("vsdx-svg/reference-architecture.vsdx")
	defer vf.Close()
	for _, page := range vf.Pages {
		for _, shape := range page.ChildShapes() {
			text := shape.Text()
			if text != "" {
				fmt.Printf("Shape %s (%s): Text='%s'\n", shape.ID, shape.ShapeName, text)
			}
		}
	}
}
