/*
 * jailtime version 0.6
 * Copyright (c)2015-2018 Christian Blichmann
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

package spec // import "blichmann.eu/code/jailtime/spec"

import (
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

func TestParseSpecLineEmpty(t *testing.T) {
	checkParseSpecLineEmpty("", t)
	checkParseSpecLineEmpty("  # Comment, but empty", t)
}

func TestParseSpecLineDirective(t *testing.T) {
	const expectCmd = "/bin/true"
	stmts, err := parseSpecLine(testFile, testLine, "run "+expectCmd, nil)
	if err != nil {
		t.Errorf("expected no error, actual: %s", err)
	}
	if n := len(stmts); n != 1 {
		t.Errorf("expected single statement, actual: %d", n)
	}
	if cmd, ok := stmts[0].(Run); !ok {
		t.Error("expected type Run")
	} else if cmd.Command() != expectCmd {
		t.Errorf("expected %s, actual: %s", expectCmd, cmd.Command())
	}

	const expectInclude = "no_such.jailspec"
	var includeFile string
	stmts, err = parseSpecLine(testFile, testLine, "include "+expectInclude,
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
