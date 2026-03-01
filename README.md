# vsdx-go

A Go library for reading, editing, and writing Microsoft Visio (.vsdx) files.

This is a Go port of the Python [vsdx](https://github.com/dave-howard/vsdx) library (v0.6.1).

## Installation

```bash
go get github.com/MichelW6667/vsdx-go/vsdx
```

Requires Go 1.21 or later.

## Quick Start

### Open and read a .vsdx file

```go
package main

import (
    "fmt"
    "github.com/MichelW6667/vsdx-go/vsdx"
)

func main() {
    vis, err := vsdx.Open("my_file.vsdx")
    if err != nil {
        panic(err)
    }
    defer vis.Close()

    // List pages
    for _, name := range vis.GetPageNames() {
        fmt.Println("Page:", name)
    }

    // Get shapes on first page
    page := vis.GetPage(0)
    for _, shape := range page.ChildShapes() {
        fmt.Printf("Shape ID=%s Text=%q Pos=(%.2f, %.2f)\n",
            shape.ID, shape.Text(), shape.X(), shape.Y())
    }
}
```

### Find and modify shapes

```go
// Find shape by text
shape := page.FindShapeByText("Hello")

// Modify shape properties
shape.SetX(3.0)
shape.SetY(5.0)
shape.SetWidth(2.0)
shape.SetHeight(1.5)
shape.SetText("Updated Text")
shape.SetFillColor("#ff0000")

// Save to new file
vis.SaveVsdx("modified.vsdx")
```

### Remove a shape

```go
vis, _ := vsdx.Open("my_file.vsdx")
defer vis.Close()

shape := vis.GetPage(0).FindShapeByText("Shape to remove")
if shape != nil {
    shape.Remove()
    vis.SaveVsdx("shape_removed.vsdx")
}
```

### Search with regex

```go
shapes, err := page.FindShapesByRegex(`\d{3}-\d{4}`)
if err != nil {
    panic(err)
}
for _, s := range shapes {
    fmt.Println(s.Text())
}
```

### Find and replace text

```go
page.FindReplace("old text", "new text")
vis.SaveVsdx("updated.vsdx")
```

### Apply text templates

```go
// Replace {{key}} placeholders in shape text
page.ApplyTextContext(map[string]string{
    "date":     "2024-01-15",
    "scenario": "Production",
})
vis.SaveVsdx("rendered.vsdx")
```

### Work with connectors

```go
page := vis.GetPage(0)

// Get all connections on a page
for _, c := range page.Connects() {
    fmt.Printf("Connector %s -> Shape %s\n", c.ConnectorShapeID(), c.ShapeID())
}

// Find connectors between two shapes
connectors, _ := page.GetConnectorsBetween("1", "", "2", "")

// Get shapes connected to a specific shape
shape := page.FindShapeByID("1")
for _, connected := range shape.ConnectedShapes() {
    fmt.Println("Connected to:", connected.Text())
}
```

### Work with data properties

```go
shape := page.FindShapeByText("Server")
props := shape.DataProperties()

for label, prop := range props {
    fmt.Printf("%s = %s\n", label, prop.Value())
}

// Find shapes by property
shapes := page.FindShapesByPropertyLabelValue("Status", "Active")
```

### Master shapes

```go
// Access master pages
for _, master := range vis.MasterPages {
    fmt.Printf("Master: %s (ID=%s)\n", master.Name(), master.PageID())
}

// Get a shape's master
shape := page.FindShapeByID("1")
if masterPage := shape.MasterPage(); masterPage != nil {
    fmt.Println("Master:", masterPage.Name())
}
```

## API Overview

### Core Types

| Type | Description |
|------|-------------|
| `VisioFile` | Main entry point: open/save .vsdx files, manage pages |
| `Page` | Page or master page: shapes, connections, dimensions |
| `Shape` | Shape or group: text, position, size, style, cells |
| `Cell` | Name/value/formula pair from XML Cell element |
| `DataProperty` | Custom data property of a shape |
| `Connect` | Connection between two shapes |
| `Geometry` | Shape path definition (MoveTo, LineTo, etc.) |

### VisioFile

```go
// Open/close
vis, err := vsdx.Open("file.vsdx")     // also supports .vsdm
vis.Close()

// Pages
vis.GetPage(0)                          // by index
vis.GetPageByName("Page-1")             // by name
vis.GetPageNames()                      // list names

// Master pages
vis.MasterPages                         // []*Page
vis.GetMasterPageByID("2")              // by ID

// Save
vis.SaveVsdx("output.vsdx")
```

### Page

```go
// Properties
page.Name()                             // get name
page.SetName("New Name")                // set name
page.Width() / page.SetWidth(10.0)      // page dimensions
page.Height() / page.SetHeight(12.0)

// Shapes
page.ChildShapes()                      // top-level shapes
page.AllShapes()                        // all shapes recursively

// Search
page.FindShapeByID("5")
page.FindShapeByText("hello")
page.FindShapesByText("hello")          // all matches
page.FindShapesByRegex(`\d+`)           // regex search
page.FindShapeByPropertyLabel("Status")
page.FindShapesByPropertyLabelValue("Status", "Active")
page.FindShapesWithSameMaster(shape)
page.GetConnectorsBetween("1", "", "2", "")

// Edit
page.FindReplace("old", "new")
page.ApplyTextContext(map[string]string{"key": "value"})
page.Connects()                         // all connections
page.AddConnect(connect)                // add connection
```

### Shape

```go
// Position and size (getters and setters)
shape.X() / shape.SetX(3.0)            // PinX/PinY
shape.Y() / shape.SetY(5.0)
shape.Width() / shape.SetWidth(2.0)
shape.Height() / shape.SetHeight(1.5)
shape.Angle() / shape.SetAngle(0.5)
shape.BeginX() / shape.SetBeginX(1.0)  // connector endpoints
shape.EndX() / shape.SetEndX(5.0)

// Text
shape.Text()                            // get text (with master fallback)
shape.SetText("new text")

// Style
shape.LineColor() / shape.SetLineColor("#ff0000")
shape.FillColor() / shape.SetFillColor("#00ff00")
shape.TextColor() / shape.SetTextColor("#0000ff")
shape.LineWeight() / shape.SetLineWeight(0.5)
shape.EndArrow() / shape.SetEndArrow(13)

// Cells
shape.CellValue("PinX")                // get cell value (with master fallback)
shape.SetCellValue("PinX", "5.0")      // set or create cell
shape.CellFormula("LocPinX")
shape.SetCellFormula("LocPinX", "Width*0.5")

// Hierarchy
shape.ChildShapes()                     // direct children
shape.AllShapes()                       // recursive
shape.MasterShape()                     // master shape
shape.MasterPage()                      // master page

// Manipulation
shape.Move(1.0, 2.0)                   // move by delta
shape.Remove()                          // remove from page
shape.FindReplace("old", "new")
shape.ApplyTextFilter(map[string]string{"key": "value"})

// Bounds
shape.Bounds()                          // (beginX, beginY, endX, endY)
shape.CenterXY()                        // center position
shape.RelativeBounds()                  // relative to parent group

// Data properties
shape.DataProperties()                  // map[string]*DataProperty
```

## VSDX File Format

A `.vsdx` file is a ZIP archive containing XML files:

```
[Content_Types].xml           Content type mappings
docProps/app.xml              Document properties
visio/document.xml            Styles and stylesheets
visio/pages/pages.xml         Page definitions (names, IDs)
visio/pages/page1.xml         Page content (shapes, connects)
visio/masters/masters.xml     Master shape definitions
visio/masters/master1.xml     Individual master shapes
```

## Implementation Status

| Phase | Status | Description |
|-------|--------|-------------|
| 1. Reading | Done | Open ZIP, parse XML, populate structs |
| 2. Navigation | Done | Search shapes by ID, text, property, regex, master |
| 3. Editing | Done | Modify properties, text, style, move, remove shapes |
| 4. Writing | Done | Save modified XML back to .vsdx |
| 5. Connectors | Planned | Create new connections between shapes |
| 6. Templating | Planned | Go text/template based rendering |
| 7. Diff | Planned | Compare two .vsdx files |

## Credits

This is a Go port of the Python [vsdx](https://github.com/dave-howard/vsdx) library by Dave Howard.

## License

BSD License - see [LICENSE](LICENSE) for details.
