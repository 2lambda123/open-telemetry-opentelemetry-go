// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package log // import "go.opentelemetry.io/otel/sdk/log"

import (
	"context"
	"errors"
	"slices"
	"sync"
	"sync/atomic"
	"time"
)

const (
	dfltMaxQSize        = 2048
	dfltExpInterval     = time.Second
	dfltExpTimeout      = 30 * time.Second
	dfltExpMaxBatchSize = 512

	envarMaxQSize        = "OTEL_BLRP_MAX_QUEUE_SIZE"
	envarExpInterval     = "OTEL_BLRP_SCHEDULE_DELAY"
	envarExpTimeout      = "OTEL_BLRP_EXPORT_TIMEOUT"
	envarExpMaxBatchSize = "OTEL_BLRP_MAX_EXPORT_BATCH_SIZE"
)

// Compile-time check BatchingProcessor implements Processor.
var _ Processor = (*BatchingProcessor)(nil)

// BatchingProcessor is a processor that exports batches of log records.
type BatchingProcessor struct {
	// exporter is the bufferedExporter all batches are exported with.
	exporter *bufferExporter

	// q is the active queue of records that have not yet been exported.
	q *queue
	// batchSize is the minimum number of Records needed before an export is
	// triggered (unless the interval expires).
	batchSize int

	// pollTrigger triggers the poll goroutine to flush a batch from the queue.
	// This is sent to when it is known that the queue contains at least one
	// complete batch.
	//
	// When a send is made to the channel, the poll loop will be reset after
	// the flush. If there is still enough Records in the queue for another
	// batch the reset of the poll loop will automatically re-trigger itself.
	// There is no need for the original sender to monitor and resend.
	pollTrigger chan struct{}
	// pollKill kills the poll goroutine. This is only expected to be closed
	// once by the Shutdown method.
	pollKill chan struct{}
	// pollDone signals the poll goroutine has completed.
	pollDone chan struct{}

	// stopped holds the stopped state of the BatchingProcessor.
	stopped atomic.Bool
}

// NewBatchingProcessor decorates the provided exporter
// so that the log records are batched before exporting.
//
// All of the exporter's methods are called synchronously.
func NewBatchingProcessor(exporter Exporter, opts ...BatchingOption) *BatchingProcessor {
	cfg := newBatchingConfig(opts)

	if exporter == nil {
		// Do not panic on nil export.
		exporter = defaultNoopExporter
	}
	// Order is important here. Wrap the timeoutExporter with the chuncker to
	// ensure each export completes in timeout (instead of all chuncked
	// exports).
	exporter = newTimeoutExporter(exporter, cfg.expTimeout.Value)
	// Use a chunkExporter to ensure ForceFlush and Shutdown calls are batched
	// appropriately on export.
	exporter = newChunkExporter(exporter, cfg.expMaxBatchSize.Value)

	b := &BatchingProcessor{
		// TODO: explore making the size of this configurable.
		exporter: newBufferExporter(exporter, 1),

		q:           newQueue(cfg.maxQSize.Value),
		batchSize:   cfg.expMaxBatchSize.Value,
		pollTrigger: make(chan struct{}, 1),
		pollKill:    make(chan struct{}),
	}
	b.pollDone = b.poll(cfg.expInterval.Value)
	return b
}

// poll spawns a goroutine to handle interval polling and batch exporting. The
// returned done chan is closed when the spawned goroutine completes.
func (b *BatchingProcessor) poll(interval time.Duration) (done chan struct{}) {
	done = make(chan struct{})

	ticker := time.NewTicker(interval)
	// TODO: investigate using a sync.Pool instead of cloning.
	buf := make([]Record, b.batchSize)
	go func() {
		defer close(done)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				// TODO: handle premature ticks. If the oldest record is
				// younger than interval, do not export batch.
			case <-b.pollTrigger:
				ticker.Reset(interval)
			case <-b.pollKill:
				return
			}

			qLen := b.q.TryDequeue(buf, func(r []Record) bool {
				ok := b.exporter.EnqueueExport(r)
				if ok {
					buf = slices.Clone(buf)
				}
				return ok
			})
			if qLen >= b.batchSize {
				select {
				case b.pollTrigger <- struct{}{}:
				default:
					// Another flush signal already received.
				}
			}
		}
	}()
	return done
}

// OnEmit batches provided log record.
func (b *BatchingProcessor) OnEmit(_ context.Context, r Record) error {
	if b.stopped.Load() {
		return nil
	}
	if n := b.q.Enqueue(r); n >= b.batchSize {
		select {
		case b.pollTrigger <- struct{}{}:
		default:
			// Flush chan full. The poll goroutine will handle this by
			// re-sending any trigger until the queue has less than batchSize
			// records.
		}
	}
	return nil
}

