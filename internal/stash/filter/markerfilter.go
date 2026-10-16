package filter

import (
	"context"
	"fmt"
	"stash-vr/internal/stash/gql"
	"stash-vr/internal/util"
	"strings"

	"github.com/rs/zerolog/log"
)

type MarkerFilter struct {
	FilterOpts   gql.FindFilterType
	MarkerFilter gql.SceneMarkerFilterType
}

func SavedFilterToMarkerFilter(ctx context.Context, savedFilter gql.SavedFilterParts) (MarkerFilter, error) {
	if savedFilter.Mode != gql.FilterModeSceneMarkers {
		return MarkerFilter{}, fmt.Errorf("unsupported filter mode (%s)", savedFilter.Mode)
	}

	out := MarkerFilter{FilterOpts: gql.FindFilterType{Per_page: &perPage}}

	if ff := savedFilter.Find_filter; ff != nil {
		out.FilterOpts.Direction = ff.Direction
		out.FilterOpts.Sort = ff.Sort
		if ff.Sort != nil && strings.HasPrefix(*ff.Sort, "random_") {
			out.FilterOpts.Sort = util.Ptr("random")
		}
	}

	if savedFilter.Object_filter != nil {
		for k, v := range *savedFilter.Object_filter {
			if c, ok := v.(map[string]any); ok {
				setMarkerFilterCriterion(ctx, k, c, &out.MarkerFilter)
			}
		}
	}

	return out, nil
}

func setMarkerFilterCriterion(ctx context.Context, criterionType string, c map[string]any, mf *gql.SceneMarkerFilterType) {
	switch criterionType {
	case "created_at":
		mf.Created_at = parseTimestampCriterionInput(c)
	case "duration":
		mf.Duration = parseFloatCriterionInput(c)
	case "performers":
		mf.Performers = parseMultiCriterionInput(c)
	case "scene_created_at":
		mf.Scene_created_at = parseTimestampCriterionInput(c)
	case "scene_date":
		mf.Scene_date = parseDateCriterionInput(c)
	case "scene_tags":
		mf.Scene_tags = parseHierarchicalMultiCriterionInput(c)
	case "scene_updated_at":
		mf.Scene_updated_at = parseTimestampCriterionInput(c)
	case "tags":
		mf.Tags = parseHierarchicalMultiCriterionInput(c)
	case "updated_at":
		mf.Updated_at = parseTimestampCriterionInput(c)
	default:
		log.Ctx(ctx).Debug().Str("type", criterionType).Interface("value", c).Msg("Ignoring unsupported marker criterion")
	}
}

func parseFloatCriterionInput(c map[string]any) *gql.FloatCriterionInput {
	out := gql.FloatCriterionInput{Modifier: modifier(c)}
	if out.Modifier == gql.CriterionModifierIsNull {
		return &out
	}
	out.Value = GetOr[float64](c, "value.value", 0)
	out.Value2 = Get[float64](c, "value.value2")
	return &out
}
