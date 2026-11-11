package library

import (
	"slices"
	"testing"
)

func TestSectionNames(t *testing.T) {
	s := NewService(nil)
	s.membership.setScenes([]Section{{Name: "Best", Ids: []string{"7", "8"}}, {Name: "VR", Ids: []string{"7"}}})
	s.membership.setMarkers([]Section{{Name: "Markers: Best", Ids: []string{"m5"}}})

	for id, want := range map[string][]string{
		"7":  {"Best", "VR"},
		"8":  {"Best"},
		"m5": {"Markers: Best"},
		"9":  nil,
		"m6": nil,
	} {
		if got := s.SectionNames(id); !slices.Equal(got, want) {
			t.Errorf("SectionNames(%q) = %v, want %v", id, got, want)
		}
	}

	// rebuilding scene sections must not drop marker membership
	s.membership.setScenes(nil)
	if got := s.SectionNames("m5"); !slices.Equal(got, []string{"Markers: Best"}) {
		t.Errorf("after scene rebuild, SectionNames(m5) = %v", got)
	}
}
