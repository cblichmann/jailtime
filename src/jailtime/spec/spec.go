/*
 * jailtime version 0.1
 * Copyright (c)2015 Christian Blichmann
 *
 * Chroot specification
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
 * ARE DISCLAIMED. IN NO EVENT SHALL CHRISTIAN BLICHMANN BE LIABLE FOR ANY
 * DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES
 * (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES;
 * LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND
 * ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
 * (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF
 * THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
 */

package spec

type Statement interface{}

// General file attributes: if these are not specified in the spec, these will
// default to root:root with mode 0755 for type Directory and root:root with
// mode 0644 for RegularFile, Device and Link.
type FileAttr struct {
	Uid  int // User id
	Gid  int // Group id
	Mode int // File mode
}

type RegularFile struct {
	Source string // Source file outside the chroot
	Target string // Target inside the chroot
	*FileAttr
}

type Device struct {
	Target string // Target inside the chroot
	*FileAttr
	Type  int
	Major int
	Minor int
}

type Directory struct {
	Target string // Target inside the chroot
	*FileAttr
}

type Link struct {
	LinkSource string
	HardLink   bool
	Target     string // Target inside the chroot
	*FileAttr
}

type Run struct {
	// Command to be run outside the chroot with the current working directory
	// set to the chroot.
	Command string
}

type Statements []Statement

func statementToInt(s Statement) int {
	switch s.(type) {
	case Directory:
		return 10
	case RegularFile:
		return 20
	case Device:
		return 30
	case Link:
		return 40
	case Run:
		return 50
	default:
		return 100
	}
}

func ChrootTarget(s Statement) string {
	switch stmt := s.(type) {
	case Directory:
		return stmt.Target
	case RegularFile:
		return stmt.Target
	case Device:
		return stmt.Target
	case Link:
		return stmt.Target
	case Run:
		return ""
	default:
		panic("unsupported Statement")
	}
}

func (s Statements) Len() int {
	return len(s)
}

func (s Statements) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s Statements) Less(i, j int) bool {
	ii, ji := statementToInt(s[i]), statementToInt(s[j])
	if ii < ji {
		return true
	} else if ii > ji {
		return false
	}
	return ChrootTarget(s[i]) < ChrootTarget(s[j])
}
