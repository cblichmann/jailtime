jailtime [![Build Status](https://github.com/cblichmann/jailtime/workflows/build/badge.svg)](https://github.com/cblichmann/jailtime/actions?query=workflow%3Abuild)
========

jailtime is a command-line utility to create and manage chroot/jail
environments.
Why is this useful? jailtime helps to
  - create restricted SSH logins that only allow scp or git, etc.
  - build a Docker image without all the clutter of a fat base image based on
    a full Linux distribution.
  - restrict daemons into a filesystem sub-tree to enhance security.


Table of Contents
-----------------

  * [jailtime](README.md#jailtime-)
     * [Requirements](README.md#requirements)
     * [How to Build](README.md#how-to-build)
        * [Build using Make](README.md#build-using-make)
     * [How to Use](README.md#how-to-use)
        * [Writing Jail Specifications](README.md#writing-jail-specifications)
        * [Entering a chroot](README.md#entering-a-chroot)
     * [Bugs](README.md#bugs)
     * [Similar Tools](README.md#similar-tools)
     * [Copyright/License](README.md#copyrightlicense)


Requirements
------------

  - Go version 1.9 or later
  - Git version 1.7 or later
  - Optional: CDBS (to build the Debian packages)
  - Optional: GNU Make
  - Currently only runs on 32-bit or 64-bit x86 Linux and macOS


How to Build
------------

General way to build from source via `go get`:
```
go get blichmann.eu/code/jailtime
```

### Build using Make

To build from a specific revision/branch/tag, not using `go get`:
```bash
mkdir -p jailtime && cd jailtime
git clone https://github.com/cblichmann/jailtime.git .
# Optional: checkout a specific rev./branch/tag using i.e. git checkout
make
```

You may want to create a symlink to the binary somewhere in your path.


How to Use
----------

jailtime creates/updates a target chroot directory from an existing jail
specification (see next section). The general invocation syntax is:
```
jailtime <one or more jailspec files> <target dir>
```
Multiple jailspec files will be merged and their statements applied in order.

To get started with a rather basic chroot that allows to run Bash
interactively, see the files in the examples/ directory. For the basic shell
example:
```
jailtime examples/basic_shell.jailspec chroot_dir
```
This will copy (among other files) your local `/bin/bash` to
`chroot_dir/bin/bash` and copy its library dependencies as well. On a Debian
Jessie system, the resulting tree looks like this:
```
chroot_dir/
+- bin/
|  +- bash  cat  chgrp  chmod  chown  cp  cpio  date  dd  df  dir  ...
+- lib/x86_64-linux-gnu/
|  +- libacl.so.1      libattr.so.1     libc.so.6    libdl.so.2
|     libm.so.6        libncurses.so.5  libnsl.so.1  libpcre.so.3
|     libprocps.so.3   libpthread.so.0  librt.so.1   libselinux.so.1
|     libtinfo.so.5
+- lib64/
|  +- ld-linux-x86-64.so.2
+- usr/bin/
   +- arch  awk  base64  basename  cksum  csplit  cut  dircolors  ...
```

### Writing Jail Specifications

Jail specification files such as `examples/basic_shell.jailspec` follow a text
format with a few special directives. To start with a simple example:
```
# This is a single line comment. Blank lines and additional whitespace will be
# ignored.

# This copies the host file /bin/bash into the chroot. It also copies all
# library dependencies.
/bin/bash
```

When copying files, you can also specify the target:
```
# Copies /bin/bash to <chroot>/bin/sh.
/bin/bash /bin/sh
```
Instead of creating a copy, you can also create a (sym-)link:
```
# Copy bash to <chroot>/bin/bash and create a symlink from <chroot>/bin/sh to
# <chroot>/bin/bash
/bin/bash
/bin/sh -> /bin/bash

# Hardlinks are created with a fat arrow `=>':
/bin/bash_again => /bin/bash
```

To change file permissions inside the chroot, just append the file mode:
```
/home/myuser/ 600
/home/myuser/myfile 600
```

Some programs will likely need a few special device files in order to function.
They are created similar to normal files:
```
# Creates the two devices /dev/null and /dev/zero.
# For Linux device numbers see Documentation/admin-guide/devices.txt in the
# kernel source tree.
/dev/null c 1 3
/dev/zero c 1 5
```
Note: Device creation will most likely require jailtime to be run as root.

Use a 'run' directive for advanced customizations of the chroot:
```
# Add a nice saying, careful not to omit the leading "./"
run fortune > ./etc/motd
```
The run directive will execute the text following the `run` keyword in a shell
with the chroot directory set as its current directory.

Empty directories are created when the path name ends with a slash ('/'). There
is also a shorthand to create multiple directories, similar to Bash syntax:
```
# Creates /srv and /srv/nfs
/srv/nfs/
# Expands to /srv/nfs/alice/.ssh/ and /srv/nfs/bob/.ssh/ and creates these
# directories.
/srv/nfs/{alice,bob}/.ssh/
```

Jail specifications can also include other jail specifications:
```
include python27.jailspec
```
The include will be relative to the current specification file and file
inclusion may be nested up to 8 levels deep. Run statements are executed in
order and later specifications override earlier ones.


### Entering a chroot

On most systems, entering a chroot environment requires root or at least
administrative privileges. If `sudo` is installed, you can create and enter a
chroot with a basic shell like this:
```bash
jailtime examples/basic_shell.jailspec chroot_dir
sudo chroot chroot_dir
```
If you are on a system with [systemd](
http://freedesktop.org/wiki/Software/systemd/) (most Linux systems nowadays),
you can also easily create a lightweight container:
```bash
sudo systemd-nspawn -D chroot_dir/ /bin/bash
```
This uses the same underlying technique as [Docker](https://www.docker.com/),
Linux Containers (LXC), and allows for greater isolation.

Another good option is to use [nsjail](https://google.github.io/nsjail/),
which uses a similar technique but also allows to restrict the chroot even
further by using a seccomp-bpf based sandbox. Here is an example that changes
both the current user and group to 99999:
```bash
sudo nsjail -Mo --chroot chroot_dir/ --user 999999 --group 99999 -- /bin/bash
```

FreeBSD derived systems also have the [jail](
https://www.freebsd.org/cgi/man.cgi?query=jail&format=html) utility, which
serves a similar purpose.


Bugs
----

  - Error messages could be more specific


Similar Tools
-------------

These tools serve a similar purpose or are somewhat related:
  - [Jailkit](http://olivier.sessink.nl/jailkit/), this also supports
    checking chroots for security problems and launching daemons inside a
    chroot. In its current form, jailtime corresponds mostly to `jk_cp`, the
    utility to copy files and their dependencies to a chroot directory.
  - [copy_exec from initramfs-tools](
    http://anonscm.debian.org/cgit/kernel/initramfs-tools.git/tree/hook-functions),
    this also copies files and their library dependencies.
  - [schroot](http://anonscm.debian.org/cgit/buildd-tools/schroot.git), used
    to execute commands or interactive shells in different chroot
    environments. It also supports BTRFS and LVM snapshots as well as
    on-the-fly chroots unpacked from tar files.
  - [debootstrap](http://anonscm.debian.org/cgit/d-i/debootstrap.git), this can
    install Debian-based distributions into a filesystem directory which then
    can be used as a chroot.


Copyright/License
-----------------

jailtime version 0.8
Copyright (c)2015-2020 Christian Blichmann <jailtime@blichmann.eu>

jailtime is licensed under a two-clause BSD license, see the LICENSE file
for details.
