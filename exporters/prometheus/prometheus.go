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

package prometheus // import "go.opentelemetry.io/otel/exporters/prometheus"

// Note that this package does not support a way to export Prometheus
// Summary data points, removed in PR#1412.

import (
	"context"
	"fmt"
	"net/http"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/aggregator/aggregation"
	"go.opentelemetry.io/otel/sdk/metric/number"
	"go.opentelemetry.io/otel/sdk/metric/sdkinstrument"
	"go.opentelemetry.io/otel/sdk/resource"
)

// Exporter supports Prometheus pulls.  It does not implement the
// sdk/export/metric.Exporter interface--instead it creates a pull
// controller and reads the latest checkpointed data on-scrape.
type Exporter struct {
	handler    http.Handler
	registerer prometheus.Registerer
	gatherer   prometheus.Gatherer

	lock     sync.Mutex
	producer metric.Producer
}

var _ metric.Reader = &Exporter{}
var _ http.Handler = &Exporter{}

// ErrUnsupportedAggregator is returned for unrepresentable aggregator
// types.
var ErrUnsupportedAggregator = fmt.Errorf("unsupported aggregator type")

// Config is a set of configs for the tally reporter.
type Config struct {
	// Registry is the prometheus registry that will be used as the default Registerer and
	// Gatherer if these are not specified.
	//
	// If not set a new empty Registry is created.
	Registry *prometheus.Registry

	// Registerer is the prometheus registerer to register
	// metrics with.
	//
	// If not specified the Registry will be used as default.
	Registerer prometheus.Registerer

	// Gatherer is the prometheus gatherer to gather
	// metrics with.
	//
	// If not specified the Registry will be used as default.
	Gatherer prometheus.Gatherer
}

// New returns a new Prometheus exporter using the configured metric
// controller.  See controller.New().
func New(config Config) (*Exporter, error) {
	if config.Registry == nil {
		config.Registry = prometheus.NewRegistry()
	}

	if config.Registerer == nil {
		config.Registerer = config.Registry
	}

	if config.Gatherer == nil {
		config.Gatherer = config.Registry
	}

	e := &Exporter{
		handler:    promhttp.HandlerFor(config.Gatherer, promhttp.HandlerOpts{}),
		registerer: config.Registerer,
		gatherer:   config.Gatherer,
	}

	c := &collector{
		exp: e,
	}
	if err := config.Registerer.Register(c); err != nil {
		return nil, fmt.Errorf("cannot register the collector: %w", err)
	}
	return e, nil
}

func (e *Exporter) String() string {
	return "prometheus"
}

func (e *Exporter) Register(p metric.Producer) {
	e.lock.Lock()
	defer e.lock.Unlock()
	e.producer = p
}

func (e *Exporter) getProducer() metric.Producer {
	e.lock.Lock()
	defer e.lock.Unlock()
	return e.producer
}

func (e *Exporter) ForceFlush(_ context.Context) error {
	return nil
}

func (e *Exporter) Shutdown(_ context.Context) error {
	return nil
}

// ServeHTTP implements http.Handler.
func (e *Exporter) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	e.handler.ServeHTTP(w, r)
}

// collector implements prometheus.Collector interface.
type collector struct {
	exp *Exporter
}

var _ prometheus.Collector = (*collector)(nil)

// Describe implements prometheus.Collector.
func (c *collector) Describe(ch chan<- *prometheus.Desc) {
	// Passing nil => not reusing memory
	producer := c.exp.getProducer()
	if producer == nil {
		return
	}
	data := producer.Produce(nil)

	for _, scope := range data.Scopes {
		for _, inst := range scope.Instruments {
			for _, point := range inst.Points {
				var labelKeys []string
				mergeLabels(point.Attributes, data.Resource, &labelKeys, nil)
				ch <- c.toDesc(inst.Descriptor, labelKeys)
			}
		}
	}
}

