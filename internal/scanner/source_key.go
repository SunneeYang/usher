package scanner

import (
	"sort"
	"strings"
)

func sourceKey(sourceDirs []string) string {
	dirs := append([]string(nil), sourceDirs...)
	sort.Strings(dirs)
	return strings.Join(dirs, "\n")
}
