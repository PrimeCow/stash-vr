package heresphere

import (
	"net/url"
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
			{Url: "http://s/scene/7/stream.m3u8?apikey=k&resolution=ORIGINAL", Mime_type: str("application/vnd.apple.mpegurl"), Label: str("HLS")},
			{Url: "http://s/scene/7/stream.m3u8?apikey=k&resolution=FULL_HD", Mime_type: str("application/vnd.apple.mpegurl"), Label: str("HLS Full HD (1080p)")},
		},
	}}
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

func TestBuildMarkerVideoData(t *testing.T) {
	vd := testScene(&gql.ScenePartsFilesVideoFile{Basename: "a.mp4", Duration: 1000, Height: 2160})
	m := &library.Marker{Id: "5", SceneId: "7", Seconds: 100.5, PrimaryTag: "Licking", Stream: "http://s/m/stream", Screenshot: "http://s/m/shot"}

	dto, err := buildMarkerVideoData(&library.MarkerVideoData{Marker: m, Scene: vd}, "http://vr")
	if err != nil {
		t.Fatal(err)
	}
	if dto.Title != "Licking – Scene A" || dto.Duration != 1000000 {
		t.Errorf("title=%q duration=%v", dto.Title, dto.Duration)
	}
	if len(dto.Media) != 1 || dto.Media[0].Name != "hls" || len(dto.Media[0].Sources) != 2 {
		t.Fatalf("media = %+v", dto.Media)
	}
	if got := dto.Media[0].Sources[1].Url; got != "http://vr/heresphere/m5/stream.m3u8?resolution=1080" {
		t.Errorf("source url = %q", got)
	}
	if *dto.ThumbnailVideo != "http://s/m/stream" || *dto.ThumbnailImage != "http://s/m/shot" {
		t.Errorf("thumbs = %v %v", *dto.ThumbnailVideo, *dto.ThumbnailImage)
	}
	if dto.EventServer != nil || *dto.WriteTags || *dto.WriteRating || *dto.WriteFavorite {
		t.Errorf("marker entries must be read-only with no event server: %+v", dto)
	}
}

func TestBuildMarkerVideoData_SceneWithoutFiles(t *testing.T) {
	_, err := buildMarkerVideoData(&library.MarkerVideoData{Marker: &library.Marker{}, Scene: testScene()}, "http://vr")
	if err == nil {
		t.Fatal("want error, not panic, for a scene with no files")
	}
}

func TestStartPlaylist(t *testing.T) {
	upstream, _ := url.Parse("http://s:9999/scene/7/stream.m3u8?apikey=k&resolution=FULL_HD")
	in := "#EXTM3U\n#EXT-X-VERSION:3\n#EXTINF:2.000000,\n/scene/7/stream.m3u8/0.ts?apikey=k&resolution=FULL_HD\n#EXT-X-ENDLIST\n"
	want := "#EXTM3U\n#EXT-X-START:TIME-OFFSET=100.5,PRECISE=YES\n#EXT-X-VERSION:3\n#EXTINF:2.000000,\nhttp://s:9999/scene/7/stream.m3u8/0.ts?apikey=k&resolution=FULL_HD\n#EXT-X-ENDLIST\n"

	got, err := startPlaylist([]byte(in), upstream, 100.5)
	if err != nil || string(got) != want {
		t.Errorf("got %q, %v\nwant %q", got, err, want)
	}
	if _, err := startPlaylist([]byte("<html>"), upstream, 1); err == nil {
		t.Error("want error for a non-playlist body")
	}
}
