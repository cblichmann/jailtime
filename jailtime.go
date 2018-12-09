/*
 * jailtime version 0.7
 * Copyright (c)2015-2018 Christian Blichmann
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
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"blichmann.eu/code/jailtime/action"
	"blichmann.eu/code/jailtime/copy"
	"blichmann.eu/code/jailtime/loader"
	"blichmann.eu/code/jailtime/spec"
)

var (
	help  = flag.Bool("help", false, "display this help and exit")
	link  = flag.Bool("link", false, "hard link files instead of copying")
	force = flag.Bool("force", false, "if an existing destination file cannot "+
		"be\n"+
		"                                  opened, remove it and try again")
	removeDestination = flag.Bool("remove-destination", false, "remove each "+
		"existing destination file before\n"+
		"                                  attempting to open it (contrast "+
		"with --force)")
	reflink = flag.Bool("reflink", false, "perform lightweight copies using "+
		"CoW")
	verbose = flag.Bool("verbose", false, "explain what is being done")
	dryRun  = flag.Bool("dry-run", false, "don't do anything, just print "+
		"(implies --verbose)")
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
		fmt.Printf("jailtime 0.7\n" +
			"Copyright (c)2015-2018 Christian Blichmann\n" +
			"This software is BSD licensed, see the LICENSE file for " +
			"details.\n\n")
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
	if *dryRun {
		// --dry-run implies verbose
		*verbose = true
	}
}

func expandWithDependencies(stmts spec.Statements) spec.Statements {
	expanded := spec.ExpandLexical(stmts)
	for _, s := range expanded {
		switch stmt := s.(type) {
		case spec.RegularFile:
			deps, err := loader.ImportedLibraries(stmt.Source())
			if err != nil {
				fatalf("%s\n", err)
			}
			attr := stmt.FileAttr()
			for _, d := range deps {
				f := spec.NewRegularFile(d, d)
				*f.FileAttr() = *attr
				expanded = append(expanded, f)
			}
		}
	}
	return expanded
}

func updateChroot(chrootDir string, stmts spec.Statements) (err error) {
	reflinkOpt := copy.ReflinkNo
	if *reflink {
		reflinkOpt = copy.ReflinkAlways
	}
	for _, s := range spec.ExpandLexical(expandWithDependencies(stmts)) {
		target := filepath.Join(chrootDir, s.Target())
		if *verbose {
			fmt.Println(s.Verbose())
			if *dryRun {
				continue
			}
		}
		switch stmt := s.(type) {
		case spec.Directory:
			err = action.Directory(target, stmt)
		case spec.RegularFile:
			err = action.RegularFile(target, stmt, &copy.Options{
				Force:             *force,
				Reflink:           reflinkOpt,
				RemoveDestination: *removeDestination,
			})
		case spec.Link:
			err = action.Link(target, stmt)
		case spec.Device:
			err = action.Device(target, stmt)
		case spec.Run:
			err = action.Run(target, stmt, chrootDir)
		}
		if err != nil {
			return
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

	if err := updateChroot(flag.Arg(lastArg), stmts); err != nil {
		fatalf("%s\n", err)
	}
}
