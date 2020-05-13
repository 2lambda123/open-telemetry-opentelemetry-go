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

package metric

import (
	"errors"
	"fmt"
	"os"
	"sync"

	"go.opentelemetry.io/otel/api/core"
	"go.opentelemetry.io/otel/api/metric"
)

var ErrInvalidAsyncRunner = errors.New("unknown async runner type")

// AsyncCollector is an interface used between the MeterImpl and the
// AsyncInstrumentState helper below.  This interface is implemented by
// the SDK to provide support for running observer callbacks.
type AsyncCollector interface {
	// CollectAsync passes a batch of observations to the MeterImpl.
	CollectAsync([]core.KeyValue, []metric.Observation)
}

// AsyncInstrumentState manages an ordered set of asynchronous
// instruments and the distinct runners, taking into account batch
// observer callbacks.
type AsyncInstrumentState struct {
	lock sync.Mutex

	// errorHandler will be called in case of an invalid
	// metric.AsyncRunner, i.e., one that does not implement
	// either the single-or batch-runner interfaces.
	errorHandler func(error)
	errorOnce    sync.Once

	// runnerMap keeps the set of runners that will run each
	// collection interval.  Singletons are entered with a real
	// instrument each, batch observers are entered with a nil
	// instrument, ensuring that when a singleton callback is used
	// repeatedly, it is excuted repeatedly in the interval, while
	// when a batch callback is used repeatedly, it only executes
	// once per interval.
	runnerMap map[asyncRunnerPair]struct{}

	// runners maintains the set of runners in the order they were
	// registered.
	runners []asyncRunnerPair

	// instruments maintains the set of instruments in the order
	// they were registered.
	instruments []metric.AsyncImpl
}

// asyncRunnerPair is a map entry for Observer callback runners.
type asyncRunnerPair struct {
	// runner is used as a map key here.  The API ensures
	// that all callbacks are pointers for this reason.
	runner metric.AsyncRunner

	// inst refers to a non-nil instrument when `runner` is a
	// AsyncSingleRunner.
	inst metric.AsyncImpl
}

// NewAsyncInstrumentState returns a new *AsyncInstrumentState, for
// use by MeterImpl to manage running the set of observer callbacks in
// the correct order.
//
// errorHandler is used to print an error condition.  If errorHandler
// nil, the default error handler will be used that prints to
// os.Stderr.  Only the first error is passed to the handler, after
// which errors are skipped.
func NewAsyncInstrumentState(errorHandler func(error)) *AsyncInstrumentState {
	if errorHandler == nil {
		errorHandler = func(err error) {
			fmt.Fprintln(os.Stderr, "Metrics Async state error:", err)
		}
	}
	return &AsyncInstrumentState{
		errorHandler: errorHandler,
		runnerMap:    map[asyncRunnerPair]struct{}{},
	}
}

// Instruments returns the asynchronous instruments managed by this
// object, the set that should be checkpointed after observers are
// run.
func (a *AsyncInstrumentState) Instruments() []metric.AsyncImpl {
	a.lock.Lock()
	defer a.lock.Unlock()
	return a.instruments
}

// Register adds a new asynchronous instrument to by managed by this
// object.  This should be called during NewAsyncInstrument() and
// assumes that errors (e.g., duplicate registration) have already
// been checked.
func (a *AsyncInstrumentState) Register(inst metric.AsyncImpl, runner metric.AsyncRunner) {
	a.lock.Lock()
	defer a.lock.Unlock()

	a.instruments = append(a.instruments, inst)

	// asyncRunnerPair reflects this callback in the asyncRunners
	// list.  If this is a batch runner, the instrument is nil.
	// If this is a single-Observer runner, the instrument is
	// included.  This ensures that batch callbacks are called
	// once and single callbacks are called once per instrument.
	rp := asyncRunnerPair{
		runner: runner,
	}
	if _, ok := runner.(metric.AsyncSingleRunner); ok {
		rp.inst = inst
	}

	if _, ok := a.runnerMap[rp]; !ok {
		a.runnerMap[rp] = struct{}{}
		a.runners = append(a.runners, rp)
	}
}

// Run executes the complete set of observer callbacks.
func (a *AsyncInstrumentState) Run(collector AsyncCollector) {
	a.lock.Lock()
	runners := a.runners
	a.lock.Unlock()

	for _, rp := range runners {
		// The runner must be a single or batch runner, no
		// other implementations are possible because the
		// interface has un-exported methods.

		if singleRunner, ok := rp.runner.(metric.AsyncSingleRunner); ok {
			singleRunner.Run(rp.inst, collector.CollectAsync)
			continue
		}

		if multiRunner, ok := rp.runner.(metric.AsyncBatchRunner); ok {
			multiRunner.Run(collector.CollectAsync)
			continue
		}

		a.errorOnce.Do(func() {
			a.errorHandler(fmt.Errorf("%w: type %T (reported once)", ErrInvalidAsyncRunner, rp))
		})
	}
}
