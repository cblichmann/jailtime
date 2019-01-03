/*
 * jailtime version 0.8
 * Copyright (c)2015-2019 Christian Blichmann
 *
 * Actions for jailspec statements
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

package action // import "blichmann.eu/code/jailtime/action"

import (
	"os"
	"os/exec"
	"syscall"

	"blichmann.eu/code/jailtime/copy"
	"blichmann.eu/code/jailtime/spec"
)

func Directory(target string, d spec.Directory) error {
	mode := d.FileAttr().Mode
	if mode == spec.FileModeUnspecified {
		mode = 0755
	}
	return os.MkdirAll(target, os.FileMode(mode))
}

func RegularFile(target string, f spec.RegularFile, copts *copy.Options) error {
	if _, err := copy.File(f.Source(), target, copts); err != nil {
		return err
	}
	if mode := f.FileAttr().Mode; mode != spec.FileModeUnspecified {
		return os.Chmod(target, os.FileMode(mode))
	}
	return nil
}

func Link(target string, l spec.Link) error {
	if _, err := os.Stat(target); err == nil { // Link exists
		if err = os.Remove(target); err != nil {
			return err
		}
	}
	linkName := l.Source()
	if l.HardLink() {
		return os.Link(linkName, target)
	}
	return os.Symlink(linkName, target)
}

func Device(target string, d spec.Device) error {
	if _, err := os.Stat(target); err == nil { // Device exists
		if err = os.Remove(target); err != nil {
			return err
		}
	}
	mode := d.FileAttr().Mode
	if mode == spec.FileModeUnspecified {
		mode = 0644
	}
	return syscall.Mknod(target, uint32(d.Type()|mode),
		MakeDev(d.Major(), d.Minor()))
}

func Run(target string, r spec.Run, chrootDir string) error {
	cmd := exec.Command("/bin/sh", "-c", r.Command())
	cmd.Dir = chrootDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
