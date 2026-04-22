package naming

// This file intentionally documents the Suppress-name mode (see --suppressname)
// without implementing anything new. Suppress is a toggle on Rules and is
// fully handled inside Flatten and Build.
//
// Semantics recap:
//   * Suppress=false: name = snake(dir1)_snake(dir2)_..._snake(base).ext
//   * Suppress=true:  name = snake(dir1)_snake(dir2)_...   + "_N.ext",
//                     where N is the 1-based lexicographic rank within the
//                     same prefix.
