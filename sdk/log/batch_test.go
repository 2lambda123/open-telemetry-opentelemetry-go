// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package log

import (
	"context"
	"slices"
	"strconv"
	"sync"
	"testing"
	"time"
	"unsafe"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/log"
)

func TestNewBatchingConfig(t *testing.T) {
	otel.SetErrorHandler(otel.ErrorHandlerFunc(func(err error) {
		t.Log(err)
	}))

	testcases := []struct {
		name    string
		envars  map[string]string
		options []BatchingOption
		want    batchingConfig
	}{
		{
			name: "Defaults",
			want: batchingConfig{
				maxQSize:        newSetting(dfltMaxQSize),
				expInterval:     newSetting(dfltExpInterval),
				expTimeout:      newSetting(dfltExpTimeout),
				expMaxBatchSize: newSetting(dfltExpMaxBatchSize),
			},
		},
		{
			name: "Options",
			options: []BatchingOption{
				WithMaxQueueSize(1),
				WithExportInterval(time.Microsecond),
				WithExportTimeout(time.Hour),
				WithExportMaxBatchSize(2),
			},
			want: batchingConfig{
				maxQSize:        newSetting(1),
				expInterval:     newSetting(time.Microsecond),
				expTimeout:      newSetting(time.Hour),
				expMaxBatchSize: newSetting(2),
			},
		},
		{
			name: "Environment",
			envars: map[string]string{
				envarMaxQSize:        strconv.Itoa(1),
				envarExpInterval:     strconv.Itoa(100),
				envarExpTimeout:      strconv.Itoa(1000),
				envarExpMaxBatchSize: strconv.Itoa(10),
			},
			want: batchingConfig{
				maxQSize:        newSetting(1),
				expInterval:     newSetting(100 * time.Millisecond),
				expTimeout:      newSetting(1000 * time.Millisecond),
				expMaxBatchSize: newSetting(10),
			},
		},
		{
			name: "InvalidOptions",
			options: []BatchingOption{
				WithMaxQueueSize(-11),
				WithExportInterval(-1 * time.Microsecond),
				WithExportTimeout(-1 * time.Hour),
				WithExportMaxBatchSize(-2),
			},
			want: batchingConfig{
				maxQSize:        newSetting(dfltMaxQSize),
				expInterval:     newSetting(dfltExpInterval),
				expTimeout:      newSetting(dfltExpTimeout),
				expMaxBatchSize: newSetting(dfltExpMaxBatchSize),
			},
		},
		{
			name: "InvalidEnvironment",
			envars: map[string]string{
				envarMaxQSize:        "-1",
				envarExpInterval:     "-1",
				envarExpTimeout:      "-1",
				envarExpMaxBatchSize: "-1",
			},
			want: batchingConfig{
				maxQSize:        newSetting(dfltMaxQSize),
				expInterval:     newSetting(dfltExpInterval),
				expTimeout:      newSetting(dfltExpTimeout),
				expMaxBatchSize: newSetting(dfltExpMaxBatchSize),
			},
		},
		{
			name: "Precedence",
			envars: map[string]string{
				envarMaxQSize:        strconv.Itoa(1),
				envarExpInterval:     strconv.Itoa(100),
				envarExpTimeout:      strconv.Itoa(1000),
				envarExpMaxBatchSize: strconv.Itoa(10),
			},
			options: []BatchingOption{
				// These override the environment variables.
				WithMaxQueueSize(3),
				WithExportInterval(time.Microsecond),
				WithExportTimeout(time.Hour),
				WithExportMaxBatchSize(2),
			},
			want: batchingConfig{
				maxQSize:        newSetting(3),
				expInterval:     newSetting(time.Microsecond),
				expTimeout:      newSetting(time.Hour),
				expMaxBatchSize: newSetting(2),
			},
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			for key, value := range tc.envars {
				t.Setenv(key, value)
			}
			assert.Equal(t, tc.want, newBatchingConfig(tc.options))
		})
	}
}

