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

// Package schema provides conversion and comparison utilities OpenTelemetry
// resource schemas.
package schema // import "go.opentelemetry.io/otel/sdk/resource/internal/schema"

//go:generate go run go.opentelemetry.io/otel/sdk/resource/internal/schema/generate ./transforms.go

import (
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/Masterminds/semver/v3"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/schema/v1.0/ast"
)

var errInvalid = errors.New("invalid schema URL")

// Version returns the version of schemaURL. If schemaURL is not and
// OpenTelemetry schemaURL or an otherwise invalid URL, an error is returned.
func Version(schemaURL string) (*semver.Version, error) {
	if schemaURL == "" {
		return nil, fmt.Errorf("%w: empty", errInvalid)
	}

	// https://github.com/open-telemetry/oteps/blob/main/text/0152-telemetry-schemas.md#schema-url
	u, err := url.Parse(schemaURL)
	if err != nil {
		return nil, err
	}

	if h := u.Hostname(); h != "opentelemetry.io" {
		return nil, fmt.Errorf("%w: host not \"opentelemetry.io\": %s", errInvalid, h)
	}

	i := strings.LastIndex(u.Path, "/")
	if p := u.Path[:i]; p != "/schemas" {
		return nil, fmt.Errorf("%w: path not \"/schemas\": %s", errInvalid, p)
	}

	return semver.NewVersion(u.Path[i+1:])
}

// Upgrade upgrades attrs in place. The upgrade will be done from the schemaURL
// version to the target schemaURL version using the schema translations
// defined in target.
//
// If schemaURL is version already greater than target, no upgrade will be
// performed on attrs.
func Upgrade(from, to *semver.Version, attrs []attribute.KeyValue) error {
	return upgrade(slice(transforms, from, to), attrs)
}

func upgrade(tforms []transform, attrs []attribute.KeyValue) error {
	a := newAttributes(attrs)
	for _, t := range tforms {
		// Transformations in section "all" always are applied first.
		for _, c := range t.All.Changes {
			for key, val := range c.RenameAttributes.AttributeMap {
				a.Rename(key, val)
			}
		}
		// Only other applicable section is for resources.
		for _, c := range t.Resources.Changes {
			for key, val := range c.RenameAttributes.AttributeMap {
				a.Rename(key, val)
			}
		}
	}

	return nil
}

type transform struct {
	Version   *semver.Version
	All       ast.Attributes
	Resources ast.Attributes
}

func slice(tforms []transform, min, max *semver.Version) []transform {
	if min.GreaterThan(max) {
		return nil
	}

	low, high := -1, -1
	for i, t := range tforms {
		if low < 0 && (min.Equal(t.Version) || min.LessThan(t.Version)) {
			low = i
		}
		if high < 0 {
			if max.LessThan(t.Version) {
				high = i
				break
			}
		}
	}

	if low == -1 {
		// No transforms apply.
		return nil
	}

	if high == -1 {
		high = len(tforms)
	}

	return tforms[low:high]
}

type attributes struct {
	underlying []attribute.KeyValue

	index map[string][]int
}

func newAttributes(attr []attribute.KeyValue) *attributes {
	// Lazy built index.
	return &attributes{underlying: attr}
}

func (a *attributes) init() {
	a.index = make(map[string][]int, len(a.underlying))
	for i := range a.underlying {
		key := string(a.underlying[i].Key)
		a.index[key] = append(a.index[key], i)
	}
}

func (a *attributes) Rename(orig, repl string) {
	if a.index == nil {
		a.init()
	}

	index, ok := a.index[orig]
	if !ok {
		return
	}
	delete(a.index, orig)
	for _, i := range index {
		a.underlying[i].Key = attribute.Key(repl)
	}
	a.index[repl] = append(a.index[repl], index...)
}
