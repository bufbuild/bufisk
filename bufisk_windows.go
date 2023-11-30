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

//go:build windows

package main

import (
	"errors"
	"os"
)

const (
	unameS           = "Windows"
	executableSuffix = ".exe"
)

// extraInterruptSignals are signals beyond os.Interrupt that we want to be handled
// as interrupts.
//
// For unix-like platforms, this adds syscall.SIGTERM, although this is only
// tested on darwin and linux, which buf officially supports. Other unix-like
// platforms should have this as well, however.
var extraInterruptSignals = []os.Signal{}

// getDefaultCacheDirPath returns the cache directory path.
//
// This will be $XDG_CACHE_HOME for darwin and linux, falling back to $HOME/.cache.
// This will be %LocalAppData% for windows.
//
// See https://specifications.freedesktop.org/basedir-spec/basedir-spec-latest.html
// for darwin and linux. Note that we use the same for darwin and linux as this is
// what developers expect, as opposed to ~/Library/Preferences etc as the stdlib
// does for Go.
//
// Users cannot assume that CacheDirPath, ConfigDirPath, and DataDirPath are unique.
func getDefaultCacheDirPath() (string, error) {
	if value := os.Getenv("LOCALAPPDATA"); value != "" {
		return value, nil
	}
	return "", errors.New("%LocalAppData% is not set")
}
