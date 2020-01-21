/*
 * jailtime version 0.8
 * Copyright (c)2015-2020 Christian Blichmann
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

import "fmt"

// FileAttr General file attributes: if these are not specified in the spec,
// the file permissions of the source will be used for regular files. For
// directories, the default mode is 755.
// In all cases, user and group id default to the values of the current user.
type FileAttr struct {
	UID  int // User id
	GID  int // Group id
	Mode int // File mode
}

const FileModeUnspecified = -1

// Statement represents a single filesystem entity or command to be executed
// inside the chroot.
type Statement interface {
	// Source outside the chroot, may be empty
	Source() string

	// Target inside the chroot, may be empty
	Target() string

	// Filesystem attributes, may return nil
	FileAttr() *FileAttr

	// Verbose returns a verbose description of the statement suitable for
	// display.
	Verbose() string
}

type targetChrootObj struct {
	target   string
	fileAttr FileAttr
}

func (t targetChrootObj) Target() string {
	return t.target
}

func (t targetChrootObj) FileAttr() *FileAttr {
	return &t.fileAttr
}

type RegularFile struct {
	source string
	targetChrootObj
}

func NewRegularFile(source, target string) RegularFile {
	return RegularFile{source, targetChrootObj{target: target,
		fileAttr: FileAttr{Mode: FileModeUnspecified}}}
}

func (r RegularFile) Source() string {
	return r.source
}

func (r RegularFile) Verbose() string {
	return fmt.Sprintf("copy file: %s > %s", r.source, r.target)
}

type Device struct {
	targetChrootObj
	type_ int
	major int
	minor int
}

func NewDevice(target string, type_, major, minor int) Device {
	return Device{targetChrootObj{target: target,
		fileAttr: FileAttr{Mode: FileModeUnspecified}}, type_, major, minor}
}

func (d Device) Source() string {
	return ""
}

func (d Device) Verbose() string {
	return fmt.Sprintf("create device: %s mode 0%o", d.target,
		d.fileAttr.Mode)
}

func (d Device) Type() int {
	return d.type_
}

func (d Device) Major() int {
	return d.major
}

func (d Device) Minor() int {
	return d.minor
}

type Directory struct {
	targetChrootObj
}

func NewDirectory(target string) Directory {
	return Directory{targetChrootObj{target: target,
		fileAttr: FileAttr{Mode: FileModeUnspecified}}}
}

func (d Directory) Source() string {
	return ""
}

func (d Directory) Verbose() string {
	return fmt.Sprintf("create dir: %s mode 0%o", d.target, d.fileAttr.Mode)
}

type Link struct {
	source string
	targetChrootObj
	hardLink bool
}

func NewLink(source, target string, hardLink bool) Link {
	return Link{source, targetChrootObj{target: target}, hardLink}
}

func (l Link) Source() string {
	return l.source
}

func (l Link) Verbose() string {
	var (
		action, arrow string
	)
	if l.HardLink() {
		action = "create hardlink"
		arrow = "=>"
	} else {
		action = "create symlink"
		arrow = "->"
	}
	return fmt.Sprintf("%s: %s %s %s", action, l.target, arrow, l.source)
}

func (l Link) HardLink() bool {
	return l.hardLink
}

type Run struct {
	// Command to be run outside the chroot with the current working directory
	// set to the chroot.
	command string
}

func NewRun(command string) Run {
	return Run{command}
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

func (r Run) Verbose() string {
	return fmt.Sprintf("run command: %s", r.Command())
}

func (r Run) Command() string {
	return r.command
}

// Statements is a sortable slice of Statement elements.
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
		return 900
	default:
		return 1000
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
	if _, ok := s[i].(Run); ok {
		return false // Always sort run statements last
	}
	if ii < ji {
		return true
	}
	if ii > ji {
		return false
	}
	return s[i].Target() < s[j].Target()
}