// Enabled returns if b is enabled.
func (b *BatchingProcessor) Enabled(context.Context, Record) bool {
	return !b.stopped.Load()
}

// Shutdown flushes queued log records and shuts down the decorated exporter.
func (b *BatchingProcessor) Shutdown(ctx context.Context) error {
	if b.stopped.Swap(true) {
		return nil
	}

	// Stop the poll goroutine.
	close(b.pollKill)
	select {
	case <-b.pollDone:
	case <-ctx.Done():
		// Out of time. Do not close b.exportCh, it is not certain if the poll
		// goroutine will try to send to it still.
		return errors.Join(ctx.Err(), b.exporter.Shutdown(ctx))
	}

	// Flush remaining queued before exporter shutdown.
	//
	// Given the poll goroutine has stopped we know no more data will be
	// queued. This ensures concurrent calls to ForceFlush do not panic because
	// they are flusing to a shut down exporter.
	err := b.exporter.Export(ctx, b.q.Flush())
	return errors.Join(err, b.exporter.Shutdown(ctx))
}

// ForceFlush flushes queued log records and flushes the decorated exporter.
func (b *BatchingProcessor) ForceFlush(ctx context.Context) error {
	if b.stopped.Load() {
		return nil
	}
	err := b.exporter.Export(ctx, b.q.Flush())
	return errors.Join(err, b.exporter.ForceFlush(ctx))
}

// queue holds a queue of logging records.
//
// When the queue becomes full, the oldest records in the queue are
// overwritten.
type queue struct {
	sync.Mutex

	cap, len    int
	read, write *ring
}

func newQueue(size int) *queue {
	r := newRing(size)
	return &queue{
		cap:   size,
		read:  r,
		write: r,
	}
}

// Enqueue adds r to the queue. The queue size, including the addition of r, is
// returned.
//
// If enqueueing r will exceed the capacity of q, the oldest Record held in q
// will be dropped and r retained.
func (q *queue) Enqueue(r Record) int {
	q.Lock()
	defer q.Unlock()

	q.write.Value = r
	q.write = q.write.Next()

	q.len++
	if q.len > q.cap {
		// Overflow. Advance read to be the new "oldest".
		q.len = q.cap
		q.read = q.read.Next()
	}
	return q.len
}

// TryDequeue attempts to dequeue up to len(buf) Records. The available Records
// will be assigned into buf and passed to write. If write fails, returning
// false, the Records will not be removed from the queue. If write succeeds,
// returning true, the dequeued Records are removed from the queue. The number
// of Records remaining in the queue are returned.
//
// When write is called the lock of q is held. The write function must not call
// other methods of this q that acquire the lock.
func (q *queue) TryDequeue(buf []Record, write func([]Record) bool) int {
	q.Lock()
	defer q.Unlock()

	origRead := q.read

	n := min(len(buf), q.len)
	for i := 0; i < n; i++ {
		buf[i] = q.read.Value
		q.read = q.read.Next()
	}

	if write(buf[:n]) {
		q.len -= n
	} else {
		q.read = origRead
	}
	return q.len
}

// Flush returns all the Records held in the queue and resets it to be
// empty.
func (q *queue) Flush() []Record {
	q.Lock()
	defer q.Unlock()

	out := make([]Record, q.len)
	for i := range out {
		out[i] = q.read.Value
		q.read = q.read.Next()
	}
	q.len = 0

	return out
}

// A ring is an element of a circular list, or ring. Rings do not have a
// beginning or end; a pointer to any ring element serves as reference to the
// entire ring. Empty rings are represented as nil ring pointers. The zero
// value for a ring is a one-element ring with a nil Value.
//
// This is copied from the "container/ring" package. It uses a Record type for
// Value instead of any to avoid allocations.
type ring struct {
	next, prev *ring
	Value      Record
}

func (r *ring) init() *ring {
	r.next = r
	r.prev = r
	return r
}

// Next returns the next ring element. r must not be empty.
func (r *ring) Next() *ring {
	if r.next == nil {
		return r.init()
	}
	return r.next
}

// Prev returns the previous ring element. r must not be empty.
func (r *ring) Prev() *ring {
	if r.next == nil {
		return r.init()
	}
	return r.prev
}

// newRing creates a ring of n elements.
func newRing(n int) *ring {
	if n <= 0 {
		return nil
	}
	r := new(ring)
	p := r
	for i := 1; i < n; i++ {
		p.next = &ring{prev: p}
		p = p.next
	}
	p.next = r
	r.prev = p
	return r
}

