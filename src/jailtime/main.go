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
	"path"
	"regexp"
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

var (
	loaderRe = regexp.MustCompile("^\\s*" + LoaderExecutable +
		"\\s+\\(0x[[:xdigit:]]+\\)\\s*$")
	depRe = regexp.MustCompile("^.*\\s=>\\s+(.*)\\s+\\(0x[[:xdigit:]]+\\)\\s*$")
)

func importedLibraries(binary string) (deps []string, err error) {
	// Do not wait for the loader to return an error on non-existing files.
	if _, err = os.Stat(binary); os.IsNotExist(err) {
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
		if m := depRe.FindStringSubmatch(line); m != nil {
			if len(m[1]) > 0 {
				deps = append(deps, m[1])
			}
		} else if loaderRe.FindStringSubmatch(line) == nil {
			fatalln("bug: OS loader returned unexpected formt")
		}
	}
	err = cmd.Wait()
	return
}

func main() {
	processCommandLine()

	spec, err := OpenSpec("basic_shell")
	if err != nil {
		fatalln(err)
	}
	chrootDir := path.Clean("./chroot")

	dirsToCreate := make(map[string]bool)
	filesToCopy := make(map[string]bool)
	for _, s := range spec {
		fmt.Println(s)
		switch s.Type {
		case RegularFile:
			sourcePath := s.Path[0]
			deps, err := importedLibraries(sourcePath)
			if err != nil {
				fatalln(err)
			}
			targetPath := path.Join(chrootDir, path.Dir(s.Target))
			dirsToCreate[targetPath] = true
			//fmt.Printf("mkdir -p %s\n", targetPath)
			//os.MkdirAll(targetPath, 0755)
			//fmt.Printf("cp %s %s/\n", sourcePath, targetPath)
			filesToCopy[sourcePath] = true
			for _, d := range deps {
				fmt.Println(d)
				depPath := path.Join(chrootDir, path.Dir(d))
				dirsToCreate[depPath] = true
				//fmt.Printf("mkdir -p %s\n", depPath)
				//fmt.Printf("cp %s %s/\n", d, depPath)
				filesToCopy[d] = true
			}
		}
	}
	fmt.Println("-------")
	for k, _ := range dirsToCreate {
		fmt.Println(k)
	}
	fmt.Println("-------")
	for k, _ := range filesToCopy {
		fmt.Println(k)
	}
}
