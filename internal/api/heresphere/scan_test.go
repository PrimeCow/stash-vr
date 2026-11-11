package heresphere

import (
	"context"
	"slices"
	"strings"
	"testing"

	"stash-vr/internal/library"
	"stash-vr/internal/stash/gql"
)

func tagNames(tags []tagDto) []string {
	names := make([]string, len(tags))
	for i, tg := range tags {
		names[i] = tg.Name
	}
	return names
}

func TestBuildScan_FilterTags(t *testing.T) {
	vd := testScene(&gql.ScenePartsFilesVideoFile{Duration: 100, Height: 1080})
	markers := map[string]*library.MarkerVideoData{"m5": {Marker: &library.Marker{Id: "5", PrimaryTag: "Kiss"}, Scene: vd}}
	sections := []library.Section{{Name: "Faves", Ids: []string{"7"}}, {Name: "Markers: Best", Ids: []string{"m5"}}}

	doc, err := buildScan(context.Background(), map[string]*library.VideoData{"7": vd}, markers, sections, "http://vr")
	if err != nil || len(doc.ScanData) != 2 {
		t.Fatalf("scan = %+v, %v", doc, err)
	}
	for _, sd := range doc.ScanData {
		want := map[string]string{"7": "Filter:Faves", "m5": "Filter:Markers: Best"}[sd.id]
		if !slices.Contains(tagNames(sd.Tags), want) {
			t.Errorf("%s tags = %v, want %q", sd.id, tagNames(sd.Tags), want)
		}
	}
}

// A marker's scan entry must be as complete as a scene's (HereSphere appears
// to drop the generated tags of bare entries), differing only in id, link and title.
func TestBuildScan_MarkerEntryCarriesSceneData(t *testing.T) {
	vd := testScene(&gql.ScenePartsFilesVideoFile{Duration: 100, Height: 1080})
	markers := map[string]*library.MarkerVideoData{"m5": {Marker: &library.Marker{Id: "5", PrimaryTag: "Kiss"}, Scene: vd}}

	doc, err := buildScan(context.Background(), nil, markers, nil, "http://vr")
	if err != nil || len(doc.ScanData) != 1 {
		t.Fatalf("scan = %+v, %v", doc, err)
	}
	m := doc.ScanData[0]
	scene := videoDataToScanDataDto(context.Background(), vd, "http://vr")
	if m.Duration != 100 || !strings.HasSuffix(m.Link, "/m5") || m.Title != "Kiss – Scene A" {
		t.Errorf("marker entry = %+v", m)
	}
	if !slices.Equal(tagNames(m.Tags), tagNames(scene.Tags)) {
		t.Errorf("marker tags = %v, want scene tags %v", tagNames(m.Tags), tagNames(scene.Tags))
	}
}

// HereSphere echoes a video's tags back when they are edited; the generated
// Filter tags must be ignored rather than treated as markers.
func TestIncomingFilterTagIgnored(t *testing.T) {
	stash := &fakeStash{}
	h := &httpHandler{libraryService: library.NewService(stash)}

	h.processIncomingTags(context.Background(), "7", videoDataRequestDto{Tags: &[]tagDto{
		{Name: "Filter:Best"}, {Name: "Played:1"}, {Name: "O-Count:0"}, {Name: "Organized:false"}, {Name: "Rating:?"},
	}})

	if slices.ContainsFunc(stash.ops, func(op string) bool { return strings.Contains(op, "Marker") }) {
		t.Errorf("stash ops = %v, want no marker writes", stash.ops)
	}
}
