// Copyright 2023 Buf Technologies, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Matching the unix-like build tags in the Golang source i.e. https://github.com/golang/go/blob/912f0750472dd4f674b69ca1616bfaf377af1805/src/os/file_unix.go#L6
//
// We expanded this to all unix-like platforms, including those we don't support, as most
// of this should work without issue, and there are bigger problems with supporting i.e. js,wasm
// that are outside the scope of these build tags. Being able to build buf on i.e. openbsd
// was a blocker, see https://github.com/bufbuild/buf/issues/362 and the linked discussions.
// We still only officially support linux and darwin for buf as a whole.

//go:build aix || darwin || dragonfly || freebsd || (js && wasm) || linux || netbsd || openbsd || solaris

package main

import (
	"errors"
	"os"
	"path/filepath"
	"syscall"
)

const executableSuffix = ""

// extraInterruptSignals are signals beyond os.Interrupt that we want to be handled
// as interrupts.
//
// For unix-like platforms, this adds syscall.SIGTERM, although this is only
// tested on darwin and linux, which buf officially supports. Other unix-like
// platforms should have this as well, however.
var extraInterruptSignals = []os.Signal{
	syscall.SIGTERM,
}

// getDefaultCacheDirPath returns the cache directory path.
//
// This will be $XDG_CACHE_HOME for darwin and linux, falling back to $HOME/.cache.
// This will be %LocalAppData% for windows.
//
// See https://specifications.freedesktop.org/basedir-spec/basedir-spec-latest.html
// for darwin and linux. Note that we use the same for darwin and linux as this is
// what developers expect, as opposed to ~/Library/Preferences etc as the stdlib
// does for Go.
func getDefaultCacheDirPath() (string, error) {
	if value := os.Getenv("XDG_CACHE_HOME"); value != "" {
		return value, nil
	}
	if value := os.Getenv("HOME"); value != "" {
		return filepath.Join(value, ".cache"), nil
	}
	return "", errors.New("$XDG_CACHE_HOME and $HOME are not set")
}
