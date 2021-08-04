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

package trace

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/internal/internaltest"
)

func TestVersionSemver(t *testing.T) {
	v := version()
	assert.NotNil(t, internaltest.VersionRegex.FindStringSubmatch(v), "version is not semver: %s", v)
}
