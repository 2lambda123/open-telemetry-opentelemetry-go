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

package otlphttp_test

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"sync"
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	collectormetricpb "go.opentelemetry.io/otel/exporters/otlp/internal/opentelemetry-proto-gen/collector/metrics/v1"
	collectortracepb "go.opentelemetry.io/otel/exporters/otlp/internal/opentelemetry-proto-gen/collector/trace/v1"
	metricpb "go.opentelemetry.io/otel/exporters/otlp/internal/opentelemetry-proto-gen/metrics/v1"
	tracepb "go.opentelemetry.io/otel/exporters/otlp/internal/opentelemetry-proto-gen/trace/v1"
	"go.opentelemetry.io/otel/exporters/otlp/internal/otlptest"
	"go.opentelemetry.io/otel/exporters/otlp/internal/transform/transformjson"
	"go.opentelemetry.io/otel/exporters/otlp/otlphttp"
)

type mockCollector struct {
	endpoint string
	server   *http.Server

	spanLock     sync.Mutex
	spansStorage otlptest.SpansStorage

	metricLock     sync.Mutex
	metricsStorage otlptest.MetricsStorage

	injectHTTPStatus  []int
	injectContentType string

	clientTLSConfig *tls.Config
	expectedHeaders map[string]string
}

func (c *mockCollector) Stop() error {
	return c.server.Shutdown(context.Background())
}

func (c *mockCollector) MustStop(t *testing.T) {
	assert.NoError(t, c.server.Shutdown(context.Background()))
}

func (c *mockCollector) GetSpans() []*tracepb.Span {
	c.spanLock.Lock()
	defer c.spanLock.Unlock()
	return c.spansStorage.GetSpans()
}

func (c *mockCollector) GetResourceSpans() []*tracepb.ResourceSpans {
	c.spanLock.Lock()
	defer c.spanLock.Unlock()
	return c.spansStorage.GetResourceSpans()
}

func (c *mockCollector) GetMetrics() []*metricpb.Metric {
	c.metricLock.Lock()
	defer c.metricLock.Unlock()
	return c.metricsStorage.GetMetrics()
}

func (c *mockCollector) Endpoint() string {
	return c.endpoint
}

func (c *mockCollector) ClientTLSConfig() *tls.Config {
	return c.clientTLSConfig
}

