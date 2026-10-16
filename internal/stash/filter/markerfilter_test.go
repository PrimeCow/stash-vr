package filter

import (
	"context"
	"encoding/json"
	"stash-vr/internal/stash/gql"
	"testing"
)

func markerSaved(t *testing.T, sort, objectJSON string) gql.SavedFilterParts {
	t.Helper()
	var obj map[string]interface{}
	if err := json.Unmarshal([]byte(objectJSON), &obj); err != nil {
		t.Fatal(err)
	}
	dir := gql.SortDirectionEnumAsc
	return gql.SavedFilterParts{
		Mode:          gql.FilterModeSceneMarkers,
		Find_filter:   &gql.SavedFilterPartsFind_filterSavedFindFilterType{Sort: &sort, Direction: &dir},
		Object_filter: &obj,
	}
}

// Real saved filters from a Stash instance: ids are a mix of numbers and strings.
func TestMarkerFilter_TagsMixedIdTypes(t *testing.T) {
	sf := markerSaved(t, "random_46847813", `{"tags":{"modifier":"INCLUDES_ALL","value":{"depth":-1,"excluded":[],"items":[{"id":4,"label":"2D"},{"id":"35","label":"ASMR"}]}}}`)
	f, err := SavedFilterToMarkerFilter(context.Background(), sf)
	if err != nil {
		t.Fatal(err)
	}
	tags := f.MarkerFilter.Tags
	if tags == nil || tags.Modifier != gql.CriterionModifierIncludesAll {
		t.Fatalf("tags = %+v", tags)
	}
	if len(tags.Value) != 2 || tags.Value[0] != "4" || tags.Value[1] != "35" {
		t.Errorf("tag ids = %v", tags.Value)
	}
	if tags.Depth == nil || *tags.Depth != -1 {
		t.Errorf("depth = %v", tags.Depth)
	}
	if f.FilterOpts.Sort == nil || *f.FilterOpts.Sort != "random" {
		t.Errorf("sort = %v, want random", f.FilterOpts.Sort)
	}
	if f.FilterOpts.Per_page == nil || *f.FilterOpts.Per_page != -1 {
		t.Errorf("per_page = %v, want -1", f.FilterOpts.Per_page)
	}
}

func TestMarkerFilter_TitleSortKept(t *testing.T) {
	f, err := SavedFilterToMarkerFilter(context.Background(), markerSaved(t, "title", `{}`))
	if err != nil {
		t.Fatal(err)
	}
	if *f.FilterOpts.Sort != "title" {
		t.Errorf("sort = %q", *f.FilterOpts.Sort)
	}
}

func TestMarkerFilter_DurationAndUnsupported(t *testing.T) {
	sf := markerSaved(t, "title", `{"duration":{"modifier":"GREATER_THAN","value":{"value":30}},"bogus":{"modifier":"EQUALS","value":1}}`)
	f, err := SavedFilterToMarkerFilter(context.Background(), sf)
	if err != nil {
		t.Fatal(err)
	}
	if d := f.MarkerFilter.Duration; d == nil || d.Value != 30 || d.Modifier != gql.CriterionModifierGreaterThan {
		t.Errorf("duration = %+v", d)
	}
}

func TestMarkerFilter_WrongModeAndNilParts(t *testing.T) {
	if _, err := SavedFilterToMarkerFilter(context.Background(), gql.SavedFilterParts{Mode: gql.FilterModeScenes}); err == nil {
		t.Error("want error for scene-mode filter")
	}
	f, err := SavedFilterToMarkerFilter(context.Background(), gql.SavedFilterParts{Mode: gql.FilterModeSceneMarkers})
	if err != nil {
		t.Fatalf("nil find_filter/object_filter must not fail: %v", err)
	}
	if f.FilterOpts.Sort != nil || f.MarkerFilter.Tags != nil {
		t.Errorf("want empty filter, got %+v", f)
	}
}
