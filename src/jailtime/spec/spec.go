/*
 * jailtime version 0.1
 * Copyright (c)2015 Christian Blichmann
 *
 * Chroot specification file parser
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
 * ARE DISCLAIMED. IN NO EVENT SHALL CHRISTIAN BLICHMANN BE LIABLE FOR ANY
 * DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES
 * (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES;
 * LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND
 * ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
 * (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF
 * THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
 */

package spec

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"
)

type StatementType int

const (
	RegularFile = iota
	Device      // TODO(cblichmann): Not implemented
	Directory
	SymLink
	HardLink
	Run
)

type FileAttr struct {
	Uid  int
	Gid  int
	Mode int
}

type DeviceAttr struct {
	Type  int
	Major int
	Minor int
}

type Statement struct {
	Type StatementType

	// Source path. Unset for types Run and Device.
	Source string

	// For SymLink and HardLink types this specifies the link target, for
	// RegularFile and Device this specifies the destination in the chroot, for
	// the Run type this is the command to run from outside the chroot with the
	// working directory set to the chroot. Invalid for type Directory.
	Target string

	// File attributes, not valid for type Run. If not specified in the spec,
	// these default to root:root with mode 0755 for type Directory and
	// root:root with mode 0644 for RegularFile, SymLink and HardLink.
	FileAttr

	// Specifies the major and minor number for a device file to create. Valid
	// for type Device only.
	DeviceAttr
}

type JailSpec []Statement

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
	devRe = regexp.MustCompile("^(.*)\\s+([cbup])\\s+(\\d+)\\s+(\\d+)$")

	// Regular files:
	//  /usr/bin/python
	fileRe = regexp.MustCompile("^(.*?)(?:\\s+(.*))?$")
)

func (j *JailSpec) parseSpecLine(filename string, lineNo int, line string,
	includeDepth int) (results []Statement, err error) {
	// Always strip white-space
	line = strings.TrimSpace(line)

	// Always skip blank lines and lines with single-line comments
	if len(line) == 0 || strings.HasPrefix(line, "#") {
		return
	}

	results = make([]Statement, 1)
	r := &results[0]

	if m := directivesRe.FindStringSubmatch(line); m != nil {
		switch m[1] {
		case "include":
			results = nil
			err = j.parseFromFile(m[2], includeDepth+1)
		case "run":
			*r = Statement{Type: Run, Target: m[2]}
		}
	} else if m := linkRe.FindStringSubmatch(line); m != nil {
		*r = Statement{
			Source: strings.TrimSpace(m[1]),
			Target: m[3]}
		if m[2] == "->" {
			r.Type = SymLink
		} else {
			r.Type = HardLink
		}
	} else if m := dirRe.FindStringSubmatch(line); m != nil {
		comps := strings.Split(m[2], ",")
		results = make([]Statement, len(comps))
		for i, comp := range comps {
			results[i] = Statement{
				Type:   Directory,
				Source: m[1] + strings.TrimSpace(comp) + m[3]}
		}
	} else if m := devRe.FindStringSubmatch(line); m != nil {
		*r = Statement{Type: Device, Target: m[1]}
		r.DeviceAttr.Type = int(m[2][0])
		r.DeviceAttr.Major, _ = strconv.Atoi(m[3])
		r.DeviceAttr.Minor, _ = strconv.Atoi(m[4])
	} else if m := fileRe.FindStringSubmatch(line); m != nil {
		// From here on we should only be left with regular files
		*r = Statement{Type: RegularFile, Source: m[1]}
		if len(m[2]) == 0 {
			r.Target = m[1]
		} else {
			r.Target = m[2]
		}
	} else {
		return nil, fmt.Errorf("%s:%d: invalid spec statement: %s", filename,
			lineNo, line)
	}
	return
}

func (j *JailSpec) parseFromFile(filename string, includeDepth int) error {
	if includeDepth > 8 {
		return fmt.Errorf("nesting level too deep while including: %s",
			filename)
	}

	f, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	r := bufio.NewReader(f)
	var line string
	lineNo := 0
	for {
		line, err = r.ReadString('\n')
		if err == io.EOF {
			break
		} else if err != nil {
			return err
		}
		lineNo++
		st, err := j.parseSpecLine(filename, lineNo, line, includeDepth)
		if err != nil {
			return err
		}
		if len(st) > 0 {
			*j = append(*j, st...)
		}
	}
	return nil
}

func Parse(filename string) (JailSpec, error) {
	j := JailSpec{}
	if err := j.parseFromFile(filename, 0 /* Depth */); err != nil {
		return nil, err
	}
	return j, nil
}
