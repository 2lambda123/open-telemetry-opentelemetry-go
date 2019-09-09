// Copyright 2019, OpenTelemetry Authors
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
	"time"

	"go.opentelemetry.io/api/core"
)

// Event is used to describe an Event with a message string and set of
// Attributes.
type Event struct {
	// Message describes the Event.
	Message string

	// Attributes contains a list of keyvalue pairs.
	Attributes []core.KeyValue

	// Time is the time at which this event was recorded.
	Time time.Time
}
