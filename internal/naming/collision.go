package naming

import (
	"fmt"
	"path"
	"strings"
)

// withSuffix returns baseName with an incremental "_N" applied before the
// extension. withSuffix(1, "foo.png") = "foo_1.png".
func withSuffix(idx int, baseName string) string {
	if idx <= 0 {
		return baseName
	}
	ext := path.Ext(baseName)
	stem := strings.TrimSuffix(baseName, ext)
	return fmt.Sprintf("%s_%d%s", stem, idx, ext)
}
