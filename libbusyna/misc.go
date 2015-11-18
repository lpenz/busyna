package libbusyna

import (
	"bufio"
	"log"
	"os"
	"regexp"
)

// TmpEnd finishes a temporary file by closing its file descriptor and deleting
// it in the filesystem.
func TmpEnd(f *os.File) {
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

// ReFindMap returns a map with the matched substrings of a regexp.
func ReFindMap(re *regexp.Regexp, l string) map[string]string {
	m := re.FindAllStringSubmatch(l, -1)[0]
	names := re.SubexpNames()
	r := make(map[string]string)
	for i := 0; i < len(names); i++ {
		r[names[i]] = m[i]
	}
	return r
}

// ChanFromFile returns a channel that receives the provided file,
// line-by-line.
func ChanFromFile(filename string) <-chan string {
	fd, err := os.Open(filename)
	if err != nil {
		log.Fatal(err)
	}
	c := make(chan string)
	go func() {
		defer close(c)
		defer fd.Close()
		scanner := bufio.NewScanner(fd)
		for scanner.Scan() {
			c <- scanner.Text()
		}
		if err := scanner.Err(); err != nil {
			log.Fatal(err)
		}
	}()
	return c
}

// ChanFromList returns a channel that receives the elements of the list.
func ChanFromList(list []string) <-chan string {
	c := make(chan string)
	go func() {
		defer close(c)
		for _, l := range list {
			c <- l
		}
	}()
	return c
}
