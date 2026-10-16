package library

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/Khan/genqlient/graphql"
)

func TestParseMarkerVideoId(t *testing.T) {
	for _, tc := range []struct {
		in     string
		wantId string
		wantOk bool
	}{
		{"m123", "123", true},
		{MarkerVideoId("9"), "9", true},
		{"123", "", false}, // scene id
		{"m", "", false},
		{"mabc", "", false},
		{"m-1", "", false},
		{"m12x", "", false},
		{"M12", "", false},
		{"", "", false},
	} {
		id, ok := ParseMarkerVideoId(tc.in)
		if id != tc.wantId || ok != tc.wantOk {
			t.Errorf("ParseMarkerVideoId(%q) = %q, %v; want %q, %v", tc.in, id, ok, tc.wantId, tc.wantOk)
		}
	}
}

type recordingStash struct {
	ops    []string
	canned map[string]string
}

func (r *recordingStash) MakeRequest(_ context.Context, req *graphql.Request, resp *graphql.Response) error {
	r.ops = append(r.ops, req.OpName)
	body, ok := r.canned[req.OpName]
	if !ok {
		return errors.New("fake stash: no canned response for " + req.OpName)
	}
	return json.Unmarshal([]byte(body), resp.Data)
}

// DeoVR shares GetSections and looks every id up as a scene, so GetSections
// must stay marker-free.
func TestGetSections_NeverQueriesMarkers(t *testing.T) {
	stash := &recordingStash{canned: map[string]string{
		"FindSavedSceneFilters": `{"findSavedFilters":[]}`,
		"FindAllSceneIds":       `{"findScenes":{"scenes":[{"id":"1"},{"id":"2"}]}}`,
		"FindAllTags":           `{"findTags":{"tags":[]}}`,
	}}
	sections, err := NewService(stash).GetSections(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(sections) != 1 || len(sections[0].Ids) != 2 {
		t.Errorf("sections = %+v", sections)
	}
	for _, op := range stash.ops {
		if op == "FindMarkers" || op == "FindSavedMarkerFilters" {
			t.Errorf("GetSections queried markers (%s)", op)
		}
	}
}
