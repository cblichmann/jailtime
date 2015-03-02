/*
 * jailtime version 0.1
 * Copyright (c)2015 Christian Blichmann
 *
 * Chroot specification file parser.
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

package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
)

type SpecStatementType int

const (
	RegularFile = iota
	Directory
	SymLink
	HardLink
	Run
)

type SpecFileAttr struct {
	Uid  int
	Gid  int
	Mode int
}

type specStatement struct {
	Type SpecStatementType

	// Multiple paths only valid for Directory type
	Path []string

	// For SymLink and HardLink types this specifies the link target, for
	// RegularFile this specifies the destination in the chroot, for the
	// Run type this is the command to run from outside the chroot with the
	// working directory set to the chroot. Invalid for type Directory.
	Target string

	// File attributes, not valid for type Run. If not specified in the spec,
	// these default to root:root with mode 0755 for type Directory and
	// root:root with mode 0644 for RegularFile, SymLink and HardLink.
	SpecFileAttr
}

type SpecFile []specStatement

var (
	directivesRe = regexp.MustCompile("^(include|run)\\s+(.+)$")
	linkRe       = regexp.MustCompile("^(.*)\\s*(->|=>)\\s*(.*)$")
	dirRe        = regexp.MustCompile("^([^{]+)(?:{([^}]+)})?(.*)/$")
	fileRe       = regexp.MustCompile("^(.*?)(?:\\s+(.*))?$")
)

func (s *SpecFile) parseSpecLine(filename string, lineNo int, line string,
	includeDepth int) (*specStatement, error) {
	// Always strip white-space
	line = strings.TrimSpace(line)

	// Always skip blank lines and lines with single-line comments
	if len(line) == 0 || strings.HasPrefix(line, "#") {
		return nil, nil
	}

	// Handle directives
	if m := directivesRe.FindStringSubmatch(line); m != nil {
		switch m[1] {
		case "include":
			if err := s.parseFromFile(m[2], includeDepth+1); err != nil {
				return nil, err
			}
			return nil, nil
		case "run":
			return &specStatement{Type: Run, Target: m[2]}, nil
		}
	}

	if m := linkRe.FindStringSubmatch(line); m != nil {
		result := &specStatement{
			Path:   []string{strings.TrimSpace(m[1])},
			Target: m[3]}
		if m[2] == "->" {
			result.Type = SymLink
		} else {
			result.Type = HardLink
		}
		return result, nil
	}

	if m := dirRe.FindStringSubmatch(line); m != nil {
		result := &specStatement{Type: Directory}
		for _, comp := range strings.Split(m[2], ",") {
			result.Path = append(result.Path, m[1]+strings.TrimSpace(comp)+m[3])
		}
		if len(result.Path) == 0 {
			result.Path = []string{m[1] + m[3]}
		}
		return result, nil
	}

	// From here on we should only be left with regular files
	if m := fileRe.FindStringSubmatch(line); m != nil {
		result := &specStatement{Type: RegularFile, Path: []string{m[1]}}
		if len(m[2]) == 0 {
			result.Target = m[1]
		} else {
			result.Target = m[2]
		}
		return result, nil
	}
	return nil, fmt.Errorf("invalid spec statement: %s", line)
}

func (s *SpecFile) parseFromFile(filename string, includeDepth int) error {
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
		st, err := s.parseSpecLine(filename, lineNo, line, includeDepth)
		if err != nil {
			return err
		}
		if st != nil {
			*s = append(*s, *st)
		}
	}
	return nil
}

func OpenSpec(filename string) (SpecFile, error) {
	s := SpecFile{}
	if err := s.parseFromFile(filename, 0 /* Depth */); err != nil {
		return nil, err
	}
	return s, nil
}
