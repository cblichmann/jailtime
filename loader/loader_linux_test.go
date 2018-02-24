/*
 * jailtime version 0.6
 * Copyright (c)2015-2018 Christian Blichmann
 *
 * Tests for the Linux import library utility
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
	"os"
	"reflect"
	"sort"
	"testing"
)

func TestImportedLibaries(t *testing.T) {
	wd, _ := os.Getwd()
	defer os.Chdir(wd)
	if err := os.Chdir("testdata"); err != nil {
		t.Error(err)
	}

	paths, err := ImportedLibraries("netcat.elf")
	if err != nil {
		t.Error(err)
	}
	sort.Strings(paths)
	expected := []string{
		// Keep sorted
		"/lib/x86_64-linux-gnu/libbsd.so.0",
		"/lib/x86_64-linux-gnu/libc.so.6",
		"/lib/x86_64-linux-gnu/libpthread.so.0",
		"/lib/x86_64-linux-gnu/libresolv.so.2",
		"/lib/x86_64-linux-gnu/librt.so.1",
		"/lib64/ld-linux-x86-64.so.2",
	}
	if !reflect.DeepEqual(paths, expected) {
		t.Errorf("expected %s, actual %s", expected, paths)
	}
}
