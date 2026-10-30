package stash

import (
	"fmt"
	"regexp"
	"slices"
	"stash-vr/internal/stash/gql"
	"strconv"
	"strings"
)

type Stream struct {
	Name    string
	Sources []Source
}

type Source struct {
	Resolution int
	Url        string
}

var rgxResolution = regexp.MustCompile(`\((\d+)p\)`)

func GetDirectStream(sp *gql.SceneParts) Stream {
	directStream := Source{
		Resolution: sp.Files[0].Height,
		Url:        *sp.Paths.Stream,
	}

	return Stream{
		Name:    "direct",
		Sources: []Source{directStream},
	}
}
func GetTranscodingStream(sp *gql.SceneParts) Stream {
	return Stream{
		Name:    "transcoding",
		Sources: sourcesByMime(sp, "video/mp4"),
	}
}

// GetHLSStream is Stash's segmented stream: unlike the mp4 transcode it can be
// seeked, as each segment is transcoded on request.
func GetHLSStream(sp *gql.SceneParts) Stream {
	return Stream{
		Name:    "hls",
		Sources: sourcesByMime(sp, "application/vnd.apple.mpegurl"),
	}
}

func sourcesByMime(sp *gql.SceneParts, mime string) []Source {
	sources := make([]Source, 0)
	seenResolutions := make(map[int]struct{})
	for _, stream := range sp.SceneStreams {
		if strings.HasPrefix(*stream.Mime_type, mime) && *stream.Label != "Direct stream" {
			resolution, err := parseResolutionFromLabel(*stream.Label)
			if err != nil {
				resolution = sp.Files[0].Height
			}
			if _, seen := seenResolutions[resolution]; seen {
				continue
			}
			sources = append(sources, Source{
				Resolution: resolution,
				Url:        stream.Url,
			})
			seenResolutions[resolution] = struct{}{}
		}
	}
	slices.SortFunc(sources, func(a, b Source) int { return b.Resolution - a.Resolution })
	return sources
}

func parseResolutionFromLabel(label string) (int, error) {
	match := rgxResolution.FindStringSubmatch(label)
	if len(match) < 2 {
		return 0, fmt.Errorf("no resolution height found in label")
	}
	res, err := strconv.Atoi(match[1])
	if err != nil {
		return 0, fmt.Errorf("atoi: %w", err)
	}
	return res, nil
}
