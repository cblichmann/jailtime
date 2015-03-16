#!/usr/bin/env make
#
# jailtime version 0.1
# Copyright (c)2015 Christian Blichmann
#
# Makefile for POSIX compatible systems

# Find location of this Makefile
this_dir := $(dir $(abspath $(lastword $(MAKEFILE_LIST))))

# Source Configuration {
go_packages = \
	jailtime/jailtime
# } Source Configuration

# Directories
bin_dir = $(this_dir)bin
go_binaries = $(addprefix $(bin_dir)/,$(go_packages))

third_party_dir := $(abspath $(this_dir)../third_party)

export GOPATH ?= $(third_party_dir)/go:$(this_dir)

.PHONY: all
all: $(go_binaries)

env:
#	# Use like this: $ eval $(make env)
	@echo export GOPATH=$(GOPATH)

clean:
	@echo "  [Clean]     Removing build artifacts"
	@for i in bin pkg; do rm -rf $(this_dir)$$i; done

$(go_binaries): %:
	@echo "  [Install]   $@"
	@go install $(@:$(bin_dir)/%=%)

