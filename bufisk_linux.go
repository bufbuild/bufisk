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

// Matching the unix-like build tags *minus darwin* in the Golang source i.e. https://github.com/golang/go/blob/912f0750472dd4f674b69ca1616bfaf377af1805/src/os/file_unix.go#L6
//
// We expanded this to all unix-like platforms, including those we don't support, as most
// of this should work without issue, and there are bigger problems with supporting i.e. js,wasm
// that are outside the scope of these build tags. Being able to build buf on i.e. openbsd
// was a blocker, see https://github.com/bufbuild/buf/issues/362 and the linked discussions.
// We still only officially support linux and darwin for buf as a whole.

//go:build aix || dragonfly || freebsd || (js && wasm) || linux || netbsd || openbsd || solaris

package main

const unameS = "Linux"
