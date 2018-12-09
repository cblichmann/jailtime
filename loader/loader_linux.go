/*
 * jailtime version 0.8
 * Copyright (c)2015-2018 Christian Blichmann
 *
 * Import library utility
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

package loader // import "blichmann.eu/code/jailtime/loader"

import (
	"debug/elf"
	"os"
	"path/filepath"
	"strings"
)

// readELFInterpreter returns the value of the interpreter (dynamic loader)
// listed in the ELF program header of f. If there is no interpreter set,
// returns the empty string.
func readELFInterpreter(f *elf.File) string {
	const pathMax = 4096
	for _, p := range f.Progs {
		if p.Type == elf.PT_INTERP {
			r := p.Open()
			m := p.Filesz
			if m > pathMax {
				m = pathMax
			}
			b := make([]byte, m)
			r.Read(b)
			return strings.TrimRight(string(b), "\x00")
		}
	}
	return ""
}

func ImportedLibraries(filename string) (deps []string, err error) {
	// Note: The code below will likely work for the BSDs/Solaris as well, but
	//       is untested on those patforms.
	f, err := os.Open(filename)
	if err != nil {
		return
	}
	defer f.Close()

	b := make([]byte, len(elf.ELFMAG))
	if _, err2 := f.Read(b); err2 != nil || string(b) != elf.ELFMAG {
		// File is either too small or not an ELF
		return
	}

	e, err := elf.NewFile(f)
	if err != nil {
		return
	}
	defer e.Close()

	libs, err := e.ImportedLibraries()
	if err != nil {
		return
	}

	interp := readELFInterpreter(e)
	interpBase := filepath.Base(interp)
	resolved := map[string]string{interpBase: interp}
	paths := append([]string{filepath.Dir(interp)}, LdSearchPaths...)
	for {
		numResolved := len(resolved)
		for _, l := range libs {
			if _, ok := resolved[l]; ok {
				continue
			}
			r := FindLibraryFunc(l, paths, func(path string) bool {
				g, err := elf.Open(path)
				if err != nil {
					return false
				}
				defer g.Close()
				if g.Class == e.Class && g.Machine == e.Machine {
					newLibs, err := g.ImportedLibraries()
					libs = append(libs, newLibs...)
					return err == nil
				}
				return false
			})
			if r != "" {
				resolved[l] = r
			}
		}
		if numResolved == len(resolved) {
			break
		}
	}
	for _, v := range resolved {
		deps = append(deps, v)
	}
	return
}
