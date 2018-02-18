/*
 * jailtime version 0.6
 * Copyright (c)2015-2018 Christian Blichmann
 *
 * Specification file parser
 *
 * Redistribution and use in source and binary forms, with or without
 * modification, are permitted provided that the following conditions are met:
 *     * Redistributions of source code must retain the above copyright
 *       notice, this list of conditions and the following disclaimer.
 *     * Redistributions in binary form must reproduce the above copyright
 *       notice, this list of conditions and the following disclaimer in the
 *       documentation and/or other materials provided with the distribution.
 *
 * THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS"
 * AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE
 * IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE
 * ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT HOLDER OR CONTRIBUTORS BE
 * LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR
 * CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF
 * SUBSTITUTE GOODS OR SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS
 * INTERRUPTION) HOWEVER CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN
 * CONTRACT, STRICT LIABILITY, OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE)
 * ARISING IN ANY WAY OUT OF THE USE OF THIS SOFTWARE, EVEN IF ADVISED OF THE
 * POSSIBILITY OF SUCH DAMAGE.
 */

package spec // import "blichmann.eu/code/jailtime/spec"

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"syscall"
)

var (
	// TODO(cblichmann): Replace regexp matching with custom parsing code to
	//                   enable better error reporting.
	// Directives:
	//   include /some/file
	//   run echo 'test'
	directivesRe = regexp.MustCompile("^(include|run)\\s+(.+)$")

	// Links:
	//   /path/symlink_name -> /bin/bash
	//   /path/hardlink => /bin/bash
	linkRe = regexp.MustCompile("^(.*)\\s*(->|=>)\\s*(.*)$")

	// Directories:
	//   /some/dir/
	//   /var/lib/{all,of,these}/
	dirRe = regexp.MustCompile("^([^{]+)(?:{([^}]+)})?(.*)/$")

	// Device files:
	//  /dev/console c 5 1
	devRe = regexp.MustCompile("^(.*)\\s+([cbups])\\s+(\\d+)\\s+(\\d+)$")

	// Regular files:
	//  /usr/bin/python
	fileRe = regexp.MustCompile("^(.*?)(?:\\s+(.*))?$")
)

func parseSpecLine(filename string, lineNo int, line string,
	includeDepth int) (lineStmts Statements, err error) {
	// Always strip white-space
	line = strings.TrimSpace(line)

	// Always skip blank lines and lines with single-line comments
	if len(line) == 0 || strings.HasPrefix(line, "#") {
		return
	}

	if m := directivesRe.FindStringSubmatch(line); m != nil {
		switch m[1] {
		case "include":
			lineStmts, err = parseFromFile(m[2], includeDepth+1)
		case "run":
			lineStmts = Statements{NewRun(m[2])}
		}
	} else if m := linkRe.FindStringSubmatch(line); m != nil {
		lineStmts = Statements{NewLink(m[3], strings.TrimSpace(m[1]),
			m[2] == "=>")}
	} else if m := dirRe.FindStringSubmatch(line); m != nil {
		comps := strings.Split(m[2], ",")
		lineStmts = make(Statements, len(comps))
		for i, comp := range comps {
			lineStmts[i] = NewDirectory(m[1] + strings.TrimSpace(comp) + m[3])
		}
	} else if m := devRe.FindStringSubmatch(line); m != nil {
		type_ := 0
		switch m[2][0] {
		case 'c':
			fallthrough
		case 'u':
			type_ = syscall.S_IFCHR
		case 'b':
			type_ = syscall.S_IFBLK
		case 'p':
			type_ = syscall.S_IFIFO
		case 's':
			type_ = syscall.S_IFSOCK
		}
		major, _ := strconv.Atoi(m[3])
		minor, _ := strconv.Atoi(m[3])
		d := NewDevice(m[1], type_, major, minor)
		lineStmts = Statements{d}
	} else if m := fileRe.FindStringSubmatch(line); m != nil {
		// From here on we should only be left with regular files
		f := RegularFile{source: m[1]}
		if len(m[2]) == 0 {
			f.target = m[1]
		} else {
			f.target = m[2]
		}
		lineStmts = Statements{f}
	} else {
		err = fmt.Errorf("%s:%d: invalid spec statement: %s", filename,
			lineNo, line)
	}
	return
}

func parseFromFile(filename string, includeDepth int) (stmts Statements,
	err error) {
	if includeDepth > 8 {
		err = fmt.Errorf("nesting level too deep, including: %s", filename)
		return
	}

	f, err := os.Open(filename)
	if err != nil {
		return
	}
	defer f.Close()

	s := bufio.NewScanner(f)
	var line string
	var lineStmts Statements
	lineNo := 0
	for s.Scan() {
		lineNo++
		line = s.Text()
		lineStmts, err = parseSpecLine(filename, lineNo, line, includeDepth)
		if err != nil {
			return
		}
		if len(lineStmts) > 0 {
			stmts = append(stmts, lineStmts...)
		}
	}
	//s.Err() will return nil if the scanner encountered io.EOF without other
	//errors
	err = s.Err()
	return
}

func Parse(filename string) (Statements, error) {
	return parseFromFile(filename, 0 /* Include depth */)
}
