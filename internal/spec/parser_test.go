/*
 * jailtime version 0.8
 * Copyright (c)2015-2020 Christian Blichmann
 *
 * Specification file parser tests
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
	"testing"
)

const (
	testFile = "no_such.spec"
	testLine = 31
)

func checkParseSpecLineEmpty(line string, t *testing.T) {
	t.Helper()
	if stmts, err := parseSpecLine(testFile, testLine, line, nil); err != nil {
		t.Errorf("expected no error, actual: %s", err)
	} else if stmts != nil {
		t.Errorf("expected empty stmts, actual: %s", stmts)
	}
}

func checkParseSpecLineSingleStmt(line string, t *testing.T) Statement {
	stmts, err := parseSpecLine(testFile, testLine, line, nil)
	if err != nil {
		t.Errorf("expected no error, actual: %s", err)
	}
	if n := len(stmts); n != 1 {
		t.Fatalf("expected single statement, actual: %d", n)
	}
	return stmts[0]
}

func TestParseSpecLineEmpty(t *testing.T) {
	checkParseSpecLineEmpty("", t)
	checkParseSpecLineEmpty("  # Comment, but empty", t)
}

func TestParseSpecLineRegularFile(t *testing.T) {
	const (
		expectSource       = "/some/file"
		expectTarget       = "/some/target"
		expectTargetNumber = "/600"
		expectModeUnset    = -1
		expectMode         = 0600
	)

	// /some/file
	stmt := checkParseSpecLineSingleStmt(expectSource, t)
	if f, ok := stmt.(RegularFile); !ok {
		t.Error("expected type RegularFile")
	} else if source := f.Source(); source != expectSource {
		t.Errorf("expected %s, actual: %s", expectSource, source)
	} else if target := f.Target(); target != source {
		t.Errorf("expected %s, actual: %s", source, target)
	} else if mode := f.FileAttr().Mode; mode != expectModeUnset {
		t.Errorf("expected %o, actual: %o", expectModeUnset, mode)
	}

	// /some/file 600
	stmt = checkParseSpecLineSingleStmt(fmt.Sprintf("%s %o", expectSource,
		expectMode), t)
	if f, ok := stmt.(RegularFile); !ok {
		t.Error("expected type RegularFile")
	} else if source := f.Source(); source != expectSource {
		t.Errorf("expected %s, actual: %s", expectSource, source)
	} else if target := f.Target(); target != source {
		t.Errorf("expected %s, actual: %s", source, target)
	} else if mode := f.FileAttr().Mode; mode != expectMode {
		t.Errorf("expected %o, actual: %o", expectMode, mode)
	}

	// /some/file /some/target
	stmt = checkParseSpecLineSingleStmt(fmt.Sprintf("%s %s", expectSource,
		expectTarget), t)
	if f, ok := stmt.(RegularFile); !ok {
		t.Error("expected type RegularFile")
	} else if source := f.Source(); source != expectSource {
		t.Errorf("expected %s, actual: %s", expectSource, source)
	} else if target := f.Target(); target != expectTarget {
		t.Errorf("expected %s, actual: %s", expectTarget, target)
	} else if mode := f.FileAttr().Mode; mode != expectModeUnset {
		t.Errorf("expected %o, actual: %o", expectModeUnset, mode)
	}

	// /some/file /600
	stmt = checkParseSpecLineSingleStmt(fmt.Sprintf("%s %s", expectSource,
		expectTargetNumber), t)
	if f, ok := stmt.(RegularFile); !ok {
		t.Error("expected type RegularFile")
	} else if source := f.Source(); source != expectSource {
		t.Errorf("expected %s, actual: %s", expectSource, source)
	} else if target := f.Target(); target != expectTargetNumber {
		t.Errorf("expected %s, actual: %s", expectTargetNumber, target)
	} else if mode := f.FileAttr().Mode; mode != expectModeUnset {
		t.Errorf("expected %o, actual: %o", expectModeUnset, mode)
	}

	// /some/file /some/target 600
	stmt = checkParseSpecLineSingleStmt(fmt.Sprintf("%s %s %o", expectSource,
		expectTarget, expectMode), t)
	if f, ok := stmt.(RegularFile); !ok {
		t.Error("expected type RegularFile")
	} else if source := f.Source(); source != expectSource {
		t.Errorf("expected %s, actual: %s", expectSource, source)
	} else if target := f.Target(); target != expectTarget {
		t.Errorf("expected %s, actual: %s", expectTarget, target)
	} else if mode := f.FileAttr().Mode; mode != expectMode {
		t.Errorf("expected %o, actual: %o", expectMode, mode)
	}
}

func TestParseSpecLineDirectory(t *testing.T) {
	const (
		expectSource       = "/sub/dir/"
		expectTarget       = "/sub/dir"
		expectTargetNumber = "/755"
		expectModeUnset    = -1
		expectMode         = 0755
	)

	// /some/file
	stmt := checkParseSpecLineSingleStmt(expectSource, t)
	if d, ok := stmt.(Directory); !ok {
		t.Error("expected type Directory")
	} else if source := d.Source(); len(source) > 0 {
		t.Errorf("expected \"\", actual: %s", source)
	} else if target := d.Target(); target != expectTarget {
		t.Errorf("expected %s, actual: %s", expectTarget, target)
	} else if mode := d.FileAttr().Mode; mode != expectMode {
		t.Errorf("expected %o, actual: %o", expectMode, mode)
	}
}

func TestParseSpecLineDirective(t *testing.T) {
	const expectCmd = "/bin/true"
	stmt := checkParseSpecLineSingleStmt("run "+expectCmd, t)
	if cmd, ok := stmt.(Run); !ok {
		t.Error("expected type Run")
	} else if cmd.Command() != expectCmd {
		t.Errorf("expected %s, actual: %s", expectCmd, cmd.Command())
	}

	const expectInclude = "no_such.jailspec"
	var includeFile string
	_, err := parseSpecLine(testFile, testLine, "include "+expectInclude,
		func(filename string) (Statements, error) {
			includeFile = filename
			return nil, nil
		})
	if err != nil {
		t.Errorf("expected no error, actual: %s", err)
	} else if includeFile != expectInclude {
		t.Errorf("expected %s, actual: %s", expectInclude, includeFile)
	}
}
