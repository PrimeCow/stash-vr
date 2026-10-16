package heresphere

import (
	"strings"
	"testing"
	"time"

	"stash-vr/internal/library"
	"stash-vr/internal/stash/gql"
)

func str(s string) *string { return &s }

func testScene(files ...*gql.ScenePartsFilesVideoFile) *library.VideoData {
	return &library.VideoData{SceneParts: &gql.SceneParts{
		Id:         "7",
		Title:      str("Scene A"),
		Created_at: time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC),
		Files:      files,
		SceneStreams: []*gql.ScenePartsSceneStreamsSceneStreamEndpoint{
			{Url: "http://s/scene/7/stream", Mime_type: str("video/mp4"), Label: str("Direct stream")},
			{Url: "http://s/scene/7/stream.mp4?apikey=k&resolution=ORIGINAL", Mime_type: str("video/mp4"), Label: str("MP4")},
			{Url: "http://s/scene/7/stream.mp4?apikey=k&resolution=FULL_HD", Mime_type: str("video/mp4"), Label: str("MP4 Full HD (1080p)")},
		},
	}}
}

func TestWithStart(t *testing.T) {
	for _, tc := range []struct {
		in      string
		seconds float64
		want    string
	}{
		{"http://s/scene/7/stream.mp4?apikey=k&resolution=ORIGINAL", 933, "http://s/scene/7/stream.mp4?apikey=k&resolution=ORIGINAL&start=933"},
		{"http://s/scene/7/stream.mp4", 12.5, "http://s/scene/7/stream.mp4?start=12.5"},
		{"http://s/scene/7/stream.mp4?start=1&resolution=LOW", 40, "http://s/scene/7/stream.mp4?resolution=LOW&start=40"},
	} {
		got, err := withStart(tc.in, tc.seconds)
		if err != nil || got != tc.want {
			t.Errorf("withStart(%q, %v) = %q, %v; want %q", tc.in, tc.seconds, got, err, tc.want)
		}
	}
}

func TestMarkerTitle(t *testing.T) {
	for _, tc := range []struct {
		tag, title, want string
	}{
		{"Licking", "", "Licking – Scene A"},
		{"Licking", "Solo", "Licking: Solo – Scene A"},
		{"", "Solo", "Solo – Scene A"},
		{"", "", "Scene A"},
	} {
		got := markerTitle(&library.Marker{PrimaryTag: tc.tag, Title: tc.title}, "Scene A")
		if got != tc.want {
			t.Errorf("markerTitle(%q,%q) = %q; want %q", tc.tag, tc.title, got, tc.want)
		}
	}
}

func TestMarkerDurationMs(t *testing.T) {
	if got := markerDurationMs(100, 40); got != 60000 {
		t.Errorf("got %v", got)
	}
	if got := markerDurationMs(100, 150); got != 0 {
		t.Errorf("marker past end of scene must clamp to 0, got %v", got)
	}
}

func TestBuildMarkerVideoData(t *testing.T) {
	vd := testScene(&gql.ScenePartsFilesVideoFile{Basename: "a.mp4", Duration: 1000, Height: 2160})
	m := &library.Marker{Id: "5", SceneId: "7", Seconds: 100.5, PrimaryTag: "Licking", Stream: "http://s/m/stream", Screenshot: "http://s/m/shot"}

	dto, err := buildMarkerVideoData(&library.MarkerVideoData{Marker: m, Scene: vd})
	if err != nil {
		t.Fatal(err)
	}
	if dto.Title != "Licking – Scene A" || dto.Duration != 899500 {
		t.Errorf("title=%q duration=%v", dto.Title, dto.Duration)
	}
	if len(dto.Media) != 1 || dto.Media[0].Name != "transcoding" || len(dto.Media[0].Sources) != 2 {
		t.Fatalf("media = %+v", dto.Media)
	}
	for _, s := range dto.Media[0].Sources {
		if !strings.Contains(s.Url, "start=100.5") {
			t.Errorf("source %q lacks start offset", s.Url)
		}
	}
	if *dto.ThumbnailVideo != "http://s/m/stream" || *dto.ThumbnailImage != "http://s/m/shot" {
		t.Errorf("thumbs = %v %v", *dto.ThumbnailVideo, *dto.ThumbnailImage)
	}
	if dto.EventServer != nil || *dto.WriteTags || *dto.WriteRating || *dto.WriteFavorite {
		t.Errorf("marker entries must be read-only with no event server: %+v", dto)
	}
}

func TestBuildMarkerVideoData_SceneWithoutFiles(t *testing.T) {
	_, err := buildMarkerVideoData(&library.MarkerVideoData{Marker: &library.Marker{}, Scene: testScene()})
	if err == nil {
		t.Fatal("want error, not panic, for a scene with no files")
	}
}
