/*
 * jailtime version 0.8
 * Copyright (c)2015-2022 Christian Blichmann
 *
 * File copy utility
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

package copy

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

func TestFile(t *testing.T) {
	const content string = "128 byte test content =========|" +
		"=======|=======|=======|=======|" +
		"=======|=======|=======|=======|" +
		"=======|=======|=======|=======|"

	td, err := ioutil.TempDir("", "copy_test")
	if err != nil {
		t.Fatal(err)
	}
	t.Log(td)
	defer os.RemoveAll(td)

	tf := filepath.Join(td, "testfile")
	if err := ioutil.WriteFile(tf, []byte(content), 0666); err != nil {
		t.Fatal(err)
	}

	var written int64

	// Simple copy with default settings
	cf := filepath.Join(td, "copiedfile")
	written, err = File(tf, cf, nil)
	if err != nil {
		t.Fatal(err)
	}
	if written != int64(len(content)) {
		t.Errorf("expected %d, actual %d", len(content), written)
	}

	// Copy with progress callback and small buffer, overwrite
	numCalled := 0
	written, err = File(tf, cf, &Options{
		Progress: func(written, total int64) bool {
			numCalled++
			return true
		},
		BufSize: int64(len(content) / 2),
	})
	if err != nil {
		t.Fatal(err)
	}
	if written != int64(len(content)) {
		t.Errorf("expected %d, actual %d", len(content), written)
	}
	if numCalled < 2 {
		t.Errorf("expected at least %d, actual %d", 2, numCalled)
	}
}
