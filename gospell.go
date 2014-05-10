package main

// fixes misspelings
// Abandonned  ==> Abandoned
// based on codespell

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
)

type misspelling struct {
	data   string
	fix    bool
	reason string
}

var (
	misspellings map[string]misspelling
	rx           = regexp.MustCompile(`[\w\-]+`)
	wg           sync.WaitGroup
)

func buildMisspellings(filename *string) {
	var fix bool
	var reason string
	misspellings = make(map[string]misspelling)

	f, _ := os.Open(*filename)
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		items := strings.Split(line, "->")

		key := items[0]

		data := strings.TrimSpace(items[1])
		comma := strings.LastIndex(data, ",")
		if comma == -1 {
			fix = true
			reason = ""
		} else if comma == len(data)-1 {
			data = data[:comma]
			fix = false
			reason = ""
		} else {
			reason = strings.TrimSpace(data[comma+1:])
			data = data[:comma]
			fix = false
		}
		misspellings[key] = misspelling{data: data, fix: fix, reason: reason}
	}
}

func fixCase(word string, fixword string) string {
	if word == strings.Title(word) {
		return strings.Title(fixword)
	} else if word == strings.ToUpper(word) {
		return strings.ToUpper(word)
	} else {
		return fixword
	}
}

func parseFile(filename string) {
	defer wg.Done()
	var i int
	f, _ := os.Open(filename)
	defer f.Close()
	s := bufio.NewScanner(f)
	for s.Scan() {
		for _, word := range rx.FindAllString(s.Text(), -1) {
			lword := strings.ToLower(word)
			if misspelling, ok := misspellings[lword]; ok {
				fixword := fixCase(word, misspelling.data)
				fmt.Printf("\033[33m%s:%d\033[0m: \033[31m%s\033[0m ==> \033[32m%s\033[0m\n", filename, i, word, fixword)
			}
		}

		i++
	}
}

func isHidden(path string) bool {
	// TODO: handle unicode paths
	basename := filepath.Base(path)
	if basename != "." && basename != ".." && basename[0] == '.' {
		return true
	}
	return false
}

func walker(path string, info os.FileInfo, err error) error {
	if info.IsDir() {
		if isHidden(path) {
			return filepath.SkipDir
		}
		if info.Mode()&os.ModeSymlink == os.ModeSymlink {
			return filepath.SkipDir
		}
		return nil
	}

	if info.Mode().IsRegular() {
		wg.Add(1)
		go parseFile(path)
	}

	return nil
}

func main() {
	dictionary := flag.String("dictionary", "/usr/share/codespell/dictionary.txt", "Custom dictionary file that contains spelling corrections")
	flag.Parse()
	args := flag.Args()
	if len(args) == 0 {
		args = append(args, ".")
	}

	buildMisspellings(dictionary)

	for _, filename := range args {
		filepath.Walk(filename, walker)
	}
	wg.Wait()
}