// Len computes the number of elements in ring r. It executes in time
// proportional to the number of elements.
func (r *ring) Len() int {
	n := 0
	if r != nil {
		n = 1
		for p := r.Next(); p != r; p = p.next {
			n++
		}
	}
	return n
}

// Do calls function f on each element of the ring, in forward order. The
// behavior of Do is undefined if f changes *r.
func (r *ring) Do(f func(Record)) {
	if r != nil {
		f(r.Value)
		for p := r.Next(); p != r; p = p.next {
			f(p.Value)
		}
	}
}

type batchingConfig struct {
	maxQSize        setting[int]
	expInterval     setting[time.Duration]
	expTimeout      setting[time.Duration]
	expMaxBatchSize setting[int]
}

func newBatchingConfig(options []BatchingOption) batchingConfig {
	var c batchingConfig
	for _, o := range options {
		c = o.apply(c)
	}

	c.maxQSize = c.maxQSize.Resolve(
		clearLessThanOne[int](),
		getenv[int](envarMaxQSize),
		clearLessThanOne[int](),
		fallback[int](dfltMaxQSize),
	)
	c.expInterval = c.expInterval.Resolve(
		clearLessThanOne[time.Duration](),
		getenv[time.Duration](envarExpInterval),
		clearLessThanOne[time.Duration](),
		fallback[time.Duration](dfltExpInterval),
	)
	c.expTimeout = c.expTimeout.Resolve(
		clearLessThanOne[time.Duration](),
		getenv[time.Duration](envarExpTimeout),
		clearLessThanOne[time.Duration](),
		fallback[time.Duration](dfltExpTimeout),
	)
	c.expMaxBatchSize = c.expMaxBatchSize.Resolve(
		clearLessThanOne[int](),
		getenv[int](envarExpMaxBatchSize),
		clearLessThanOne[int](),
		fallback[int](dfltExpMaxBatchSize),
	)

	return c
}

// BatchingOption applies a configuration to a [BatchingProcessor].
type BatchingOption interface {
	apply(batchingConfig) batchingConfig
}

type batchingOptionFunc func(batchingConfig) batchingConfig

func (fn batchingOptionFunc) apply(c batchingConfig) batchingConfig {
	return fn(c)
}

// WithMaxQueueSize sets the maximum queue size used by the Batcher.
// After the size is reached log records are dropped.
//
// If the OTEL_BLRP_MAX_QUEUE_SIZE environment variable is set,
// and this option is not passed, that variable value will be used.
//
// By default, if an environment variable is not set, and this option is not
// passed, 2048 will be used.
// The default value is also used when the provided value is less than one.
func WithMaxQueueSize(size int) BatchingOption {
	return batchingOptionFunc(func(cfg batchingConfig) batchingConfig {
		cfg.maxQSize = newSetting(size)
		return cfg
	})
}

// WithExportInterval sets the maximum duration between batched exports.
//
// If the OTEL_BLRP_SCHEDULE_DELAY environment variable is set,
// and this option is not passed, that variable value will be used.
//
// By default, if an environment variable is not set, and this option is not
// passed, 1s will be used.
// The default value is also used when the provided value is less than one.
func WithExportInterval(d time.Duration) BatchingOption {
	return batchingOptionFunc(func(cfg batchingConfig) batchingConfig {
		cfg.expInterval = newSetting(d)
		return cfg
	})
}

// WithExportTimeout sets the duration after which a batched export is canceled.
//
// If the OTEL_BLRP_EXPORT_TIMEOUT environment variable is set,
// and this option is not passed, that variable value will be used.
//
// By default, if an environment variable is not set, and this option is not
// passed, 30s will be used.
// The default value is also used when the provided value is less than one.
func WithExportTimeout(d time.Duration) BatchingOption {
	return batchingOptionFunc(func(cfg batchingConfig) batchingConfig {
		cfg.expTimeout = newSetting(d)
		return cfg
	})
}

// WithExportMaxBatchSize sets the maximum batch size of every export.
// A batch will be split into multiple exports to not exceed this size.
//
// If the OTEL_BLRP_MAX_EXPORT_BATCH_SIZE environment variable is set,
// and this option is not passed, that variable value will be used.
//
// By default, if an environment variable is not set, and this option is not
// passed, 512 will be used.
// The default value is also used when the provided value is less than one.
func WithExportMaxBatchSize(size int) BatchingOption {
	return batchingOptionFunc(func(cfg batchingConfig) batchingConfig {
		cfg.expMaxBatchSize = newSetting(size)
		return cfg
	})
}
