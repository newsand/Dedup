package compare

import "deduplicator/internal/model"

// Registry resolves a FileType to the Comparator responsible for it.
//
// The zero value is useless; always build one with NewRegistry. Later
// registrations with a conflicting FileType overwrite earlier ones — the
// top-level wiring (internal/cli) is the single place that builds the Registry
// so this is easy to reason about.
type Registry struct {
	byType map[model.FileType]Comparator
}

// NewRegistry returns a Registry pre-populated with the supplied Comparators.
func NewRegistry(cs ...Comparator) *Registry {
	r := &Registry{byType: make(map[model.FileType]Comparator)}
	for _, c := range cs {
		for _, t := range c.Supports() {
			r.byType[t] = c
		}
	}
	return r
}

// Resolve returns the Comparator for t, or (nil, false) when no Comparator
// was registered for that type.
func (r *Registry) Resolve(t model.FileType) (Comparator, bool) {
	c, ok := r.byType[t]
	return c, ok
}