func TestBatchingProcessor(t *testing.T) {
	ctx := context.Background()

	t.Run("Polling", func(t *testing.T) {
		e := newTestExporter(nil)
		const size = 15
		b := NewBatchingProcessor(
			e,
			WithMaxQueueSize(2*size),
			WithExportMaxBatchSize(2*size),
			WithExportInterval(time.Nanosecond),
			WithExportTimeout(time.Hour),
		)
		for _, r := range make([]Record, size) {
			assert.NoError(t, b.OnEmit(ctx, r))
		}
		var got []Record
		assert.Eventually(t, func() bool {
			for _, r := range e.Records() {
				got = append(got, r...)
			}
			return len(got) == size
		}, 2*time.Second, time.Microsecond)
		_ = b.Shutdown(ctx)
	})

	t.Run("OnEmit", func(t *testing.T) {
		e := newTestExporter(nil)
		b := NewBatchingProcessor(
			e,
			WithMaxQueueSize(100),
			WithExportMaxBatchSize(10),
			WithExportInterval(time.Hour),
			WithExportTimeout(time.Hour),
		)
		for _, r := range make([]Record, 15) {
			assert.NoError(t, b.OnEmit(ctx, r))
		}
		assert.NoError(t, b.Shutdown(ctx))

		assert.Equal(t, 2, e.ExportN())
	})

	t.Run("Enabled", func(t *testing.T) {
		b := NewBatchingProcessor(defaultNoopExporter)
		assert.True(t, b.Enabled(ctx, Record{}))

		_ = b.Shutdown(ctx)
		assert.False(t, b.Enabled(ctx, Record{}))
	})

	t.Run("Shutdown", func(t *testing.T) {
		t.Run("Error", func(t *testing.T) {
			e := newTestExporter(assert.AnError)
			b := NewBatchingProcessor(e)
			assert.ErrorIs(t, b.Shutdown(ctx), assert.AnError, "exporter error not returned")
			assert.NoError(t, b.Shutdown(ctx))
		})

		t.Run("Multiple", func(t *testing.T) {
			e := newTestExporter(nil)
			b := NewBatchingProcessor(e)

			const shutdowns = 3
			for i := 0; i < shutdowns; i++ {
				assert.NoError(t, b.Shutdown(ctx))
			}
			assert.Equal(t, 1, e.ShutdownN(), "exporter Shutdown calls")
		})

		t.Run("OnEmit", func(t *testing.T) {
			e := newTestExporter(nil)
			b := NewBatchingProcessor(e)
			assert.NoError(t, b.Shutdown(ctx))

			want := e.ExportN()
			assert.NoError(t, b.OnEmit(ctx, Record{}))
			assert.Equal(t, want, e.ExportN(), "Export called after shutdown")
		})

		t.Run("ForceFlush", func(t *testing.T) {
			e := newTestExporter(nil)
			b := NewBatchingProcessor(e)

			assert.NoError(t, b.OnEmit(ctx, Record{}))
			assert.NoError(t, b.Shutdown(ctx))

			assert.NoError(t, b.ForceFlush(ctx))
			assert.Equal(t, 0, e.ForceFlushN(), "ForceFlush called after shutdown")
		})

		t.Run("CanceledContext", func(t *testing.T) {
			e := newTestExporter(nil)
			e.ExportTrigger = make(chan struct{})
			t.Cleanup(func() { close(e.ExportTrigger) })
			b := NewBatchingProcessor(e)

			ctx := context.Background()
			c, cancel := context.WithCancel(ctx)
			cancel()

			assert.ErrorIs(t, b.Shutdown(c), context.Canceled)
		})
	})

	t.Run("ForceFlush", func(t *testing.T) {
		t.Run("Flush", func(t *testing.T) {
			e := newTestExporter(assert.AnError)
			b := NewBatchingProcessor(
				e,
				WithMaxQueueSize(100),
				WithExportMaxBatchSize(10),
				WithExportInterval(time.Hour),
				WithExportTimeout(time.Hour),
			)
			t.Cleanup(func() { _ = b.Shutdown(ctx) })

			var r Record
			r.SetBody(log.BoolValue(true))
			require.NoError(t, b.OnEmit(ctx, r))

			assert.ErrorIs(t, b.ForceFlush(ctx), assert.AnError, "exporter error not returned")
			assert.Equal(t, 1, e.ForceFlushN(), "exporter ForceFlush calls")
			if assert.Equal(t, 1, e.ExportN(), "exporter Export calls") {
				got := e.Records()
				if assert.Len(t, got[0], 1, "records received") {
					assert.Equal(t, r, got[0][0])
				}
			}
		})

		t.Run("CanceledContext", func(t *testing.T) {
			e := newTestExporter(nil)
			e.ExportTrigger = make(chan struct{})
			b := NewBatchingProcessor(e)
			t.Cleanup(func() { _ = b.Shutdown(ctx) })

			var r Record
			r.SetBody(log.BoolValue(true))
			_ = b.OnEmit(ctx, r)
			t.Cleanup(func() { _ = b.Shutdown(ctx) })
			t.Cleanup(func() { close(e.ExportTrigger) })

			c, cancel := context.WithCancel(ctx)
			cancel()
			assert.ErrorIs(t, b.ForceFlush(c), context.Canceled)
		})
	})

	t.Run("ConcurrentSafe", func(t *testing.T) {
		const goRoutines = 10

		e := newTestExporter(nil)
		b := NewBatchingProcessor(e)
		stop := make(chan struct{})
		var wg sync.WaitGroup
		for i := 0; i < goRoutines-1; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for {
					select {
					case <-stop:
						return
					default:
						assert.NoError(t, b.OnEmit(ctx, Record{}))
						assert.NoError(t, b.ForceFlush(ctx))
					}
				}
			}()
		}

		require.Eventually(t, func() bool {
			return e.ExportN() > 0
		}, 2*time.Second, time.Microsecond, "export before shutdown")

		wg.Add(1)
		go func() {
			defer wg.Done()
			assert.NoError(t, b.Shutdown(ctx))
			close(stop)
		}()

		wg.Wait()
	})
}

