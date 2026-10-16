package heresphere

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/Khan/genqlient/graphql"
	"stash-vr/internal/library"
)

// fakeStash records every operation and fails it, so a test can assert that
// nothing but the expected read reached Stash.
type fakeStash struct{ ops []string }

func (f *fakeStash) MakeRequest(_ context.Context, req *graphql.Request, _ *graphql.Response) error {
	f.ops = append(f.ops, req.OpName)
	return errors.New("fake stash: unavailable")
}

func do(t *testing.T, h http.Handler, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(method, path, strings.NewReader(body)))
	return w
}

// HereSphere's two-way sync posts rating/tags/deleteFile for the open video.
// For a marker none of that may reach Stash; the only call allowed is the read
// that resolves the marker.
func TestMarkerVideoData_IgnoresWrites(t *testing.T) {
	stash := &fakeStash{}
	h := Router(library.NewService(stash))

	w := do(t, h, http.MethodPost, "/m5", `{"deleteFile":true,"rating":4.5,"isFavorite":true,"tags":[{"name":"#:x"},{"name":"/o"}]}`)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500 (fake stash is down)", w.Code)
	}
	if !slices.Equal(stash.ops, []string{"FindMarkers"}) {
		t.Errorf("stash ops = %v, want only [FindMarkers]", stash.ops)
	}
}

func TestMarkerEvents_NoOp(t *testing.T) {
	stash := &fakeStash{}
	h := Router(library.NewService(stash))

	w := do(t, h, http.MethodPost, "/events/m5", `{"id":"http://x/heresphere/events/m5","event":1}`)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}
	if len(stash.ops) != 0 {
		t.Errorf("stash ops = %v, want none", stash.ops)
	}
}

// cannedStash answers operations with fixed JSON and records what was asked.
type cannedStash struct {
	mu     sync.Mutex
	ops    []string
	canned map[string]string
}

func (c *cannedStash) MakeRequest(_ context.Context, req *graphql.Request, resp *graphql.Response) error {
	c.mu.Lock()
	c.ops = append(c.ops, req.OpName)
	c.mu.Unlock()
	body, ok := c.canned[req.OpName]
	if !ok {
		return errors.New("fake stash: no canned response for " + req.OpName)
	}
	return json.Unmarshal([]byte(body), resp.Data)
}

func TestIndex_AppendsMarkerSectionsAfterScenes(t *testing.T) {
	stash := &cannedStash{canned: map[string]string{
		"UIConfiguration":        `{"configuration":{"ui":{}}}`,
		"FindSavedSceneFilters":  `{"findSavedFilters":[]}`,
		"FindAllSceneIds":        `{"findScenes":{"scenes":[{"id":"1"},{"id":"2"}]}}`,
		"FindAllTags":            `{"findTags":{"tags":[]}}`,
		"FindSavedMarkerFilters": `{"findSavedFilters":[{"id":"9","name":"Best","mode":"SCENE_MARKERS","find_filter":{"sort":"title","direction":"ASC"},"object_filter":{"tags":{"modifier":"INCLUDES_ALL","value":{"depth":-1,"items":[{"id":4}]}}}},{"id":"10","name":"Broken","mode":"SCENES","find_filter":null,"object_filter":{}}]}`,
		"FindMarkers":            `{"findSceneMarkers":{"scene_markers":[{"id":"31","seconds":5,"title":"","stream":"s","screenshot":"p","primary_tag":{"name":"Licking"},"scene":{"id":"2"}}]}}`,
	}}
	h := Router(library.NewService(stash))

	w := do(t, h, http.MethodPost, "/", `{}`)

	var got struct {
		Library []struct {
			Name string
			List []string
		}
	}
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("status %d, body %q: %v", w.Code, w.Body.String(), err)
	}
	if len(got.Library) != 2 {
		t.Fatalf("library = %+v, want scene section then one marker section (a filter that cannot be converted is skipped)", got.Library)
	}
	if got.Library[0].Name != "All" || len(got.Library[0].List) != 2 {
		t.Errorf("scene section = %+v", got.Library[0])
	}
	if got.Library[1].Name != "Markers: Best" || len(got.Library[1].List) != 1 || !strings.HasSuffix(got.Library[1].List[0], "/heresphere/m31") {
		t.Errorf("marker section = %+v", got.Library[1])
	}
}

// Marker sections are an extra: if Stash can't supply them, scenes still load.
func TestIndex_MarkerQueryFailureKeepsScenes(t *testing.T) {
	stash := &cannedStash{canned: map[string]string{
		"UIConfiguration":       `{"configuration":{"ui":{}}}`,
		"FindSavedSceneFilters": `{"findSavedFilters":[]}`,
		"FindAllSceneIds":       `{"findScenes":{"scenes":[{"id":"1"}]}}`,
		"FindAllTags":           `{"findTags":{"tags":[]}}`,
		// FindSavedMarkerFilters deliberately missing: the fake returns an error.
	}}
	h := Router(library.NewService(stash))

	w := do(t, h, http.MethodPost, "/", `{}`)

	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"All"`) || strings.Contains(w.Body.String(), "Markers") {
		t.Errorf("status %d, body %s", w.Code, w.Body.String())
	}
}
