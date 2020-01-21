/*
 * jailtime version 0.8
 * Copyright (c)2015-2020 Christian Blichmann
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

package spec

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
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
	linkRe = regexp.MustCompile("^(.+)\\s*(->|=>)\\s*(.+)$")

	// Directories:
	//   /some/dir/
	//   /var/lib/{all,of,these}/
	//   /home/user/ 600
	dirRe = regexp.MustCompile("^([^{]+)(?:{([^}]+)})?(.*)/(?:\\s+(\\d+))?$")

	// Device files:
	//  /dev/null c 1 3 666
	//  /dev/console c 5 1
	devRe = regexp.MustCompile(
		"^(.+)\\s+([cbups])\\s+(\\d+)\\s+(\\d+)(?:\\s+(\\d+))?$")

	// Regular files:
	//  /bin/bash              # Copy to /bin/bash, original permissions
	//  /bin/dash /bin/sh      # Copy to /bin/sh, original permissions
	//  /usr/bin/python 755    # File mode is 755
	// Special cases:
	//  /Users/John\ Doe/cfg.txt /private/etc/motd 644  # Escaping, mode 644
	//  /tmp/cache755 /755     # File name is "755" in chroot dir
	//  /tmp/cache755 755 755  # File name is "755" in chroot dir, mode 755
	fileRe = regexp.MustCompile("^(.+?)(?:\\s+(.+?))?(?:\\s+(\\d+))?$")
)

// parseMode parses an octal file mode into a positive integer. Returns -1 on
// error.
func parseMode(s string) int {
	if mode, err := strconv.ParseInt(s, 8, 16); err == nil {
		return int(mode)
	}
	return -1
}

func parseSpecLine(filename string, lineNo int, line string,
	includer func(filename string) (Statements, error)) (
	lineStmts Statements, err error) {
	// Always strip white-space
	line = strings.TrimSpace(line)

	// Always skip blank lines and lines with single-line comments
	if len(line) == 0 || strings.HasPrefix(line, "#") {
		return
	}

	if m := directivesRe.FindStringSubmatch(line); m != nil {
		switch m[1] {
		case "include":
			if includer != nil {
				lineStmts, err = includer(m[2])
			}
		case "run":
			lineStmts = Statements{NewRun(m[2])}
		}
	} else if m := linkRe.FindStringSubmatch(line); m != nil {
		lineStmts = Statements{NewLink(m[3], strings.TrimSpace(m[1]),
			m[2] == "=>")}
	} else if m := dirRe.FindStringSubmatch(line); m != nil {
		mode := 0755
		if rawMode := m[4]; len(rawMode) > 0 {
			if mode = parseMode(rawMode); mode < 0 {
				return nil, fmt.Errorf("%s:%d: invalid directory mode: %s",
					filename, lineNo, rawMode)
			}
		}
		comps := strings.Split(m[2], ",")
		lineStmts = make(Statements, len(comps))
		for i, comp := range comps {
			d := NewDirectory(m[1] + strings.TrimSpace(comp) + m[3])
			d.fileAttr.Mode = mode
			lineStmts[i] = d
		}
	} else if m := devRe.FindStringSubmatch(line); m != nil {
		source := m[1]
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
		minor, _ := strconv.Atoi(m[4])
		mode := FileModeUnspecified
		if rawMode := m[5]; len(rawMode) > 0 {
			if mode = parseMode(rawMode); mode < 0 {
				return nil, fmt.Errorf("%s:%d: invalid file mode: %s", filename,
					lineNo, rawMode)
			}
		}
		d := NewDevice(source, type_, major, minor)
		d.fileAttr.Mode = mode
		lineStmts = Statements{d}
	} else if m := fileRe.FindStringSubmatch(line); m != nil {
		// From here on we should only be left with regular files
		source, target, rawMode := m[1], m[2], m[3]
		mode := FileModeUnspecified
		if len(rawMode) > 0 { // Three arguments
			if mode = parseMode(rawMode); mode < 0 {
				return nil, fmt.Errorf("%s:%d: invalid file mode: %s", filename,
					lineNo, rawMode)
			}
		} else if len(target) == 0 {
			target = source
		} else if mode = parseMode(target); mode >= 0 {
			// Two, but second parses as number
			target = source
		}
		f := NewRegularFile(source, target)
		f.fileAttr.Mode = mode
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

	fromDir, err := filepath.EvalSymlinks(filename)
	if err != nil {
		return
	}
	if fromDir, err = filepath.Abs(fromDir); err != nil {
		return
	}
	fromDir = filepath.Dir(fromDir)

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
		lineStmts, err = parseSpecLine(filename, lineNo, line,
			func(filename string) (Statements, error) {
				return parseFromFile(filepath.Join(fromDir, filename),
					includeDepth+1)
			})
		if err != nil {
			return
		}
		if len(lineStmts) > 0 {
			stmts = append(stmts, lineStmts...)
		}
	}
	// s.Err() will return nil if the scanner encountered io.EOF without other
	// errors
	err = s.Err()
	return
}

// Parse parses a jailspec file, resolving all include directives. On success,
// it returns a list of statements and a nil error. Otherwise it returns nil
// for the list and the encountered error.
func Parse(filename string) (Statements, error) {
	return parseFromFile(filename, 0 /* Include depth */)
}
