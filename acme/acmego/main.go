// Copyright 2014 The Go Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Acmego watches acme for .go files being written.
// Each time a .go file is written, acmego checks whether the
// import block needs adjustment. If so, it makes the changes
// in the window body but does not write the file.
package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"9fans.net/go/acme"
)

func main() {
	fmts := newFmts()
	l, err := acme.Log()
	if err != nil {
		log.Fatal(err)
	}

	for {
		event, err := l.Read()
		if err != nil {
			log.Fatal(err)
		}
		modified := false
		anyextFmtUsed := false
		if event.Name != "" && event.Op == "put" {
			if fmter, ok := fmts[fileExt(event.Name)]; ok {
				modified = reformat(event.ID, event.Name, fmter)
			} else {
				anyextFmtUsed = true
				modified = reformat(event.ID, event.Name, fmts["anyext"])
			}
			if !modified || anyextFmtUsed {
				output, _ := exec.Command("bl2plus", event.Name).CombinedOutput()
				fmt.Fprintf(os.Stderr, "%s", output)
			}
		}
	}
}

func fileExt(filePath string) string {
	if n := strings.LastIndex(filePath, "."); n != -1 {
		return filePath[n+1:]
	}
	return ""
}

func reformat(id int, name string, fmter Formatter) bool {
	win, err := acme.Open(id, nil)
	if err != nil {
		log.Print(err)
		return false
	}
	w := Window{win, false}
	defer w.CloseFiles()

	old, err := ioutil.ReadFile(name)
	if err != nil {
		//log.Print(err)
		return false
	}
	new, err := fmter.format(name)
	if err != nil {
		return false
	}

	if bytes.Equal(old, new) {
		return false
	}

	f, err := ioutil.TempFile("", "acmego")
	if err != nil {
		log.Print(err)
		return false
	}
	if _, err := f.Write(new); err != nil {
		log.Print(err)
		return false
	}
	tmp := f.Name()
	f.Close()
	defer os.Remove(tmp)

	diff, _ := exec.Command("/usr/bin/diff", name, tmp).CombinedOutput()

	latest, err := w.ReadAll("body")
	if err != nil {
		log.Print(err)
		return false
	}
	if !bytes.Equal(old, latest) {
		log.Printf("skipped update to %s: window modified since Put\n", name, len(old), len(latest))
		return false
	}

	w.Write("ctl", []byte("mark"))
	w.Write("ctl", []byte("nomark"))
	diffLines := strings.Split(string(diff), "\n")
	for i := len(diffLines) - 1; i >= 0; i-- {
		line := diffLines[i]
		if line == "" {
			continue
		}
		if line == `\ No newline at end of file` {
			w.Addr("$")
			w.Write("data", []byte("\n"))
			continue
		}
		if line[0] == '<' || line[0] == '-' || line[0] == '>' {
			continue
		}
		j := 0
		for j < len(line) && line[j] != 'a' && line[j] != 'c' && line[j] != 'd' {
			j++
		}
		if j >= len(line) {
			log.Printf("cannot parse diff line: %q", line)
			break
		}
		oldStart, oldEnd := parseSpan(line[:j])
		newStart, newEnd := parseSpan(line[j+1:])
		if oldStart == 0 || newStart == 0 {
			continue
		}
		switch line[j] {
		case 'a':
			err := w.Addr("%d+#0", oldStart)
			if err != nil {
				log.Print(err)
				break
			}
			w.Write("data", findLines(new, newStart, newEnd))
		case 'c':
			err := w.Addr("%d,%d", oldStart, oldEnd)
			if err != nil {
				log.Print(err)
				break
			}

			w.Write("data", findLines(new, newStart, newEnd))
		case 'd':
			err := w.Addr("%d,%d", oldStart, oldEnd)
			if err != nil {
				log.Print(err)
				break
			}

			w.Write("data", nil)
		}
	}
	return w.modified
}

// Encapsulates an Acme window along with its current state, modified or not.
// This will allow us to execute additional fmt tools like bl2plus once the
// window has been saved (not modified) and the original formatter has done
// its job.
type Window struct {
	*acme.Win
	modified bool
}

func (w *Window) Write(ftype string, data []byte) {
	w.Win.Write(ftype, data)
	w.modified = true
}

func parseSpan(text string) (start, end int) {
	i := strings.Index(text, ",")
	if i < 0 {
		n, err := strconv.Atoi(text)
		if err != nil {
			log.Printf("cannot parse span %q", text)
			return 0, 0
		}
		return n, n
	}
	start, err1 := strconv.Atoi(text[:i])
	end, err2 := strconv.Atoi(text[i+1:])
	if err1 != nil || err2 != nil {
		log.Printf("cannot parse span %q", text)
		return 0, 0
	}
	return start, end
}

func findLines(text []byte, start, end int) []byte {
	i := 0

	start--
	for ; i < len(text) && start > 0; i++ {
		if text[i] == '\n' {
			start--
			end--
		}
	}
	startByte := i
	for ; i < len(text) && end > 0; i++ {
		if text[i] == '\n' {
			end--
		}
	}
	endByte := i
	return text[startByte:endByte]
}
