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

package instrument // import "go.opentelemetry.io/otel/metric/instrument"

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/otel/attribute"
)

func TestInt64ObservableOptions(t *testing.T) {
	const (
		token  int64 = 43
		desc         = "Instrument description."
		uBytes       = "By"
	)

	got := NewInt64ObservableConfig(
		WithDescription(desc),
		WithUnit(uBytes),
		WithInt64Callback(func(_ context.Context, obsrv Int64Observer) error {
			obsrv.Observe(token)
			return nil
		}),
	)
	assert.Equal(t, desc, got.Description(), "description")
	assert.Equal(t, uBytes, got.Unit(), "unit")

	// Functions are not comparable.
	cBacks := got.Callbacks()
	require.Len(t, cBacks, 1, "callbacks")
	o := &int64Observer{}
	err := cBacks[0](context.Background(), o)
	require.NoError(t, err)
	assert.Equal(t, token, o.got, "callback not set")
}

type int64Observer struct {
	Asynchronous
	got int64
}

func (o *int64Observer) Observe(v int64, _ ...attribute.KeyValue) {
	o.got = v
}
