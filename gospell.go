package main

// fixes misspelings
// Abandonned  ==> Abandoned
// based on codespell

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"github.com/kr/fs"
	"os"
	"path/filepath"
	"runtime"
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

func isBinary(file *os.File) bool {
	data := make([]byte, 1024)
	count, _ := file.Read(data)
	file.Seek(0, 0)
	if bytes.Index(data[:count], []byte("\x00")) != -1 {
		return true
	}
	return false
}

func parseFile(filename string) {
	var word, lword string
	defer wg.Done()
	f, _ := os.Open(filename)
	defer f.Close()

	// skip binary files
	if isBinary(f) {
		return
	}

	s := bufio.NewScanner(f)
	s.Split(bufio.ScanWords)
	for s.Scan() {
		word = s.Text()
		lword = strings.ToLower(word)
		if misspelling, ok := misspellings[lword]; ok {
			if misspelling.fix {
				fixword := fixCase(word, misspelling.data)
				fmt.Printf("%s: \033[31m%s\033[0m ==> \033[32m%s\033[0m\n", filename, word, fixword)
			}
		}
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

func visit(filename string) error {
	walker := fs.Walk(filename)
	for walker.Step() {
		if err := walker.Err(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			continue
		}
		path := walker.Path()
		info := walker.Stat()
		if info.IsDir() {
			if isHidden(path) {
				walker.SkipDir()
			}
		}
		if info.Mode()&os.ModeSymlink == os.ModeSymlink {
			walker.SkipDir()
		}
		if info.Mode().IsRegular() {
			wg.Add(1)
			go parseFile(path)
		}
	}
	return nil
}

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	dictionary := flag.String("dictionary", "dictionary.txt", "Custom dictionary file that contains spelling corrections")
	flag.Parse()
	args := flag.Args()
	if len(args) == 0 {
		args = append(args, ".")
	}

	buildMisspellings(dictionary)

	for _, filename := range args {
		visit(filename)
	}
	wg.Wait()
}
