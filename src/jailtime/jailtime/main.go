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
	"jailtime/loader"
	"jailtime/spec"
	"os"
	"path"
)

const (
	VersionMajor = 0
	VersionMinor = 1
)

var (
	help    = flag.Bool("help", false, "display this help and exit")
	version = flag.Bool("version", false, "display version and exit")
	link    = flag.Bool("link", false, "hard link files instead of copying")
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

//func expand(j *spec.JailSpec) ([]*spec.Statement, error) {
//	for _, s := range j {
//	}
//	return
//}

func apply(j *spec.JailSpec, targetChroot string) error {
	return nil
}

func main() {
	processCommandLine()

	sp, err := spec.Parse("basic_shell")
	if err != nil {
		fatalln(err)
	}
	//chrootDir := path.Clean("./chroot")

	todo := make(map[string]*spec.Statement) // Target => Source
	toRun := make([]string, 0, 10)
	for _, s := range sp {
		switch s.Type {
		case spec.RegularFile:
			deps, err := loader.ImportedLibraries(s.Source)
			if err != nil {
				fatalln(err)
			}
			targetDir := path.Dir(s.Target)
			todo[targetDir] = &spec.Statement{
				Type:   spec.Directory,
				Target: s.Target,
				FileAttr: spec.FileAttr{
					Uid:  s.FileAttr.Uid,
					Gid:  s.FileAttr.Gid,
					Mode: s.FileAttr.Mode | 0111 /* ugo+x */}}
			todo[s.Target] = &s

			for _, d := range deps {
				targetDir = path.Dir(d)
				// TODO(cblichmann): Copy mode from dependencies' dir and file.
				todo[targetDir] = &spec.Statement{
					Type:   spec.Directory,
					Target: targetDir,
					FileAttr: spec.FileAttr{
						Uid:  0,
						Gid:  0,
						Mode: 0755}}
				todo[d] = &spec.Statement{
					Type:   spec.RegularFile,
					Source: d,
					Target: d,
					FileAttr: spec.FileAttr{
						Uid:  0,
						Gid:  0,
						Mode: 0644}}
			}
		case spec.Device:
			fallthrough
		case spec.Directory:
			fallthrough
		case spec.SymLink:
			fallthrough
		case spec.HardLink:
			todo[s.Source] = &s

		case spec.Run:
			toRun = append(toRun, s.Target)
		}
	}

	for k, _ := range todo {
		fmt.Println(k)
	}
}
