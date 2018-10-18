/*
 * jailtime version 0.7
 * Copyright (c)2015-2018 Christian Blichmann
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

package copy // import "blichmann.eu/code/jailtime/copy"

import (
	"bufio"
	"io"
	"os"
)

const (
	ReflinkNo = iota
	ReflinkAlways
	ReflinkAuto
)

type Options struct {
	Force             bool
	Reflink           int
	RemoveDestination bool
	Progress          func(written, total int64) bool
	BufSize           int64
}

const defaultBufSize = 1 << 20 // 1 MiB

// File copies the file named in src to a file named in dest. It returns the
// number of bytes written and an error, if any. File uses buffered I/O and
// delegates to io.Copy to do the actual work. The behavior can be optionally
// influenced by setting options in opt.
func File(src, dest string, opt *Options) (written int64, err error) {
	if opt == nil {
		opt = &Options{}
	}
	if opt.Progress == nil {
		opt.Progress = func(w, t int64) bool { return true }
	}
	if opt.BufSize <= 0 {
		opt.BufSize = defaultBufSize
	}

	s, err := os.Open(src)
	if err != nil {
		return
	}
	defer s.Close() // Safe to ignore close error on read-only files

	var fi os.FileInfo
	if fi, err = s.Stat(); err != nil {
		return
	}
	total := fi.Size()

	if opt.RemoveDestination {
		if err = os.Remove(dest); err != nil {
			return
		}
	}

	hardFail := false
	reflink := HaveCoW() && opt.Reflink != ReflinkNo
	var t *os.File
Retry:
	if reflink {
		err = cloneFile(src, dest)
		if err != nil && opt.Reflink == ReflinkAuto {
			reflink = false
			goto Retry
		}
		opt.Progress(total, total)
		return
	} else {
		t, err = os.Create(dest)
	}
	if err != nil {
		if opt.Force && !hardFail {
			hardFail = true
			if err = os.Remove(dest); err == nil {
				goto Retry
			}
		}
		return
	}
	err = t.Chmod(fi.Mode())
	if err != nil {
		return
	}

	r, w := bufio.NewReader(s), bufio.NewWriter(t)
	var copied int64
	for written = 0; opt.Progress(written, total) && written < total; {
		copied, err = io.CopyN(w, r, opt.BufSize)
		written += copied
		if copied != opt.BufSize {
			break
		}
	}
	if err == io.EOF {
		err = nil
	}
	lastErr := err
	if err := w.Flush(); lastErr == nil {
		lastErr = err
	}
	if err := t.Close(); lastErr == nil {
		lastErr = err
	}

	err = lastErr
	return
}
