package scan

import "sort"

// sortSlice is a tiny generic wrapper around sort.SliceStable to keep walk.go
// free of direct sort imports and to preserve stable order for equal keys.
func sortSlice[T any](s []T, less func(i, j int) bool) {
	sort.SliceStable(s, less)
}
