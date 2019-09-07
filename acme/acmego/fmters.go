package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

type Formatter interface {
	format(string) ([]byte, error)
}

func buildCmd(cmd, file string, args ...string) *exec.Cmd {
	if len(args) > 0 {
		args = append(args, file)
		return exec.Command(cmd, args...)
	}
	return exec.Command(cmd, file)
}

type GoImportFmt struct {
	cmd string
}

func (g *GoImportFmt) format(file string) ([]byte, error) {
	cmd := buildCmd(g.cmd, file)
	// Grab the parent directory of the file where we are going to execute
	// the command.
	cmd.Dir = filepath.Dir(file)
	new, err := cmd.CombinedOutput()
	if err != nil {
		// Probably a syntax error, use the compiler for better message.
		// For now use 'go build file.go' and strip the package header.
		// We run it in /var/run so that paths do not get shortened
		// (assuming /var/run exists and no one is editing go files under that path).
		// A better fix to both would be to use go tool 6g, but we don't know
		// whether 6g is the right architecture. Could parse 'go env' output.
		// Or maybe the go command should have 'go tool compile' and 'go tool link'.
		cmd := exec.Command("go", "build", file)
		cmd.Dir = "/var/run"
		out, _ := cmd.CombinedOutput()
		start := []byte("# command-line-arguments\n")
		if !bytes.HasPrefix(out, start) {
			fmt.Fprintf(os.Stderr, "goimports %s: %v\n%s", file, err, new)
			return new, err
		}
		fmt.Fprintf(os.Stderr, "%s", out)
	}
	return new, err
}

type PyFmt struct {
	cmd string
}

func (py *PyFmt) format(file string) ([]byte, error) {
	cmd := buildCmd(py.cmd, file)
	new, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Fprintf(os.Stderr, "yapf %s: %v\n%s", file, err, new)
	}
	return new, err
}

type RustFmt struct {
	cmd string
}

func (rs *RustFmt) format(file string) ([]byte, error) {
	cmd := buildCmd(rs.cmd, file)
	new, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s %s: %v\n%s", rs.cmd, file, err, new)
	}
	return new, err
}

// Default formatter adds an end of line at the end of file
// This formatter makes use of the executable implemented in
// https://github.com/jordilin/aeol
type DefaultEolFmt struct {
	cmd string
}

func (df *DefaultEolFmt) format(file string) ([]byte, error) {
	cmd := buildCmd(df.cmd, file)
	new, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Fprintf(os.Stderr, "default fmt eol %s: %v\n%s", file, err, new)
	}
	return new, err
}

type ElmFmt struct {
	cmd string
}

func (el *ElmFmt) format(file string) ([]byte, error) {
	cmd := buildCmd(el.cmd, file)
	new, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s %s: %v\n%s", el.cmd, file, err, new)
	}
	return new, err
}

func newFmts() map[string]Formatter {
	gofmt := &GoImportFmt{cmd: "goimports"}
	pyfmt := &PyFmt{cmd: "yapf"}
	rustfmt := &RustFmt{cmd: "fmtrust"}
	defaultfmt := &DefaultEolFmt{cmd: "aeol"}
	elmfmt := &ElmFmt{cmd: "elmfmt"}
	fmts := make(map[string]Formatter)
	fmts["py"] = pyfmt
	fmts["go"] = gofmt
	fmts["rs"] = rustfmt
	fmts["elm"] = elmfmt
	fmts["anyext"] = defaultfmt
	return fmts
}
