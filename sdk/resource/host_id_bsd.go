// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

//go:build dragonfly || freebsd || netbsd || openbsd || solaris
// +build dragonfly freebsd netbsd openbsd solaris

package resource // import "go.opentelemetry.io/otel/sdk/resource"

import (
	"errors"
	"strings"
)

// hostIDReaderBSD implements hostIDReader
type hostIDReaderBSD struct {
	execCommand commandExecutor
	readFile    fileReader
}

// read attempts to read the machine-id from /etc/hostid. If not found it will
// execute `kenv -q smbios.system.uuid`. If neither location yields an id an
// error will be returned.
func (r *hostIDReaderBSD) read() (string, error) {
	if result, err := r.readFile("/etc/hostid"); err == nil {
		return strings.TrimSpace(result), nil
	}

	if result, err := r.execCommand("kenv", "-q", "smbios.system.uuid"); err == nil {
		return strings.TrimSpace(result), nil
	}

	return "", errors.New("host id not found in: /etc/hostid or kenv")
}

var platformHostIDReader hostIDReader = &hostIDReaderBSD{
	execCommand: execCommand,
	readFile:    readFile,
}
