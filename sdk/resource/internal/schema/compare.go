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

package schema

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/Masterminds/semver/v3"
)

type Comparison uint8

const (
	invalidComparison Comparison = iota
	EqualTo
	GreaterThan
	LessThan
)

type errInvalidVer struct {
	ver string
	err error
}

func (e *errInvalidVer) Error() string {
	return fmt.Sprintf("invalid version for %q: %s", e.ver, e.err)
}

// CompareVersions compares schema URL versions and returns the Comparison of a
// vs b (i.e. a is [comparison value] b).
func CompareVersions(a, b string) (Comparison, error) {
	aVer, err := version(a)
	if err != nil {
		return invalidComparison, &errInvalidVer{ver: a, err: err}
	}

	bVer, err := version(b)
	if err != nil {
		return invalidComparison, &errInvalidVer{ver: b, err: err}
	}

	switch aVer.Compare(bVer) {
	case -1:
		return LessThan, nil
	case 0:
		return EqualTo, nil
	case 1:
		return GreaterThan, nil
	default:
		msg := fmt.Sprintf("unknown comparison: %q, %q", aVer, bVer)
		panic(msg)
	}
}

func version(schemaURL string) (*semver.Version, error) {
	// https://github.com/open-telemetry/oteps/blob/main/text/0152-telemetry-schemas.md#schema-url
	u, err := url.Parse(schemaURL)
	if err != nil {
		return nil, err
	}

	return semver.NewVersion(u.Path[strings.LastIndex(u.Path, "/")+1:])
}
