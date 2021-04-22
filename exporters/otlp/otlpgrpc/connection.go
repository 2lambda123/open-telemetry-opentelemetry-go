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

package otlpgrpc

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"

	"github.com/cenkalti/backoff/v4"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"google.golang.org/grpc/encoding/gzip"

	"go.opentelemetry.io/otel/exporters/otlp"
	"go.opentelemetry.io/otel/exporters/otlp/internal/otlpconfig"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

type connection struct {
	// Ensure pointer is 64-bit aligned for atomic operations on both 32 and 64 bit machines.
	lastConnectErrPtr unsafe.Pointer

	// mu protects the connection as it is accessed by the
	// exporter goroutines and background connection goroutine
	mu sync.Mutex
	cc *grpc.ClientConn

	// these fields are read-only after constructor is finished
	cfg                  otlpconfig.Config
	sCfg                 otlpconfig.SignalConfig
	metadata             metadata.MD
	newConnectionHandler func(cc *grpc.ClientConn)

	// these channels are created once
	disconnectedCh             chan bool
	backgroundConnectionDoneCh chan struct{}
	stopCh                     chan struct{}

	// this is for tests, so they can replace the closing
	// routine without a worry of modifying some global variable
	// or changing it back to original after the test is done
	closeBackgroundConnectionDoneCh func(ch chan struct{})
}

func newConnection(cfg otlpconfig.Config, sCfg otlpconfig.SignalConfig, handler func(cc *grpc.ClientConn)) *connection {
	c := new(connection)
	c.newConnectionHandler = handler
	c.cfg = cfg
	c.sCfg = sCfg
	if len(c.sCfg.Headers) > 0 {
		c.metadata = metadata.New(c.sCfg.Headers)
	}
	c.closeBackgroundConnectionDoneCh = func(ch chan struct{}) {
		close(ch)
	}
	return c
}

func (c *connection) startConnection(ctx context.Context) {
	c.stopCh = make(chan struct{})
	c.disconnectedCh = make(chan bool, 1)
	c.backgroundConnectionDoneCh = make(chan struct{})

	if err := c.connect(ctx); err == nil {
		c.setStateConnected()
	} else {
		c.setStateDisconnected(err)
	}
	go c.indefiniteBackgroundConnection()
}

func (c *connection) lastConnectError() error {
	errPtr := (*error)(atomic.LoadPointer(&c.lastConnectErrPtr))
	if errPtr == nil {
		return nil
	}
	return *errPtr
}

func (c *connection) saveLastConnectError(err error) {
	var errPtr *error
	if err != nil {
		errPtr = &err
	}
	atomic.StorePointer(&c.lastConnectErrPtr, unsafe.Pointer(errPtr))
}

func (c *connection) setStateDisconnected(err error) {
	c.saveLastConnectError(err)
	select {
	case c.disconnectedCh <- true:
	default:
	}
	c.newConnectionHandler(nil)
}

func (c *connection) setStateConnected() {
	c.saveLastConnectError(nil)
}

func (c *connection) connected() bool {
	return c.lastConnectError() == nil
}

const defaultConnReattemptPeriod = 10 * time.Second

func (c *connection) indefiniteBackgroundConnection() {
	defer func() {
		c.closeBackgroundConnectionDoneCh(c.backgroundConnectionDoneCh)
	}()

	connReattemptPeriod := c.cfg.ReconnectionPeriod
	if connReattemptPeriod <= 0 {
		connReattemptPeriod = defaultConnReattemptPeriod
	}

	// No strong seeding required, nano time can
	// already help with pseudo uniqueness.
	rng := rand.New(rand.NewSource(time.Now().UnixNano() + rand.Int63n(1024)))

	// maxJitterNanos: 70% of the connectionReattemptPeriod
	maxJitterNanos := int64(0.7 * float64(connReattemptPeriod))

	for {
		// Otherwise these will be the normal scenarios to enable
		// reconnection if we trip out.
		// 1. If we've stopped, return entirely
		// 2. Otherwise block until we are disconnected, and
		//    then retry connecting
		select {
		case <-c.stopCh:
			return

		case <-c.disconnectedCh:
			// Quickly check if we haven't stopped at the
			// same time.
			select {
			case <-c.stopCh:
				return

			default:
			}

			// Normal scenario that we'll wait for
		}

		if err := c.connect(context.Background()); err == nil {
			c.setStateConnected()
		} else {
			// this code is unreachable in most cases
			// c.connect does not establish connection
			c.setStateDisconnected(err)
		}

		// Apply some jitter to avoid lockstep retrials of other
		// collector-exporters. Lockstep retrials could result in an
		// innocent DDOS, by clogging the machine's resources and network.
		jitter := time.Duration(rng.Int63n(maxJitterNanos))
		select {
		case <-c.stopCh:
			return
		case <-time.After(connReattemptPeriod + jitter):
		}
	}
}

func (c *connection) connect(ctx context.Context) error {
	cc, err := c.dialToCollector(ctx)
	if err != nil {
		return err
	}
	c.setConnection(cc)
	c.newConnectionHandler(cc)
	return nil
}

// setConnection sets cc as the client connection and returns true if
// the connection state changed.
func (c *connection) setConnection(cc *grpc.ClientConn) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	// If previous clientConn is same as the current then just return.
	// This doesn't happen right now as this func is only called with new ClientConn.
	// It is more about future-proofing.
	if c.cc == cc {
		return false
	}

	// If the previous clientConn was non-nil, close it
	if c.cc != nil {
		_ = c.cc.Close()
	}
	c.cc = cc
	return true
}

