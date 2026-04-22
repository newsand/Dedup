package naming

import (
	"path/filepath"
	"sort"
	"strings"

	"deduplicator/internal/model"
)

// Rules controls name generation.
type Rules struct {
	// Suppress drops the original base-name and uses an incremental "_N"
	// instead. It is enabled by --suppressname.
	Suppress bool
}

// Flatten turns a relative path into a flattened, snake_cased name (without
// suffix). This is the primary name the file would get before collision
// resolution.
//
// Example: "clientes/2025/evento A/foto 01.png" -> "clientes_2025_evento_a_foto_01.png"
func Flatten(rel string, rules Rules) string {
	rel = filepath.ToSlash(rel)
	ext := strings.ToLower(filepath.Ext(rel))
	segments := strings.Split(strings.TrimSuffix(rel, filepath.Ext(rel)), "/")

	var dirs []string
	if len(segments) > 0 {
		dirs = segments[:len(segments)-1]
	}
	base := ""
	if len(segments) > 0 {
		base = segments[len(segments)-1]
	}

	parts := make([]string, 0, len(dirs)+1)
	for _, d := range dirs {
		if s := ToSnake(d); s != "" {
			parts = append(parts, s)
		}
	}
	if !rules.Suppress {
		if s := ToSnake(base); s != "" {
			parts = append(parts, s)
		}
	}
	name := strings.Join(parts, "_")
	return name + ext
}

// Build resolves collisions and returns one OutputMapping per input file.
//
// Algorithm:
//  1. Compute the primary flattened name for every file.
//  2. Group files by that primary name (for Suppress, also by flattened dir
//     prefix, because Suppress produces many files sharing the same prefix).
//  3. Within each group, sort members by AbsolutePath and assign incremental
//     suffixes: first gets no suffix (non-Suppress) / `_1` (Suppress),
//     next gets `_1` / `_2`, etc. — always deterministic.
//
// outputDir is the directory every OutputPath is rooted in; pass it as the
// absolute output directory.
func Build(files []model.DiscoveredFile, outputDir string, rules Rules) map[string]model.OutputMapping {
	type candidate struct {
		file      model.DiscoveredFile
		primary   string // flattened key; for Suppress this is the "dir prefix + ext"
		extension string
	}

	cands := make([]candidate, len(files))
	for i, f := range files {
		primary := Flatten(f.RelativePath, rules)
		ext := filepath.Ext(primary)
		cands[i] = candidate{file: f, primary: primary, extension: ext}
	}

	// Deterministic processing: sort candidates by AbsolutePath up front so
	// suffix assignment (_1, _2, ...) is stable across runs.
	sort.SliceStable(cands, func(i, j int) bool {
		return cands[i].file.AbsolutePath < cands[j].file.AbsolutePath
	})

	// Group by primary key (for Suppress, grouping is by primary too — since
	// Flatten with Suppress=true yields exactly the dir-prefix + ext).
	groups := make(map[string][]int)
	order := make([]string, 0)
	for i, c := range cands {
		if _, seen := groups[c.primary]; !seen {
			order = append(order, c.primary)
		}
		groups[c.primary] = append(groups[c.primary], i)
	}

	out := make(map[string]model.OutputMapping, len(files))
	for _, key := range order {
		members := groups[key]
		for rank, idx := range members {
			c := cands[idx]

			var outName string
			if rules.Suppress {
				// In suppress mode everyone has a numeric suffix starting at 1,
				// even singletons, because the dir-prefix alone is not
				// sufficient to identify a unique file.
				outName = withSuffix(rank+1, c.primary)
			} else if rank == 0 {
				outName = c.primary
			} else {
				outName = withSuffix(rank, c.primary)
			}

			out[c.file.AbsolutePath] = model.OutputMapping{
				Path:           c.file.AbsolutePath,
				OutputName:     outName,
				OutputPath:     filepath.Join(outputDir, outName),
				CollisionIndex: rank,
			}
		}
	}

	return out
}
