/*
 * jailtime version 0.4
 * Copyright (c)2015-2017 Christian Blichmann
 *
 * Linux-specific Copy-on-Write functionality
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
	"syscall"
)

func HaveCoW() bool {
	return true
}

func cloneFile(src, dest string) error {
	srcFd, err := syscall.Open(src, syscall.O_RDONLY, 0644)
	if err != nil {
		return err
	}
	defer syscall.Close(srcFd) // Safe to ignore close error on read-only files
	destFd, err := syscall.Open(dest, syscall.O_WRONLY|syscall.O_CREAT, 0644)
	if err != nil {
		return err
	}
	defer syscall.Close(destFd)
	return ioctl(destFd,
		// BTRFS_IOC_CLONE, see linux/btrfs.h
		iow(btrfsIoCtlMagic, 9, 4 /* sizeof(int) */),
		uintptr(srcFd))
}
