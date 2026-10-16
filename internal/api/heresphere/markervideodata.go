package heresphere

import (
	"fmt"
	"net/url"
	"stash-vr/internal/library"
	"stash-vr/internal/stash"
	"stash-vr/internal/util"
	"strconv"
	"time"
)

// buildMarkerVideoData presents a marker as a video that plays the scene's
// transcoded stream from the marker's start. Marker entries are read-only.
func buildMarkerVideoData(mv *library.MarkerVideoData) (*videoDataDto, error) {
	m, vd := mv.Marker, mv.Scene
	if len(vd.SceneParts.Files) == 0 {
		return nil, fmt.Errorf("scene %s has no files", vd.Id())
	}

	e := mediaDto{Name: "transcoding"}
	for _, s := range stash.GetTranscodingStream(vd.SceneParts).Sources {
		u, err := withStart(s.Url, m.Seconds)
		if err != nil {
			return nil, err
		}
		e.Sources = append(e.Sources, sourceDto{Resolution: s.Resolution, Url: u})
	}
	if len(e.Sources) == 0 {
		return nil, fmt.Errorf("scene %s has no transcoding sources", vd.Id())
	}

	dto := videoDataDto{
		Access:         1,
		Title:          markerTitle(m, vd.Title()),
		DateAdded:      vd.SceneParts.Created_at.Format(time.DateOnly),
		Duration:       markerDurationMs(vd.SceneParts.Files[0].Duration, m.Seconds),
		ThumbnailImage: util.Ptr(stash.ApiKeyed(m.Screenshot)),
		ThumbnailVideo: util.Ptr(stash.ApiKeyed(m.Stream)),
		Media:          []mediaDto{e},
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

// markerDurationMs is the remaining scene length after the marker start, in
// milliseconds, never negative.
func markerDurationMs(sceneSeconds, startSeconds float64) float64 {
	return max(sceneSeconds-startSeconds, 0) * 1000
}

func withStart(rawUrl string, seconds float64) (string, error) {
	u, err := url.Parse(rawUrl)
	if err != nil {
		return "", fmt.Errorf("parse stream url: %w", err)
	}
	q := u.Query()
	q.Set("start", strconv.FormatFloat(seconds, 'f', -1, 64))
	u.RawQuery = q.Encode()
	return u.String(), nil
}
