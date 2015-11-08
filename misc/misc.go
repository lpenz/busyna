// Package misc has useful generic functions
package misc

import (
	"log"
	"os"
	"regexp"
)

// Tmpend finishes a temporary file by closing its file descriptor and deleting
// it in the filesystem.
func Tmpend(f *os.File) {
	if f == nil {
		return
	}
	if err := f.Close(); err != nil {
		log.Fatal(err)
	}
	if err := os.Remove(f.Name()); err != nil {
		log.Fatal(err)
	}
	f = nil
}

// ReFindMap returns a map with the matched substrings of a regexp
func ReFindMap(re *regexp.Regexp, l string) map[string]string {
	m := re.FindAllStringSubmatch(l, -1)[0]
	names := re.SubexpNames()
	r := make(map[string]string)
	for i := 0; i < len(names); i++ {
		r[names[i]] = m[i]
	}
	return r
}
