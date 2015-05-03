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

// General file attributes: if these are not specified in the spec, these will
// default to root:root with mode 0755 for type Directory and root:root with
// mode 0644 for RegularFile, Device and Link.
type FileAttr struct {
	Uid  int // User id
	Gid  int // Group id
	Mode int // File mode
}

type Statement interface {
	// Source outside the chroot
	Source() string

	// Target inside the chroot
	Target() string

	FileAttr() *FileAttr
}

type RegularFile struct {
	source   string
	target   string
	fileAttr FileAttr
}

func NewRegularFile(source, target string) RegularFile {
	return RegularFile{source: source, target: target}
}

func (r RegularFile) Source() string {
	return r.source
}

func (r RegularFile) Target() string {
	return r.target
}

func (r RegularFile) FileAttr() *FileAttr {
	return &r.fileAttr
}

type Device struct {
	target   string
	fileAttr FileAttr
	Type     int
	Major    int
	Minor    int
}

func (d Device) Source() string {
	return ""
}

func (d Device) Target() string {
	return d.target
}

func (d Device) FileAttr() *FileAttr {
	return &d.fileAttr
}

type Directory struct {
	target   string
	fileAttr FileAttr
}

func NewDirectory(target string) Directory {
	return Directory{target: target}
}

func (d Directory) Source() string {
	return ""
}

func (d Directory) Target() string {
	return d.target
}

func (d Directory) FileAttr() *FileAttr {
	return &d.fileAttr
}

type Link struct {
	source   string
	target   string
	fileAttr FileAttr
	HardLink bool
}

func (l Link) Source() string {
	return l.source
}

func (l Link) Target() string {
	return l.target
}

func (l Link) FileAttr() *FileAttr {
	return &l.fileAttr
}

type Run struct {
	// Command to be run outside the chroot with the current working directory
	// set to the chroot.
	Command string
}

func (r Run) Source() string {
	return ""
}

func (r Run) Target() string {
	return ""
}

func (r Run) FileAttr() *FileAttr {
	return nil
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
	}
	if ii > ji {
		return false
	}
	return s[i].Target() < s[j].Target()
}
