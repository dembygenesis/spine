package testsuite

import (
	"crypto/sha256"
	"encoding/hex"
	"io/fs"
	"sort"
)

// schemaHash fingerprints every file in migrations: the same files always give
// the same hash, and any change to any of them gives a different one.
//
// The names are sorted because fs.WalkDir's order is not a promise, and a hash
// that depends on iteration order rebuilds the template for nothing.
func schemaHash(migrations fs.FS) string {
	var names []string
	_ = fs.WalkDir(migrations, ".", func(path string, d fs.DirEntry, err error) error {
		if err == nil && !d.IsDir() {
			names = append(names, path)
		}
		return nil
	})
	sort.Strings(names)

	sum := sha256.New()
	for _, n := range names {
		b, err := fs.ReadFile(migrations, n)
		if err != nil {
			continue
		}
		sum.Write([]byte(n))
		sum.Write(b)
	}
	return hex.EncodeToString(sum.Sum(nil))[:16]
}
