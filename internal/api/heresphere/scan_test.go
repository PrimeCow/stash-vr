package heresphere

import (
	"context"
	"slices"
	"testing"

	"stash-vr/internal/library"
	"stash-vr/internal/stash/gql"
)

func TestBuildScan_FilterTags(t *testing.T) {
	vd := testScene(&gql.ScenePartsFilesVideoFile{Duration: 100, Height: 1080})
	markers := map[string]*library.MarkerVideoData{"m5": {Marker: &library.Marker{Id: "5", PrimaryTag: "Kiss"}, Scene: vd}}
	sections := []library.Section{{Name: "Faves", Ids: []string{"7"}}, {Name: "Markers: Best", Ids: []string{"m5"}}}

	doc, err := buildScan(context.Background(), map[string]*library.VideoData{"7": vd}, markers, sections, "http://vr")
	if err != nil || len(doc.ScanData) != 2 {
		t.Fatalf("scan = %+v, %v", doc, err)
	}
	for _, sd := range doc.ScanData {
		names := make([]string, len(sd.Tags))
		for i, tg := range sd.Tags {
			names[i] = tg.Name
		}
		want := map[string]string{"7": "Filter:Faves", "m5": "Filter:Markers: Best"}[sd.id]
		if !slices.Contains(names, want) {
			t.Errorf("%s tags = %v, want %q", sd.id, names, want)
		}
	}
}
