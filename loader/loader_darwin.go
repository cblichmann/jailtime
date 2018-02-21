/*
 * jailtime version 0.6
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
	"debug/macho"
	"encoding/binary"
	"fmt"
	"os"
	"runtime"
	"strings"
)

const LoaderExecutable = "/usr/lib/dyld"

const (
	// Extra constants since macOS 10.1
	LoadCmdReqDyld         macho.LoadCmd = 0x80000000
	LoadCmdReExportDylib   macho.LoadCmd = 0x1f | LoadCmdReqDyld
	LoadCmdLoadUpwardDylib macho.LoadCmd = 0x23 | LoadCmdReqDyld
)

var machoCpu macho.Cpu = 0

func init() {
	// Map the current GOARCH to a supported Mach-O CPU. We only support CPUs
	// that are supported by macOS/iOS as well.
	if c, ok := map[string]macho.Cpu{
		"386":   macho.Cpu386,
		"amd64": macho.CpuAmd64,
		"arm":   macho.CpuArm,
		// TODO(cblichmann): Enable once AArch64 is officially supported.
		//"arm64": macho.CpuArm64,
	}[runtime.GOARCH]; ok {
		machoCpu = c
	}
}

func openMachO(binary string) (*macho.File, error) {
	if machoCpu == 0 {
		return nil, fmt.Errorf("no Mach-O arch matching GOARCH: %s",
			runtime.GOARCH)
	}
	// Try to open as non-fat binary first
	if f, err := macho.Open(binary); err == nil {
		if f.Cpu != machoCpu {
			return nil, fmt.Errorf("unexpected Mach-O arch: %s", machoCpu)
		}
		return f, nil
	}
	fat, err := macho.OpenFat(binary)
	if err != nil {
		return nil, err
	}
	for _, a := range fat.Arches {
		if a.FatArchHeader.Cpu == machoCpu {
			return a.File, nil
		}
	}
	return nil, fmt.Errorf("Mach-O arch not in file: %s", machoCpu)
}

func ImportedLibraries(filename string) (deps []string, err error) {
	f, err := os.Open(filename)
	if err != nil {
		return
	}
	defer f.Close()

	m := make([]byte, 4 /* uint32 */)
	f.Read(m) // Ignore errors
	be := binary.BigEndian.Uint32(m[:])
	le := binary.LittleEndian.Uint32(m[:])
	if be != macho.Magic32 && be != macho.Magic64 &&
		le != macho.Magic32 && le != macho.Magic64 {
		// File is either too small or not a Mach-O
		return
	}

	resolved := map[string]bool{
		// On macOS, all binaries need "dyld"
		LoaderExecutable: true,
		filename:         false}
	for {
		todo := []string{}
		for l, r := range resolved {
			if r {
				continue
			}
			var f *macho.File
			if f, err = openMachO(l); err != nil {
				return
			}
			defer f.Close()
			var newLibs []string
			bo := f.ByteOrder
			for _, cmd := range f.Loads {
				raw := cmd.Raw()
				// TODO(cblichmann): Optionally handle weak dylibs.
				c := macho.LoadCmd(bo.Uint32(raw[0:4]))
				if c == LoadCmdReExportDylib || c == LoadCmdLoadUpwardDylib {
					n := bo.Uint32(raw[8:12])
					if n < uint32(len(raw)) {
						path := strings.TrimRight(string(raw[n:]), "\x00")
						todo = append(todo, path)
					}
				}
			}
			if newLibs, err = f.ImportedLibraries(); err != nil {
				return
			}
			todo = append(todo, newLibs...)
			resolved[l] = true
		}
		if len(todo) == 0 {
			break
		}
		for _, l := range todo {
			if _, ok := resolved[l]; !ok {
				resolved[l] = false
			}
		}
	}
	deps = []string{}
	for l, _ := range resolved {
		deps = append(deps, l)
	}
	return
}
