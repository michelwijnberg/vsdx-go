package vsdx

import (
	"strings"
	"testing"
)

// SetText is the only Laag-C mutator that does NOT have a localize-style
// inheritance fix, because Text inheritance works via the master-fallback in
// Text() rather than via cell aliasing. The audit's concern (EC-007) is
// destructive behaviour: a Visio Text element can contain <cp>, <pp>, <fld>
// markers that interleave with the literal text to apply character format,
// paragraph format, or insert fields. SetText calls clearAllText which
// recursively SetText("") + SetTail("") on every child — but leaves the
// child ELEMENTS in place. So after SetText:
//
//   - The literal text is the new value (correct)
//   - Existing <cp>/<pp>/<fld> children are still present in the XML, but
//     their text content is now empty
//   - The original interleaving positions are lost
//
// For pure-text shapes this is fine. For format-rich shapes the user's
// formatting structure survives but is detached from the text it used to
// annotate. The tests below pin the current behaviour so any future change
// to SetText's contract (e.g. "remove all children" or "preserve formatting
// runs") is intentional and reviewed.

func TestSetTextContract_SetAndRead(t *testing.T) {
	data := loadFixtureBytes(t, setWidthFixture)
	v := openFromBytes(t, data)
	defer v.Close()
	s := findShapeByName(t, v, "Rounded Rectangle.5")
	s.SetText("Hello")
	if got := s.Text(); !strings.HasPrefix(got, "Hello") {
		t.Errorf("Text() = %q, want prefix %q", got, "Hello")
	}
}

func TestSetTextContract_RoundTripXML(t *testing.T) {
	data := loadFixtureBytes(t, setWidthFixture)
	AssertRoundTripXML(t, data,
		func(v *VisioFile) {
			findShapeByName(t, v, "Rounded Rectangle.5").SetText("ROUNDTRIPPED")
		},
		func(t *testing.T, v *VisioFile) {
			s := findShapeByName(t, v, "Rounded Rectangle.5")
			if !strings.HasPrefix(s.Text(), "ROUNDTRIPPED") {
				t.Errorf("after round-trip Text() = %q, want prefix %q", s.Text(), "ROUNDTRIPPED")
			}
		},
	)
}

func TestSetTextContract_EmptyClearsText(t *testing.T) {
	data := loadFixtureBytes(t, setWidthFixture)
	v := openFromBytes(t, data)
	defer v.Close()
	s := findShapeByName(t, v, "Rounded Rectangle.5")
	s.SetText("")
	if got := s.Text(); got != "" {
		t.Errorf("Text() after SetText(\"\") = %q, want empty", got)
	}
}

// Idempotence: SetText("X") then SetText("X") again must yield the same
// observable state. Without this, repeated assignments could compound (e.g.
// duplicate cp children).
func TestSetTextContract_Idempotent(t *testing.T) {
	data := loadFixtureBytes(t, setWidthFixture)
	AssertIdempotent(t, data,
		func(v *VisioFile) {
			findShapeByName(t, v, "Rounded Rectangle.5").SetText("REPEATED")
		},
		func(v *VisioFile) string {
			return snapshotShape(findShapeByName(t, v, "Rounded Rectangle.5")).XMLHash
		},
	)
}

// Master isolation: SetText on an instance must not change any other shape's
// text. Particular concern: shapes whose Text() falls back to master via the
// "no local Text element" path. We test with the Can instances which all
// share a master.
func TestSetTextContract_MasterIsolation(t *testing.T) {
	data := loadFixtureBytes(t, setWidthFixture)
	AssertMasterIsolation(t, data, "Can.15", func(s *Shape) {
		s.SetText("ISOLATED")
	})
}

// EC-007 PIN-DOWN: this test characterises the CURRENT destructive behaviour
// of SetText on a shape that has format-marker children (<cp> for character
// formatting). The current implementation:
//
//   - Keeps the <cp> elements in the XML
//   - Empties their text content (clearAllText)
//   - Loses the interleaving so the new text has no formatting boundaries
//
// We don't say "this is wrong" here — we PIN the current behaviour so a
// future refactor must make an explicit decision. If SetText starts removing
// children entirely (option A) or preserving full runs (option B), this
// test will fail and force a doc update.
func TestSetTextContract_EC007_PreservesFormatMarkerElements(t *testing.T) {
	v, err := Open(testFile("test12_colors.vsdx"))
	if err != nil {
		t.Skipf("test12_colors.vsdx not available: %v", err)
	}
	defer v.Close()

	// Find a shape with at least one <cp> child in its Text element.
	target := findShape(t, v, func(s *Shape) bool {
		textEl := s.XML().FindElement("Text")
		if textEl == nil {
			return false
		}
		for _, c := range textEl.ChildElements() {
			if c.Tag == "cp" {
				return true
			}
		}
		return false
	})

	textEl := target.XML().FindElement("Text")
	cpCountBefore := 0
	for _, c := range textEl.ChildElements() {
		if c.Tag == "cp" {
			cpCountBefore++
		}
	}
	if cpCountBefore == 0 {
		t.Skip("no <cp> children found in fixture")
	}

	target.SetText("OVERWRITTEN")

	// Re-find Text element on the same shape (SetText might have replaced it).
	textElAfter := target.XML().FindElement("Text")
	if textElAfter == nil {
		t.Fatal("Text element disappeared after SetText")
	}
	cpCountAfter := 0
	for _, c := range textElAfter.ChildElements() {
		if c.Tag == "cp" {
			cpCountAfter++
		}
	}

	// CURRENT behaviour: cp count is preserved (clearAllText doesn't delete).
	// If you change SetText to delete format markers, update this test AND
	// section §10 + EC-007 in EDIT_CONTRACTS.md.
	if cpCountAfter != cpCountBefore {
		t.Errorf("EC-007 contract change: cp count went from %d to %d after SetText. Update EDIT_CONTRACTS.md if intentional.",
			cpCountBefore, cpCountAfter)
	}

	// The new text must be readable as the top-level text content.
	if got := target.Text(); !strings.Contains(got, "OVERWRITTEN") {
		t.Errorf("Text() = %q, expected to contain %q", got, "OVERWRITTEN")
	}
}