// Collect exports the last calculated Reader state.
//
// Collect is invoked whenever prometheus.Gatherer is also invoked.
// For example, when the HTTP endpoint is invoked by Prometheus.
func (c *collector) Collect(ch chan<- prometheus.Metric) {
	// Passing nil => not reusing memory
	data := c.exp.producer.Produce(nil)

	for _, scope := range data.Scopes {
		for _, inst := range scope.Instruments {
			numberKind := inst.Descriptor.NumberKind

			for _, point := range inst.Points {

				agg := point.Aggregation

				var labelKeys, labels []string
				mergeLabels(point.Attributes, data.Resource, &labelKeys, &labels)

				desc := c.toDesc(inst.Descriptor, labelKeys)

				switch agg.Kind() {
				case aggregation.HistogramKind:
					if err := c.exportHistogram(ch, agg.(aggregation.Histogram), numberKind, desc, labels); err != nil {
						otel.Handle(fmt.Errorf("exporting histogram: %w", err))
					}
				case aggregation.MonotonicSumKind:
					if err := c.exportMonotonicCounter(ch, agg.(aggregation.Sum), numberKind, desc, labels); err != nil {
						otel.Handle(fmt.Errorf("exporting monotonic counter: %w", err))
					}
				case aggregation.NonMonotonicSumKind:
					if err := c.exportGauge(ch, agg.(aggregation.Sum).Sum(), numberKind, desc, labels); err != nil {
						otel.Handle(fmt.Errorf("exporting gauge: %w", err))
					}
				case aggregation.GaugeKind:
					if err := c.exportGauge(ch, agg.(aggregation.Gauge).Gauge(), numberKind, desc, labels); err != nil {
						otel.Handle(fmt.Errorf("exporting gauge: %w", err))
					}
				default:
					otel.Handle(fmt.Errorf("%w: %s", ErrUnsupportedAggregator, agg.Kind()))
				}
			}
		}
	}
}

func (c *collector) exportGauge(ch chan<- prometheus.Metric, value number.Number, kind number.Kind, desc *prometheus.Desc, labels []string) error {
	m, err := prometheus.NewConstMetric(desc, prometheus.GaugeValue, value.CoerceToFloat64(kind), labels...)
	if err != nil {
		return fmt.Errorf("error creating constant metric: %w", err)
	}

	ch <- m
	return nil
}

func (c *collector) exportMonotonicCounter(ch chan<- prometheus.Metric, sum aggregation.Sum, kind number.Kind, desc *prometheus.Desc, labels []string) error {
	v := sum.Sum()

	m, err := prometheus.NewConstMetric(desc, prometheus.CounterValue, v.CoerceToFloat64(kind), labels...)
	if err != nil {
		return fmt.Errorf("error creating constant metric: %w", err)
	}

	ch <- m
	return nil
}

func (c *collector) exportHistogram(ch chan<- prometheus.Metric, hist aggregation.Histogram, kind number.Kind, desc *prometheus.Desc, labels []string) error {
	buckets := hist.Buckets()
	sum := hist.Sum()

	var totalCount uint64
	// counts maps from the bucket upper-bound to the cumulative count.
	// The bucket with upper-bound +inf is not included.
	counts := make(map[float64]uint64, len(buckets.Boundaries))
	for i := range buckets.Boundaries {
		boundary := buckets.Boundaries[i]
		totalCount += uint64(buckets.Counts[i])
		counts[boundary] = totalCount
	}
	// Include the +inf bucket in the total count.
	totalCount += uint64(buckets.Counts[len(buckets.Counts)-1])

	m, err := prometheus.NewConstHistogram(desc, totalCount, sum.CoerceToFloat64(kind), counts, labels...)
	if err != nil {
		return fmt.Errorf("error creating constant histogram: %w", err)
	}

	ch <- m
	return nil
}

func (c *collector) toDesc(inst sdkinstrument.Descriptor, labelKeys []string) *prometheus.Desc {
	return prometheus.NewDesc(sanitize(inst.Name), inst.Description, labelKeys, nil)
}

// mergeLabels merges the export.Record's labels and resources into a
// single set, giving precedence to the record's labels in case of
// duplicate keys.  This outputs one or both of the keys and the
// values as a slice, and either argument may be nil to avoid
// allocating an unnecessary slice.
func mergeLabels(attrs attribute.Set, res *resource.Resource, keys, values *[]string) {
	if keys != nil {
		*keys = make([]string, 0, attrs.Len()+res.Len())
	}
	if values != nil {
		*values = make([]string, 0, attrs.Len()+res.Len())
	}

	// Duplicate keys are resolved by taking the record label value over
	// the resource value.
	mi := attribute.NewMergeIterator(&attrs, res.Set())
	for mi.Next() {
		label := mi.Label()
		if keys != nil {
			*keys = append(*keys, sanitize(string(label.Key)))
		}
		if values != nil {
			*values = append(*values, label.Value.Emit())
		}
	}
}
