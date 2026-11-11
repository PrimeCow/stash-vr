package library

import "sync"

// membership remembers which sections (saved filters) each video id appeared
// in when the sections were last built, so per-video responses can name their
// filters without re-running every saved filter.
type membership struct {
	mu      sync.RWMutex
	scenes  map[string][]string
	markers map[string][]string
}

func sectionNamesById(sections []Section) map[string][]string {
	m := make(map[string][]string)
	for _, s := range sections {
		for _, id := range s.Ids {
			m[id] = append(m[id], s.Name)
		}
	}
	return m
}

func (m *membership) setScenes(sections []Section) {
	names := sectionNamesById(sections)
	m.mu.Lock()
	m.scenes = names
	m.mu.Unlock()
}

func (m *membership) setMarkers(sections []Section) {
	names := sectionNamesById(sections)
	m.mu.Lock()
	m.markers = names
	m.mu.Unlock()
}

// SectionNames returns the names of the sections the video id (scene id or
// marker video id) was in when the index was last built.
func (libraryService *Service) SectionNames(videoId string) []string {
	m := &libraryService.membership
	m.mu.RLock()
	defer m.mu.RUnlock()
	if _, ok := ParseMarkerVideoId(videoId); ok {
		return m.markers[videoId]
	}
	return m.scenes[videoId]
}
