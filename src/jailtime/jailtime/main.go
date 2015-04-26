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
	"flag"
	"fmt"
	"jailtime/copy"
	"jailtime/loader"
	"jailtime/spec"
	"os"
	"os/exec"
	"path"
	"sort"
)

const (
	VersionMajor = 0
	VersionMinor = 1
)

var (
	help = flag.Bool("help", false, "display this help and exit")
	link = flag.Bool("link", false, "hard link files instead of copying")
	//noClobber = flag.Bool("no-clobber", false, "do not overwrite existing "+
	//	"files")
	//reflink = flag.Bool("reflink", false, "perform lightweight copies using "+
	//	"CoW")
	//oneFilesystem = flag.Bool("one-filesystem", false, "do not cross "+
	//	"filesystem boundaries")
	verbose = flag.Bool("verbose", false, "explain what is being done")
	version = flag.Bool("version", false, "display version and exit")
)

func fatalln(v ...interface{}) {
	fmt.Fprint(os.Stderr, "jailtime: ")
	fmt.Fprintln(os.Stderr, v...)
	os.Exit(1)
}

// Prints more GNU-looking usage text.
func printUsage() {
	fmt.Printf("Usage: jailtime [OPTION]... FILE... TARGET\n" +
		"Create or update the chroot environment in TARGET using " +
		"specification\n" +
		"FILEs. TARGET needs to be a directory or not exist.\n\n")
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
		os.Exit(0)
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

func ExpandLexical(stmts spec.Statements) spec.Statements {
	todo := make(map[string]bool)
	// Expect at least half or the files to expand at least to its dir
	expanded := make(spec.Statements, 0, 3*len(stmts)/2)
	runs := spec.Statements{}
	for _, s := range stmts {
		var target string
		var dir string
		switch stmt := s.(type) {
		case spec.Directory:
			target = stmt.Target
			dir = target
			expanded = append(expanded, stmt)
		case spec.Run:
			// Do not deduplicate run statements
			runs = append(runs, stmt)
			continue
		default:
			target = spec.ChrootTarget(stmt)
		}
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
				expanded = append(expanded, spec.Directory{Target: dir})
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
			deps, err := loader.ImportedLibraries(stmt.Source)
			if err != nil {
				fatalln(err)
			}
			for _, d := range deps {
				expanded = append(expanded, spec.RegularFile{Source: d,
					Target: d})
			}
		}
	}
	return ExpandLexical(expanded)
}

func RunCommands(chrootDir string, runs []spec.Run) (err error) {
	for _, r := range runs {
		fmt.Println(r.Command)
		cmd := exec.Command("/bin/sh", "-c", r.Command)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err = cmd.Run(); err != nil {
			return
		}
	}
	return
}

func UpdateChroot(chrootDir string, stmts spec.Statements) error {
	for _, s := range ExpandWithDependencies(stmts) {
		fmt.Println(s)
		_ = s
		switch stmt := s.(type) {
		case spec.Directory:
			target := path.Join(chrootDir, stmt.Target)
			if err := os.MkdirAll(target, 0755); err != nil {
				return err
			}
		case spec.RegularFile:
			target := path.Join(chrootDir, stmt.Target)
			if _, err := copy.File(stmt.Source, target, nil); err != nil {
				return err
			}
		}
	}
	//fmt.Printf("Running commands inside %s:\n", chrootDir)
	return nil //RunCommands(chrootDir, runs)
}

func main() {
	//fmt.Println(copy.File("test2cp", "__test", &copy.Options{
	//	Progress: func(written, total int64) bool {
	//		fmt.Printf("%d/%d\n", written, total)
	//		return true
	//	}}))
	processCommandLine()

	stmts, err := spec.Parse("basic_shell")
	if err != nil {
		fatalln(err)
	}
	chrootDir := path.Clean("./chroot")
	if err = UpdateChroot(chrootDir, stmts); err != nil {
		fatalln(err)
	}
}