func (c *connection) dialToCollector(ctx context.Context) (*grpc.ClientConn, error) {
	dialOpts := []grpc.DialOption{}
	if c.cfg.ServiceConfig != "" {
		dialOpts = append(dialOpts, grpc.WithDefaultServiceConfig(c.cfg.ServiceConfig))
	}
	if c.sCfg.GRPCCredentials != nil {
		dialOpts = append(dialOpts, grpc.WithTransportCredentials(c.sCfg.GRPCCredentials))
	} else if c.sCfg.Insecure {
		dialOpts = append(dialOpts, grpc.WithInsecure())
	}
	if c.sCfg.Compression == otlp.GzipCompression {
		dialOpts = append(dialOpts, grpc.WithDefaultCallOptions(grpc.UseCompressor(gzip.Name)))
	}
	if len(c.cfg.DialOptions) != 0 {
		dialOpts = append(dialOpts, c.cfg.DialOptions...)
	}

	ctx, cancel := c.contextWithStop(ctx)
	defer cancel()
	ctx = c.contextWithMetadata(ctx)
	return grpc.DialContext(ctx, c.sCfg.Endpoint, dialOpts...)
}

func (c *connection) contextWithMetadata(ctx context.Context) context.Context {
	if c.metadata.Len() > 0 {
		return metadata.NewOutgoingContext(ctx, c.metadata)
	}
	return ctx
}

func (c *connection) shutdown(ctx context.Context) error {
	close(c.stopCh)
	// Ensure that the backgroundConnector returns
	select {
	case <-c.backgroundConnectionDoneCh:
	case <-ctx.Done():
		return ctx.Err()
	}

	c.mu.Lock()
	cc := c.cc
	c.cc = nil
	c.mu.Unlock()

	if cc != nil {
		return cc.Close()
	}

	return nil
}

func (c *connection) contextWithStop(ctx context.Context) (context.Context, context.CancelFunc) {
	// Unify the parent context Done signal with the connection's
	// stop channel.
	ctx, cancel := context.WithCancel(ctx)
	go func(ctx context.Context, cancel context.CancelFunc) {
		select {
		case <-ctx.Done():
			// Nothing to do, either cancelled or deadline
			// happened.
		case <-c.stopCh:
			cancel()
		}
	}(ctx, cancel)
	return ctx, cancel
}

func (c *connection) doRequest(ctx context.Context, fn func(context.Context) error) error {
	expBackoff := newExponentialBackoff(c.cfg.RetrySettings)

	for {
		err := fn(ctx)
		if err == nil {
			// request succeeded.
			return nil
		}

		if !c.cfg.RetrySettings.Enabled {
			return err
		}

		// We have an error, check gRPC status code.
		st := status.Convert(err)
		if st.Code() == codes.OK {
			// Not really an error, still success.
			return nil
		}

		// Now, this is this a real error.

		if !shouldRetry(st.Code()) {
			// It is not a retryable error, we should not retry.
			return err
		}

		// Need to retry.
		var delay time.Duration

		// Respect server throttling.
		throttle := getThrottleDuration(st)

		backoffDelay := expBackoff.NextBackOff()
		if backoffDelay == backoff.Stop {
			// throw away the batch
			err = fmt.Errorf("max elapsed time expired %w", err)
			return err
		}

		if backoffDelay > throttle {
			delay = backoffDelay
		} else {
			delay = throttle
		}

		// back-off, but get interrupted when shutting down or request is cancelled or timed out.
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-c.stopCh:
			return fmt.Errorf("interrupted due to shutdown %w", err)
		case <-time.After(delay):
		}
	}
}

func shouldRetry(code codes.Code) bool {
	switch code {
	case codes.OK:
		// Success. This function should not be called for this code, the best we
		// can do is tell the caller not to retry.
		return false

	case codes.Canceled,
		codes.DeadlineExceeded,
		codes.PermissionDenied,
		codes.Unauthenticated,
		codes.ResourceExhausted,
		codes.Aborted,
		codes.OutOfRange,
		codes.Unavailable,
		codes.DataLoss:
		// These are retryable errors.
		return true

	case codes.Unknown,
		codes.InvalidArgument,
		codes.NotFound,
		codes.AlreadyExists,
		codes.FailedPrecondition,
		codes.Unimplemented,
		codes.Internal:
		// These are fatal errors, don't retry.
		return false

	default:
		// Don't retry on unknown codes.
		return false
	}
}

func getThrottleDuration(status *status.Status) time.Duration {
	// See if throttling information is available.
	for _, detail := range status.Details() {
		if t, ok := detail.(*errdetails.RetryInfo); ok {
			if t.RetryDelay.Seconds > 0 || t.RetryDelay.Nanos > 0 {
				// We are throttled. Wait before retrying as requested by the server.
				return time.Duration(t.RetryDelay.Seconds)*time.Second + time.Duration(t.RetryDelay.Nanos)*time.Nanosecond
			}
			return 0
		}
	}
	return 0
}

func newExponentialBackoff(rs otlp.RetrySettings) *backoff.ExponentialBackOff {
	// Do not use NewExponentialBackOff since it calls Reset and the code here must
	// call Reset after changing the InitialInterval (this saves an unnecessary call to Now).
	expBackoff := &backoff.ExponentialBackOff{
		InitialInterval:     rs.InitialInterval,
		RandomizationFactor: backoff.DefaultRandomizationFactor,
		Multiplier:          backoff.DefaultMultiplier,
		MaxInterval:         rs.MaxInterval,
		MaxElapsedTime:      rs.MaxElapsedTime,
		Stop:                backoff.Stop,
		Clock:               backoff.SystemClock,
	}
	expBackoff.Reset()

	return expBackoff
}
