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

package main

import (
	"embed"
	"flag"
	"log"
	"os"
	"path/filepath"
	"strings"

	"text/template"
)

var (
	out = flag.String("output", "./", "output directory")
	tag = flag.String("tag", "", "OpenTelemetry tagged version")

	//go:embed templates/*.tmpl
	rootFS embed.FS
)

// SemanticConventions are information about the semantic conventions being
// generated.
type SemanticConventions struct {
	// SemVer is the semantic version (i.e. 1.7.0 and not v1.7.0).
	SemVer string
	// TagVer is the tagged version (i.e. v1.7.0 and not 1.7.0).
	TagVer string
}

func render(dest string, sc *SemanticConventions) error {
	tmpls, err := template.ParseFS(rootFS, "templates/*.tmpl")
	if err != nil {
		return err
	}
	for _, tmpl := range tmpls.Templates() {
		target := filepath.Join(dest, strings.TrimSuffix(tmpl.Name(), ".tmpl"))
		wr, err := os.Create(target)
		if err != nil {
			return err
		}

		tmpl.Execute(wr, sc)
	}

	return nil
}

func main() {
	flag.Parse()

	if *tag == "" {
		log.Fatalf("invalid tag: %q", *tag)
	}

	sc := &SemanticConventions{
		SemVer: strings.TrimPrefix(*tag, "v"),
		TagVer: *tag,
	}

	if err := render(*out, sc); err != nil {
		log.Fatal(err)
	}
}
