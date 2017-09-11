/*
 * jailtime version 0.6
 * Copyright (c)2015-2017 Christian Blichmann
 *
 * Create and manage chroot/jail environments
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

package main

import (
	"blichmann.eu/code/jailtime/copy"
	"blichmann.eu/code/jailtime/loader"
	"blichmann.eu/code/jailtime/spec"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path"
	"sort"
	"syscall"
)

var (
	help  = flag.Bool("help", false, "display this help and exit")
	link  = flag.Bool("link", false, "hard link files instead of copying")
	force = flag.Bool("force", false, "if an existing destination file cannot "+
		"be\n"+
		"                                  opened, remove it and try again")
	removeDestination = flag.Bool("remove-destination", false, "remove each "+
		"existing destination file before attempting to open it (contrast "+
		"with --force)")
	reflink = flag.Bool("reflink", false, "perform lightweight copies using "+
		"CoW")
	verbose = flag.Bool("verbose", false, "explain what is being done")
	version = flag.Bool("version", false, "display version and exit")
	// TODO(cblichmann): Implement these
	//noClobber = flag.Bool("no-clobber", false, "do not overwrite existing "+
	//	"files")
	//oneFilesystem = flag.Bool("one-filesystem", false, "do not cross "+
	//	"filesystem boundaries")
)

func fatalf(format string, v ...interface{}) {
	fmt.Fprintf(os.Stderr, "jailtime: "+format, v...)
	os.Exit(1)
}

// Prints more GNU-looking usage text.
func printUsage() {
	fmt.Printf("Usage: %s [OPTION]... FILE... TARGET\n"+
		"Create or update the chroot environment in TARGET using "+
		"specification\n"+
		"FILEs. TARGET should be a directory and is created if it does not\n"+
		"exist.\n\n", os.Args[0])
	flag.VisitAll(func(f *flag.Flag) {
		fmt.Printf("      --%-23s %s\n", f.Name, f.Usage)
	})
	fmt.Printf("\nFor bug reporting instructions, please see:\n" +
		"<https://github.com/cblichmann/jailtime/issues>\n")
}

func processCommandLine() {
	flag.Usage = func() {}

	// Parse command-line flags twice to work around the issue that the flag
	// package stops parsing after the first non-option
	flag.Parse()
	args := flag.Args()
	flagPos := 0
	for i, arg := range args {
		if arg[0:2] == "--" {
			args[i], args[flagPos] = args[flagPos], args[i]
			flagPos++
		}
	}
	flag.CommandLine.Parse(flag.Args())

	if *help {
		printUsage()
		os.Exit(0)
	}
	if *version {
		fmt.Printf("jailtime 0.5\n" +
			"Copyright (c)2015-2017 Christian Blichmann\n" +
			"This software is BSD licensed, see the source for copying " +
			"conditions.\n\n")
		os.Exit(0)
	}
	fatalHelp := fmt.Sprintf("Try '%s' --help for more information.",
		os.Args[0])
	if flag.NArg() == 0 {
		fatalf("missing file operand\n%s\n", fatalHelp)
	}
	if flag.NArg() == 1 {
		fatalf("missing destination operand after '%s'\n%s\n", flag.Arg(0),
			fatalHelp)
	}
}

func ExpandLexical(stmts spec.Statements) spec.Statements {
	todo := make(map[string]bool)
	// Expect at least half of the files to expand at least to its dir
	expanded := make(spec.Statements, 0, 3*len(stmts)/2)
	for _, s := range stmts {
		var dir string
		switch stmt := s.(type) {
		case spec.Directory:
			dir = stmt.Target()
			expanded = append(expanded, stmt)
		case spec.Run:
			// Do not deduplicate run statements
			expanded = append(expanded, stmt)
			continue
		}
		target := s.Target()
		if _, ok := todo[target]; ok {
			continue
		}
		todo[target] = true
		if dir != target {
			expanded = append(expanded, s)
			dir = path.Dir(target)
		}
		for dirLen := 0; dirLen != len(dir); {
			if _, ok := todo[dir]; !ok {
				expanded = append(expanded, spec.NewDirectory(dir))
				todo[dir] = true
			}
			dirLen = len(dir)
			dir = path.Dir(dir)
		}
	}
	sort.Sort(expanded)
	return expanded
}

func ExpandWithDependencies(stmts spec.Statements) spec.Statements {
	expanded := ExpandLexical(stmts)
	for _, s := range expanded {
		switch stmt := s.(type) {
		case spec.RegularFile:
			deps, err := loader.ImportedLibraries(stmt.Source())
			if err != nil {
				fatalf("%s\n", err)
			}
			for _, d := range deps {
				expanded = append(expanded, spec.NewRegularFile(d, d))
			}
		}
	}
	return ExpandLexical(expanded)
}

func MakeDev(major, minor int) int {
	// Taken from glibc's sys/sysmacros.h
	return int(uint64(minor)&0xFF |
		(uint64(major)&0xFFF)<<8 |
		(uint64(minor) & ^uint64(0xFF))<<12 |
		(uint64(major) & ^uint64(0xFFF))<<32)
}

func UpdateChroot(chrootDir string, stmts spec.Statements) (err error) {
	reflinkOpt := copy.ReflinkNo
	if *reflink {
		reflinkOpt = copy.ReflinkAlways
	}
	for _, s := range ExpandWithDependencies(stmts) {
		target := path.Join(chrootDir, s.Target())
		switch stmt := s.(type) {
		case spec.Directory:
			if *verbose {
				fmt.Printf("create dir: %s\n", target)
			}
			if err = os.MkdirAll(target, 0755); err != nil {
				return
			}
		case spec.RegularFile:
			if *verbose {
				fmt.Printf("copy file: %s > %s\n", stmt.Source(), target)
			}
			if _, err = copy.File(stmt.Source(), target, &copy.Options{
				Force:             *force,
				Reflink:           reflinkOpt,
				RemoveDestination: *removeDestination,
			}); err != nil {
				return
			}
		case spec.Link:
			linkName := stmt.Source()
			var action string
			var arrow string
			if stmt.HardLink() {
				action = "create hardlink"
				arrow = "=>"
			} else {
				action = "create symlink"
				arrow = "->"
			}
			if *verbose {
				fmt.Printf("%s: %s %s %s\n", action, target, arrow, linkName)
			}
			if _, err = os.Stat(target); err == nil { // Link exists
				if err = os.Remove(target); err != nil {
					return
				}
			}
			if stmt.HardLink() {
				err = os.Link(linkName, target)
			} else {
				err = os.Symlink(linkName, target)
			}
			if err != nil {
				return
			}
		case spec.Device:
			if *verbose {
				fmt.Printf("create device: %s\n", target)
			}
			if _, err = os.Stat(target); err == nil { // Device exists
				if err = os.Remove(target); err != nil {
					return
				}
			}
			if err = syscall.Mknod(target, uint32(stmt.Type()|0644), MakeDev(
				stmt.Major(), stmt.Minor())); err != nil {
				return
			}
		case spec.Run:
			if *verbose {
				fmt.Printf("run in %s: %s\n", chrootDir, stmt.Command())
			}
			cmd := exec.Command("/bin/sh", "-c", stmt.Command())
			cmd.Dir = chrootDir
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			if err = cmd.Run(); err != nil {
				return
			}
		}
	}
	return
}

func main() {
	processCommandLine()

	// Parse all spec files given on the command-line
	stmts := spec.Statements{}
	lastArg := flag.NArg() - 1
	for _, s := range flag.Args()[:lastArg] {
		parsed, err := spec.Parse(s)
		if err != nil {
			fatalf("%s\n", err)
		}
		stmts = append(stmts, parsed...)
	}

	if err := UpdateChroot(flag.Arg(lastArg), stmts); err != nil {
		fatalf("%s\n", err)
	}
}
