package heresphere

import (
	"context"
	"stash-vr/internal/api/internal"
	"stash-vr/internal/library"
	"stash-vr/internal/util"
	"time"

	"github.com/rs/zerolog/log"
)

type scanDocDto struct {
	ScanData []scanDataDto `json:"scanData"`
}

type scanDataDto struct {
	id           string
	Link         string   `json:"link"`
	Title        string   `json:"title"`
	DateReleased *string  `json:"dateReleased,omitempty"`
	DateAdded    string   `json:"dateAdded,omitempty"`
	Duration     float64  `json:"duration,omitempty"`
	Rating       *float32 `json:"rating,omitempty"`
	Favorites    *int     `json:"favorites,omitempty"`
	Comments     *int     `json:"comments,omitempty"`
	IsFavorite   *bool    `json:"isFavorite,omitempty"`
	Tags         []tagDto `json:"tags,omitempty"`
}

// buildScan lists scenes and markers, each tagged with the saved filters
// (sections) it appears in, so HereSphere's generated tags can filter by them.
func buildScan(ctx context.Context, vds map[string]*library.VideoData, markers map[string]*library.MarkerVideoData, sections []library.Section, baseUrl string) (*scanDocDto, error) {
	filters := make(map[string][]tagDto)
	for _, s := range sections {
		for _, id := range s.Ids {
			filters[id] = append(filters[id], tagDto{Name: internal.LegendFilter + seperator + s.Name})
		}
	}

	scanDoc := scanDocDto{ScanData: make([]scanDataDto, 0, len(vds)+len(markers))}
	for _, vd := range vds {
		scanData := videoDataToScanDataDto(ctx, vd, baseUrl)
		scanData.Tags = append(scanData.Tags, filters[scanData.id]...)
		scanDoc.ScanData = append(scanDoc.ScanData, scanData)
	}
	for id, mv := range markers {
		scanDoc.ScanData = append(scanDoc.ScanData, scanDataDto{
			id:        id,
			Link:      getVideoDataUrl(baseUrl, id),
			Title:     markerTitle(mv.Marker, mv.Scene.Title()),
			DateAdded: mv.Scene.SceneParts.Created_at.Format(time.DateOnly),
			Tags:      filters[id],
		})
	}
	log.Ctx(ctx).Debug().Int("scenes", len(vds)).Int("markers", len(markers)).Msg("/scan")
	return &scanDoc, nil
}

func videoDataToScanDataDto(ctx context.Context, vd *library.VideoData, baseUrl string) scanDataDto {
	id := vd.Id()
	scanData := scanDataDto{
		id:        id,
		Link:      getVideoDataUrl(baseUrl, id),
		Title:     vd.Title(),
		DateAdded: vd.SceneParts.Created_at.Format(time.DateOnly),
		Duration:  vd.SceneParts.Files[0].Duration,
		Tags:      getTags(vd),
	}
	if vd.SceneParts.Date != nil {
		scanData.DateReleased = util.Ptr(util.NormalizeDate(*vd.SceneParts.Date))
	}
	if vd.SceneParts.Rating100 != nil {
		scanData.Rating = util.Ptr(float32(*vd.SceneParts.Rating100) / 20.0)
	}
	if vd.SceneParts.O_counter != nil {
		scanData.Favorites = vd.SceneParts.O_counter
	}
	if vd.SceneParts.Play_count != nil {
		scanData.Comments = util.Ptr(*vd.SceneParts.Play_count)
	}
	if isFavorite(vd) {
		scanData.IsFavorite = util.Ptr(true)
	}
	return scanData
}
