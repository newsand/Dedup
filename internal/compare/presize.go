package compare

import "deduplicator/internal/model"

// PreSize partitions files by SizeBytes. Files whose size appears in the
// input exactly once cannot possibly have a byte-identical sibling elsewhere
// in the input and are returned as singletons. Everything else is returned
// as "candidates" for the Delegate.
//
// Output ordering is preserved from the input so the caller can stay
// deterministic.
func PreSize(files []model.DiscoveredFile) (candidates, singletons []model.DiscoveredFile) {
	counts := make(map[int64]int, len(files))
	for _, f := range files {
		counts[f.SizeBytes]++
	}
	for _, f := range files {
		if counts[f.SizeBytes] > 1 {
			candidates = append(candidates, f)
		} else {
			singletons = append(singletons, f)
		}
	}
	return candidates, singletons
}
