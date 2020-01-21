/*
 * jailtime version 0.8
 * Copyright (c)2015-2020 Christian Blichmann
 *
 * Statement expansion tests
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

package spec

import (
	"fmt"
	"reflect"
	"testing"
)

func TestLexicalExpand(t *testing.T) {
	expanded := ExpandLexical(Statements{
		NewRegularFile("/d_source", "/d_target"),
		NewRegularFile("/a_source", "/a_target"),
		NewRegularFile("/c_source", "/c_target"),
		NewRegularFile("/b_source", "/z_target"),  // Duplicate target
		NewRegularFile("/d_source", "/d_target2"), // Duplicate source
		NewDirectory("/target/directory/innermost/node"),
		NewRun("echo 'hello' > ./test"),
		NewRun("gzip ./test"),
		NewRun("gunzip ./test.gz"),
		NewRun("cat ./test"),
		NewRegularFile("/z_source", "/z_target"), // Duplicate target
		NewRegularFile("/e_source", "/e_target"),
	})
	expected := Statements{
		NewDirectory("/target"),
		NewDirectory("/target/directory"),
		NewDirectory("/target/directory/innermost"),
		NewDirectory("/target/directory/innermost/node"),
		NewRegularFile("/a_source", "/a_target"),
		NewRegularFile("/c_source", "/c_target"),
		NewRegularFile("/d_source", "/d_target"),
		NewRegularFile("/d_source", "/d_target2"),
		NewRegularFile("/e_source", "/e_target"),
		NewRegularFile("/b_source", "/z_target"),
		NewRun("echo 'hello' > ./test"),
		NewRun("gzip ./test"),
		NewRun("gunzip ./test.gz"),
		NewRun("cat ./test"),
	}
	if !reflect.DeepEqual(expanded, expected) {
		t.Errorf("expected %s, actual %s", expected, expanded)
	}
}

func TestLexicalExpandModes(t *testing.T) {
	dir := NewDirectory("/target/directory/innermost/node")
	dir.fileAttr.Mode = 0755 // Expect same mode on expanded directories
	expanded := ExpandLexical(Statements{dir})
	fmt.Println()
	var expectMode int
	for _, d := range expanded {
		expectMode = dir.FileAttr().Mode
		if mode := d.FileAttr().Mode; mode != expectMode {
			t.Errorf("expected %o, actual %o", expectMode, mode)
		}
	}
}
