package library

import (
	"context"
	"errors"
	"fmt"
	"github.com/rs/zerolog/log"
	"stash-vr/internal/stash/filter"
	"stash-vr/internal/stash/gql"
	"strings"
	"sync"
)

const (
	markerIdPrefix = "m"
	maxPlayerLinks = 10000
)

var ErrMarkerNotFound = errors.New("marker not found")

type Marker struct {
	Id         string
	SceneId    string
	Seconds    float64
	Title      string
	PrimaryTag string
	Stream     string
	Screenshot string
}

type MarkerVideoData struct {
	Marker *Marker
	Scene  *VideoData
}

// MarkerVideoId is the id a marker has in the player's library. Scene ids are
// plain numbers, so the prefix keeps the two apart.
func MarkerVideoId(markerId string) string { return markerIdPrefix + markerId }

func ParseMarkerVideoId(videoId string) (markerId string, ok bool) {
	rest, found := strings.CutPrefix(videoId, markerIdPrefix)
	if !found || rest == "" {
		return "", false
	}
	for _, r := range rest {
		if r < '0' || r > '9' {
			return "", false
		}
	}
	return rest, true
}

func newMarker(p *gql.MarkerParts) *Marker {
	m := Marker{Id: p.Id, Seconds: p.Seconds, Title: p.Title, Stream: p.Stream, Screenshot: p.Screenshot}
	if p.Primary_tag != nil {
		m.PrimaryTag = p.Primary_tag.Name
	}
	if p.Scene != nil {
		m.SceneId = p.Scene.Id
	}
	return &m
}

// GetMarkerSections returns one section per saved Stash marker filter and
// refreshes the marker cache. It is separate from GetSections so that
// players that only understand scenes (DeoVR) never see marker ids.
func (libraryService *Service) GetMarkerSections(ctx context.Context) ([]Section, error) {
	res, err, _ := libraryService.single.Do("markerSections", func() (interface{}, error) {
		sections, markers, err := libraryService.getMarkerSections(ctx)
		if err != nil {
			return nil, err
		}

		links := 0
		for _, s := range sections {
			links += len(s.Ids)
		}

		libraryService.muMarkerCache.Lock()
		libraryService.markerCache = markers
		libraryService.Stats.MarkerLinks = links
		libraryService.muMarkerCache.Unlock()

		if total := libraryService.Stats.Links + links; total > maxPlayerLinks {
			log.Ctx(ctx).Warn().Int("links", total).Msg("Many links; HereSphere may struggle above ~10000, consider fewer filters")
		}
		log.Ctx(ctx).Info().Int("sections", len(sections)).Int("links", links).Msg("Marker index built")

		return sections, nil
	})
	if err != nil {
		return nil, err
	}
	return res.([]Section), nil
}

// getMarkerSections builds one section per saved Stash marker filter and
// returns the markers it found, keyed by marker id.
func (libraryService *Service) getMarkerSections(ctx context.Context) ([]Section, map[string]*Marker, error) {
	resp, err := gql.FindSavedMarkerFilters(ctx, libraryService.StashClient)
	if err != nil {
		return nil, nil, fmt.Errorf("FindSavedMarkerFilters: %w", err)
	}

	sections := make([]Section, len(resp.FindSavedFilters))
	found := make([][]*Marker, len(resp.FindSavedFilters))

	wg := sync.WaitGroup{}
	wg.Add(len(resp.FindSavedFilters))
	for i, f := range resp.FindSavedFilters {
		go func(i int, f gql.SavedFilterParts) {
			defer wg.Done()
			flog := log.Ctx(ctx).With().Str("filterId", f.Id).Str("name", f.Name).Logger()

			mf, err := filter.SavedFilterToMarkerFilter(ctx, f)
			if err != nil {
				flog.Warn().Err(err).Msg("Failed to convert marker filter, skipping")
				return
			}
			r, err := gql.FindMarkers(ctx, libraryService.StashClient, nil, &mf.MarkerFilter, &mf.FilterOpts)
			if err != nil {
				flog.Err(err).Msg("Failed to find markers by filter, skipping")
				return
			}

			s := Section{Name: "Markers: " + f.Name}
			for _, sm := range r.FindSceneMarkers.Scene_markers {
				m := newMarker(&sm.MarkerParts)
				if m.SceneId == "" {
					continue
				}
				found[i] = append(found[i], m)
				s.Ids = append(s.Ids, MarkerVideoId(m.Id))
			}
			sections[i] = s
			flog.Debug().Int("markers", len(s.Ids)).Msg("Marker section built")
		}(i, f.SavedFilterParts)
	}
	wg.Wait()

	markers := make(map[string]*Marker)
	out := make([]Section, 0, len(sections))
	for i, s := range sections {
		if len(s.Ids) == 0 {
			continue
		}
		out = append(out, s)
		for _, m := range found[i] {
			markers[m.Id] = m
		}
	}
	return out, markers, nil
}

func (libraryService *Service) GetMarker(ctx context.Context, markerId string) (*MarkerVideoData, error) {
	libraryService.muMarkerCache.RLock()
	m := libraryService.markerCache[markerId]
	libraryService.muMarkerCache.RUnlock()

	if m == nil {
		resp, err := gql.FindMarkers(ctx, libraryService.StashClient, []string{markerId}, nil, nil)
		if err != nil {
			return nil, fmt.Errorf("FindMarkers: %w", err)
		}
		if len(resp.FindSceneMarkers.Scene_markers) == 0 {
			return nil, ErrMarkerNotFound
		}
		m = newMarker(&resp.FindSceneMarkers.Scene_markers[0].MarkerParts)
		if m.SceneId == "" {
			return nil, ErrMarkerNotFound
		}
		libraryService.muMarkerCache.Lock()
		libraryService.markerCache[markerId] = m
		libraryService.muMarkerCache.Unlock()
	}

	scene, err := libraryService.GetScene(ctx, m.SceneId, false)
	if err != nil {
		return nil, err
	}
	return &MarkerVideoData{Marker: m, Scene: scene}, nil
}
