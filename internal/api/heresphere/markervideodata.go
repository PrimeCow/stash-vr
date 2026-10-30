package heresphere

import (
	"bufio"
	"bytes"
	"fmt"
	"net/url"
	"stash-vr/internal/library"
	"stash-vr/internal/stash"
	"stash-vr/internal/util"
	"strconv"
	"time"
)

// buildMarkerVideoData presents a marker as its whole scene, played through
// Stash's HLS stream (seekable) via a playlist that asks the player to start at
// the marker. Marker entries are read-only.
func buildMarkerVideoData(mv *library.MarkerVideoData, baseUrl string) (*videoDataDto, error) {
	m, vd := mv.Marker, mv.Scene
	if len(vd.SceneParts.Files) == 0 {
		return nil, fmt.Errorf("scene %s has no files", vd.Id())
	}

	e := mediaDto{Name: "hls"}
	for _, s := range stash.GetHLSStream(vd.SceneParts).Sources {
		e.Sources = append(e.Sources, sourceDto{Resolution: s.Resolution, Url: getMarkerPlaylistUrl(baseUrl, m.Id, s.Resolution)})
	}
	if len(e.Sources) == 0 {
		return nil, fmt.Errorf("scene %s has no hls sources", vd.Id())
	}

	dto := videoDataDto{
		Access:         1,
		Title:          markerTitle(m, vd.Title()),
		DateAdded:      vd.SceneParts.Created_at.Format(time.DateOnly),
		Duration:       vd.SceneParts.Files[0].Duration * 1000,
		ThumbnailImage: util.Ptr(stash.ApiKeyed(m.Screenshot)),
		ThumbnailVideo: util.Ptr(stash.ApiKeyed(m.Stream)),
		Media:          []mediaDto{e},
		Tags:           getTags(vd),
		WriteFavorite:  util.Ptr(false),
		WriteRating:    util.Ptr(false),
		WriteTags:      util.Ptr(false),
	}
	set3DFormat(vd, &dto)
	return &dto, nil
}

func markerTitle(m *library.Marker, sceneTitle string) string {
	label := m.PrimaryTag
	if m.Title != "" {
		if label != "" {
			label += ": "
		}
		label += m.Title
	}
	if label == "" {
		return sceneTitle
	}
	return label + " – " + sceneTitle
}

func getMarkerPlaylistUrl(baseUrl string, markerId string, resolution int) string {
	return getVideoDataUrl(baseUrl, library.MarkerVideoId(markerId)) + "/stream.m3u8?resolution=" + strconv.Itoa(resolution)
}

// startPlaylist rewrites a Stash HLS playlist fetched from upstream so it can
// be served from here: segment uris are made absolute (Stash writes them
// host-relative) and an EXT-X-START tag asks the player to begin at seconds.
func startPlaylist(body []byte, upstream *url.URL, seconds float64) ([]byte, error) {
	var out bytes.Buffer
	sc := bufio.NewScanner(bytes.NewReader(body))
	first := true
	for sc.Scan() {
		line := sc.Text()
		if first {
			if line != "#EXTM3U" {
				return nil, fmt.Errorf("not an hls playlist")
			}
			fmt.Fprintf(&out, "%s\n#EXT-X-START:TIME-OFFSET=%s,PRECISE=YES\n", line, strconv.FormatFloat(seconds, 'f', -1, 64))
			first = false
			continue
		}
		if line != "" && line[0] != '#' {
			ref, err := url.Parse(line)
			if err != nil {
				return nil, fmt.Errorf("parse segment uri: %w", err)
			}
			line = upstream.ResolveReference(ref).String()
		}
		out.WriteString(line + "\n")
	}
	if first {
		return nil, fmt.Errorf("empty playlist")
	}
	return out.Bytes(), sc.Err()
}