func TestQueue(t *testing.T) {
	var r Record
	r.SetBody(log.BoolValue(true))

	t.Run("newQueue", func(t *testing.T) {
		const size = 1
		q := newQueue(size)
		assert.Equal(t, q.len, 0)
		assert.Equal(t, size, q.cap, "capacity")
		assert.Equal(t, size, q.read.Len(), "read ring")
		assert.Same(t, q.read, q.write, "different rings")
	})

	t.Run("Enqueue", func(t *testing.T) {
		const size = 2
		q := newQueue(size)

		var notR Record
		notR.SetBody(log.IntValue(10))

		assert.Equal(t, 1, q.Enqueue(notR), "incomplete batch")
		assert.Equal(t, 1, q.len, "length")
		assert.Equal(t, size, q.cap, "capacity")

		assert.Equal(t, 2, q.Enqueue(r), "complete batch")
		assert.Equal(t, 2, q.len, "length")
		assert.Equal(t, size, q.cap, "capacity")

		assert.Equal(t, 2, q.Enqueue(r), "overflow batch")
		assert.Equal(t, 2, q.len, "length")
		assert.Equal(t, size, q.cap, "capacity")

		assert.Equal(t, []Record{r, r}, q.Flush(), "flushed Records")
	})

	t.Run("Flush", func(t *testing.T) {
		const size = 2
		q := newQueue(size)
		q.write.Value = r
		q.write = q.write.Next()
		q.len = 1

		assert.Equal(t, []Record{r}, q.Flush(), "flushed")
	})

	t.Run("TryFlush", func(t *testing.T) {
		const size = 3
		q := newQueue(size)
		for i := 0; i < size-1; i++ {
			q.write.Value = r
			q.write = q.write.Next()
			q.len++
		}

		buf := make([]Record, 1)
		f := func([]Record) bool { return false }
		assert.Equal(t, size-1, q.TryDequeue(buf, f), "not flushed")
		require.Equal(t, size-1, q.len, "length")
		require.NotSame(t, q.read, q.write, "read ring advanced")

		var flushed []Record
		f = func(r []Record) bool {
			flushed = append(flushed, r...)
			return true
		}
		if assert.Equal(t, size-2, q.TryDequeue(buf, f), "did not flush len(buf)") {
			assert.Equal(t, []Record{r}, flushed, "Records")
		}

		buf = slices.Grow(buf, size)
		flushed = flushed[:0]
		if assert.Equal(t, 0, q.TryDequeue(buf, f), "did not flush len(queue)") {
			assert.Equal(t, []Record{r}, flushed, "Records")
		}
	})

	t.Run("ConcurrentSafe", func(t *testing.T) {
		const goRoutines = 10

		flushed := make(chan []Record, goRoutines)
		out := make([]Record, 0, goRoutines)
		done := make(chan struct{})
		go func() {
			defer close(done)
			for recs := range flushed {
				out = append(out, recs...)
			}
		}()

		var wg sync.WaitGroup
		wg.Add(goRoutines)

		b := newQueue(goRoutines)
		for i := 0; i < goRoutines; i++ {
			go func() {
				defer wg.Done()
				b.Enqueue(Record{})
				flushed <- b.Flush()
			}()
		}

		wg.Wait()
		close(flushed)
		<-done

		assert.Len(t, out, goRoutines, "flushed Records")
	})
}

func BenchmarkBatchingProcessor(b *testing.B) {
	var r Record
	body := log.BoolValue(true)
	r.SetBody(body)

	rSize := unsafe.Sizeof(r) + unsafe.Sizeof(body)
	b.Logf("record size: %d", rSize)

	ctx := context.Background()

	b.Run("OnEmit", func(b *testing.B) {
		bp := NewBatchingProcessor(
			defaultNoopExporter,
			WithMaxQueueSize(b.N),
			WithExportMaxBatchSize(b.N),
			WithExportInterval(time.Hour),
			WithExportTimeout(time.Hour),
		)

		b.SetBytes(int64(rSize))
		b.ReportAllocs()
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			var err error
			for pb.Next() {
				err = bp.OnEmit(ctx, r)
			}
			_ = err
		})
	})

	b.Run("Flush", func(b *testing.B) {
		bp := NewBatchingProcessor(defaultNoopExporter)

		b.SetBytes(int64(rSize))
		b.ReportAllocs()
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			var err error
			for pb.Next() {
				_ = bp.OnEmit(ctx, r)
				err = bp.ForceFlush(ctx)
			}
			_ = err
		})
	})
}
