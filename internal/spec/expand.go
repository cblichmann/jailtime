/*
 * jailtime version 0.8
 * Copyright (c)2015-2020 Christian Blichmann
 *
 * Statement expansion/deduplication
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
	"path/filepath"
	"sort"
)

// ExpandLexical deduplicates and sorts a list of statements while expanding
// directory paths. Run statements are never deduplicated are kept in order of
// appearace in the list.
func ExpandLexical(stmts Statements) Statements {
	done := make(map[string]bool)
	// Expect at least half of the files to expand at least to their dir
	expanded := make(Statements, 0, 3*len(stmts)/2)
	for _, s := range stmts {
		var dir string
		switch stmt := s.(type) {
		case Directory:
			dir = stmt.Target()
			expanded = append(expanded, stmt)
		case Run:
			// Do not deduplicate run statements
			expanded = append(expanded, stmt)
			continue
		}
		target := s.Target()
		if _, ok := done[target]; ok {
			continue
		}
		done[target] = true
		if dir != target {
			expanded = append(expanded, s)
			dir = filepath.Dir(target)
		}
		for dirLen := 0; dirLen != len(dir) && dir != "/"; {
			if _, ok := done[dir]; !ok {
				d := NewDirectory(dir)
				d.fileAttr = *s.FileAttr()
				if _, ok := s.(RegularFile); ok {
					if s.FileAttr().Mode == -1 {
						d.fileAttr.Mode = 0755
					}
				}
				expanded = append(expanded, d)
				done[dir] = true
			}
			dirLen = len(dir)
			dir = filepath.Dir(dir)
		}
	}
	sort.Stable(expanded)
	return expanded
}