func (c *mockCollector) serveMetrics(w http.ResponseWriter, r *http.Request) {
	if !c.checkHeaders(r) {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	contentType, err := getValidContentType(r)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	rawResponse, err := marshalMessage(contentType, &collectormetricpb.ExportMetricsServiceResponse{})
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if injectedStatus := c.getInjectHTTPStatus(); injectedStatus != 0 {
		writeReply(w, contentType, rawResponse, injectedStatus, c.injectContentType)
		return
	}
	rawRequest, err := readRequest(r)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	request := collectormetricpb.ExportMetricsServiceRequest{}
	if err := unmarshalMessage(contentType, &request, rawRequest); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	writeReply(w, contentType, rawResponse, 0, c.injectContentType)
	c.metricLock.Lock()
	defer c.metricLock.Unlock()
	c.metricsStorage.AddMetrics(&request)
}

func (c *mockCollector) serveTraces(w http.ResponseWriter, r *http.Request) {
	if !c.checkHeaders(r) {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	contentType, err := getValidContentType(r)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	rawResponse, err := marshalMessage(contentType, &collectortracepb.ExportTraceServiceResponse{})
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if injectedStatus := c.getInjectHTTPStatus(); injectedStatus != 0 {
		writeReply(w, contentType, rawResponse, injectedStatus, c.injectContentType)
		return
	}
	rawRequest, err := readRequest(r)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	request := collectortracepb.ExportTraceServiceRequest{}
	if err := unmarshalMessage(contentType, &request, rawRequest); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	writeReply(w, contentType, rawResponse, 0, c.injectContentType)
	c.spanLock.Lock()
	defer c.spanLock.Unlock()
	c.spansStorage.AddSpans(&request)
}

func (c *mockCollector) checkHeaders(r *http.Request) bool {
	for k, v := range c.expectedHeaders {
		got := r.Header.Get(k)
		if got != v {
			return false
		}
	}
	return true
}

func getValidContentType(r *http.Request) (string, error) {
	contentType := r.Header.Get("Content-Type")
	switch contentType {
	case "application/x-protobuf":
	case "application/json":
	default:
		return "", fmt.Errorf("invalid content type %q", contentType)
	}
	return contentType, nil
}

type protoMarshalerMessage interface {
	proto.Marshaler
	proto.Message
}

func marshalMessage(contentType string, message protoMarshalerMessage) ([]byte, error) {
	switch contentType {
	case "application/x-protobuf":
		return message.Marshal()
	case "application/json":
		return transformjson.Marshal(message)
	default:
		return nil, fmt.Errorf("should not happen, %s should be a valid content type by now", contentType)
	}
}

type protoUnmarshalerMessage interface {
	proto.Unmarshaler
	proto.Message
}

func unmarshalMessage(contentType string, message protoUnmarshalerMessage, data []byte) error {
	switch contentType {
	case "application/x-protobuf":
		return message.Unmarshal(data)
	case "application/json":
		return transformjson.Unmarshal(data, message)
	default:
		return fmt.Errorf("should not happen, %s should be a valid content type by now", contentType)
	}
}

func (c *mockCollector) getInjectHTTPStatus() int {
	if len(c.injectHTTPStatus) == 0 {
		return 0
	}
	status := c.injectHTTPStatus[0]
	c.injectHTTPStatus = c.injectHTTPStatus[1:]
	if len(c.injectHTTPStatus) == 0 {
		c.injectHTTPStatus = nil
	}
	return status
}

func readRequest(r *http.Request) ([]byte, error) {
	if r.Header.Get("Content-Encoding") == "gzip" {
		return readGzipBody(r.Body)
	}
	return ioutil.ReadAll(r.Body)
}

func readGzipBody(body io.Reader) ([]byte, error) {
	rawRequest := bytes.Buffer{}
	gunzipper, err := gzip.NewReader(body)
	if err != nil {
		return nil, err
	}
	defer gunzipper.Close()
	_, err = io.Copy(&rawRequest, gunzipper)
	if err != nil {
		return nil, err
	}
	return rawRequest.Bytes(), nil
}

func writeReply(w http.ResponseWriter, contentType string, rawResponse []byte, injectHTTPStatus int, injectContentType string) {
	status := http.StatusOK
	if injectHTTPStatus != 0 {
		status = injectHTTPStatus
	}
	if injectContentType != "" {
		contentType = injectContentType
	}
	w.Header().Set("Content-Type", contentType)
	w.WriteHeader(status)
	_, _ = w.Write(rawResponse)
}

type mockCollectorConfig struct {
	MetricsURLPath    string
	TracesURLPath     string
	Port              int
	InjectHTTPStatus  []int
	InjectContentType string
	WithTLS           bool
	ExpectedHeaders   map[string]string
}

func (c *mockCollectorConfig) fillInDefaults() {
	if c.MetricsURLPath == "" {
		c.MetricsURLPath = otlphttp.DefaultMetricsPath
	}
	if c.TracesURLPath == "" {
		c.TracesURLPath = otlphttp.DefaultTracesPath
	}
}

func runMockCollector(t *testing.T, cfg mockCollectorConfig) *mockCollector {
	cfg.fillInDefaults()
	ln, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", cfg.Port))
	require.NoError(t, err)
	_, portStr, err := net.SplitHostPort(ln.Addr().String())
	require.NoError(t, err)
	m := &mockCollector{
		endpoint:          fmt.Sprintf("localhost:%s", portStr),
		spansStorage:      otlptest.NewSpansStorage(),
		metricsStorage:    otlptest.NewMetricsStorage(),
		injectHTTPStatus:  cfg.InjectHTTPStatus,
		injectContentType: cfg.InjectContentType,
		expectedHeaders:   cfg.ExpectedHeaders,
	}
	mux := http.NewServeMux()
	mux.Handle(cfg.MetricsURLPath, http.HandlerFunc(m.serveMetrics))
	mux.Handle(cfg.TracesURLPath, http.HandlerFunc(m.serveTraces))
	server := &http.Server{
		Handler: mux,
	}
	if cfg.WithTLS {
		pem, err := generateWeakCertificate()
		require.NoError(t, err)
		tlsCertificate, err := tls.X509KeyPair(pem.Certificate, pem.PrivateKey)
		require.NoError(t, err)
		server.TLSConfig = &tls.Config{
			Certificates: []tls.Certificate{tlsCertificate},
		}

		m.clientTLSConfig = &tls.Config{
			InsecureSkipVerify: true,
		}
	}
	go func() {
		if cfg.WithTLS {
			_ = server.ServeTLS(ln, "", "")
		} else {
			_ = server.Serve(ln)
		}
	}()
	m.server = server
	return m
}
