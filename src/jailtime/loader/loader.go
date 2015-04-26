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

package loader

import (
	"bufio"
	"debug/elf"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

var (
	depRe = regexp.MustCompile("^.*\\s=>\\s+(.*)\\s+\\(0x[[:xdigit:]]+\\)\\s*$")
	dsoRe = regexp.MustCompile("^.*(?:\\s+=>)?\\s+\\(0x[[:xdigit:]]+\\)\\s*$")
)

func ImportedLibraries(binary string) (deps []string, err error) {
	// Do not wait for the loader to return an error on non-existing files. We
	// need to be able to read the file.
	f, err := os.Open(binary)
	if err != nil {
		return
	}
	defer f.Close()

	b := make([]byte, len(elf.ELFMAG))
	if _, err2 := f.Read(b); err2 != nil || string(b) != elf.ELFMAG {
		// File is either too small or not an ELF
		// TODO(cblichmann): Implement OS X support
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
		} else if dsoRe.FindStringSubmatch(line) != nil {
			deps = append(deps, LoaderExecutable)
		} else {
			err = fmt.Errorf("bug: OS loader returned unexpected format: ",
				strings.TrimSpace(line))
		}
	}
	err = cmd.Wait()
	return
}
