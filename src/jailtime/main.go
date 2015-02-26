// +build linux !windows !cgo

/*
 * jailtime version 0.1
 * Copyright (c)2015 Christian Blichmann
 *
 * Create and maintain chroot jails
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
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

const (
	VersionMajor = 0
	VersionMinor = 1
)

var (
	help    = flag.Bool("help", false, "display this help and exit")
	version = flag.Bool("version", false, "display version and exit")
	link    = flag.Bool("link", false, "hard ling files instead of copying")
	verbose = flag.Bool("verbose", false, "explain what is being done")
)

func fatalln(v ...interface{}) {
	fmt.Fprint(os.Stderr, "jailtime: ")
	fmt.Fprintln(os.Stderr, v...)
	os.Exit(1)
}

func printUsage() {
	fmt.Printf("Usage: jailtime [OPTION]... FILE...\n" +
		"Create or update chroot environments read from specifications in " +
		"FILEs.\n\n")
	flag.VisitAll(func(f *flag.Flag) {
		fmt.Printf("      --%-23s %s\n", f.Name, f.Usage)
	})
	fmt.Printf("\nFor bug reporting instructions, please see:\n" +
		"<https://github.com/cblichmann/jailtime/issues>\n")
}

func processCommandLine() {
	flag.Usage = func() {}
	flag.Parse()
	if *help {
		printUsage()
		return
	}
	if *version {
		fmt.Printf("jailtime %d.%d\n"+
			"Copyright (c)2015 Christian Blichmann\n"+
			"This software is BSD licensed, see the source for copying "+
			"conditions.\n\n", VersionMajor, VersionMinor)
		os.Exit(0)
	}
	if flag.NArg() == 0 {
		fatalln("missing operand")
	}
}

func importedLibraries(binary string) (deps []string, err error) {
	// Do not wait for the loader to return an error on non-existing files.
	_, err = os.Stat(binary)
	if os.IsNotExist(err) {
		return
	}
	cmd := exec.Command(LoaderExecutable, "--list", binary)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return
	}
	if err = cmd.Start(); err != nil {
		return
	}
	r := bufio.NewReader(stdout)
	deps = make([]string, 0, 10)
	var line string
	for {
		line, err = r.ReadString('\n')
		if err == io.EOF {
			break
		} else if err != nil {
			return
		}
		parts := strings.SplitN(line, "=>", 2)
		if len(parts) != 2 {
			continue
		}
		parts = strings.SplitN(parts[1], "(", 2)
		if len(parts) != 2 {
			fatalln("bug: OS loader returned unexpected format")
		}
		deps = append(deps, strings.TrimSpace(parts[0]))
	}
	err = cmd.Wait()
	return
}

type SpecStatementType int

const (
	RegularFile = iota
	Directory
	SymLink
	HardLink
	Run
)

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
	// These default to root:root with mode 0755 for type Directory and
	// root:root with mode 0644 for RegularFile, SymLink and HardLink.
	Uid  int
	Gid  int
	Mode int
}

type SpecFile []specStatement

func (s *SpecFile) parseSpecLine(filename string, lineNo int, line string,
	includeDepth int) (*specStatement, error) {
	// Always strip white-space
	line = strings.TrimSpace(line)

	// Always skip blank lines and lines with single-line comments
	if len(line) == 0 || strings.HasPrefix(line, "#") {
		return nil, nil
	}

	// Handle directives
	dirRe := regexp.MustCompile("^\\s*(include|run)\\s+(.+)$")
	if m := dirRe.FindStringSubmatch(line); len(m) > 0 {
		switch m[1] {
		case "include":
			if err := s.parseFromFile(m[2], includeDepth+1); err != nil {
				return nil, err
			}
		case "run":
			return &specStatement{Type: Run, Target: m[2]}, nil
		}
	}

	re := regexp.MustCompile("^(.*)\\s*(->|=>)\\s*(.*)$")
	if m := re.FindStringSubmatch(line); len(m) > 0 {
		fmt.Printf("L|%s|%d\n", strings.Join(m, "|"), len(m))
		return nil, nil
	}

	re2 := regexp.MustCompile("^(.*)({[^}]+})?(.*)/$")
	if m := re2.FindStringSubmatch(line); len(m) > 0 {
		fmt.Printf("D|%s|%d\n", strings.Join(m, "|"), len(m))
		return nil, nil
	}

	fmt.Printf("%s:%d: %s\n", filename, lineNo, strings.TrimSpace(line))
	return nil, nil
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
		_, err := s.parseSpecLine(filename, lineNo, line, includeDepth)
		if err != nil {
			return err
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

func main() {
	processCommandLine()

	_, err := OpenSpec("basic_shell")
	if err != nil {
		fatalln(err)
	}

	//	deps, err := importedLibraries("/bin/bash")
	//	if err != nil {
	//		fatalln(err)
	//	}
	//
	//	for _, d := range deps {
	//		fmt.Println(d)
	//	}
}
