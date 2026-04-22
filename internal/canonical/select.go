// Package canonical picks one file per DuplicateGroup as the canonical — the
// copy the tool will keep. The rule is intentionally trivial in v1.0:
//
//  1. oldest MTime wins;
//  2. on mtime tie, the smallest AbsolutePath (lexicographic, case-insensitive
//     on Windows) wins.
//
// Anything more clever (EXIF, metadata_score) is out of scope in v1.0: for
// byte-identical files those tie-breakers would always tie.
package canonical

import (
	"errors"
	"runtime"
	"sort"
	"strings"
	"time"

	"deduplicator/internal/model"
)

// Input bundles the DuplicateGroup with a lookup table of its members.
type Input struct {
	Group model.DuplicateGroup
	Files map[string]model.DiscoveredFile // AbsolutePath -> file
}

// ErrEmptyGroup is returned when the group contains no members — an internal
// invariant violation that should surface loudly rather than being swallowed.
var ErrEmptyGroup = errors.New("canonical: group has no members")

// Select returns a CanonicalSelection for the group.
func Select(in Input) (model.CanonicalSelection, error) {
	if len(in.Group.Members) == 0 {
		return model.CanonicalSelection{}, ErrEmptyGroup
	}

	members := append([]string(nil), in.Group.Members...)
	sort.SliceStable(members, func(i, j int) bool {
		a, aok := in.Files[members[i]]
		b, bok := in.Files[members[j]]
		if !aok {
			return false
		}
		if !bok {
			return true
		}
		if !a.MTime.Equal(b.MTime) {
			return a.MTime.Before(b.MTime)
		}
		return compareKey(members[i]) < compareKey(members[j])
	})

	canonicalPath := members[0]
	canonical := in.Files[canonicalPath]

	reason := model.SelectionReason{}
	runnersUpTimes := make([]time.Time, 0, len(members)-1)
	runnersUpPaths := make([]string, 0, len(members)-1)
	for _, p := range members[1:] {
		if f, ok := in.Files[p]; ok {
			runnersUpTimes = append(runnersUpTimes, f.MTime)
		}
		runnersUpPaths = append(runnersUpPaths, p)
	}

	if len(members) > 1 {
		runnerUp := in.Files[members[1]]
		switch {
		case canonical.MTime.Before(runnerUp.MTime):
			reason.ByMTime = true
		case canonical.MTime.Equal(runnerUp.MTime):
			reason.ByLexical = true
		default:
			// Should never happen: sort puts the smaller value first.
			reason.ByLexical = true
		}
	}

	reason.Details = model.SelectionDetails{
		MTimeCanonical:  canonical.MTime,
		MTimesRunnersUp: runnersUpTimes,
		PathCanonical:   canonicalPath,
		PathsRunnersUp:  runnersUpPaths,
	}

	return model.CanonicalSelection{
		BLAKE3Hex: in.Group.BLAKE3Hex,
		Canonical: canonicalPath,
		Reason:    reason,
	}, nil
}

// compareKey normalises a path for lexical comparison. Windows filesystems
// are case-insensitive, so we compare lowercased values there to stay
// consistent with how users perceive paths.
func compareKey(p string) string {
	if runtime.GOOS == "windows" {
		return strings.ToLower(p)
	}
	return p
}
