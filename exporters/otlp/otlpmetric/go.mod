module go.opentelemetry.io/otel/exporters/otlp/otlpmetric

go 1.18

require (
	github.com/stretchr/testify v1.7.1
	go.opentelemetry.io/otel v1.9.0
	go.opentelemetry.io/otel/exporters/otlp/internal/retry v1.9.0
	go.opentelemetry.io/otel/metric v0.0.0-00010101000000-000000000000
	go.opentelemetry.io/otel/sdk v0.0.0-00010101000000-000000000000
	go.opentelemetry.io/otel/sdk/metric v0.0.0-00010101000000-000000000000
	go.opentelemetry.io/proto/otlp v0.18.0
	google.golang.org/grpc v1.42.0
)

require (
	github.com/cenkalti/backoff/v4 v4.1.3 // indirect
	github.com/davecgh/go-spew v1.1.0 // indirect
	github.com/go-logr/logr v1.2.3 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/golang/protobuf v1.5.2 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	go.opentelemetry.io/otel/trace v1.9.0 // indirect
	golang.org/x/net v0.0.0-20210405180319-a5a99cb37ef4 // indirect
	golang.org/x/sys v0.0.0-20210510120138-977fb7262007 // indirect
	golang.org/x/text v0.3.5 // indirect
	google.golang.org/genproto v0.0.0-20211118181313-81c1377c94b1 // indirect
	google.golang.org/protobuf v1.27.1 // indirect
	gopkg.in/yaml.v3 v3.0.0-20200313102051-9f266ea9e77c // indirect
)

replace go.opentelemetry.io/otel/metric => ../../../metric

replace go.opentelemetry.io/otel => ../../..

replace go.opentelemetry.io/otel/sdk/metric => ../../../sdk/metric

replace go.opentelemetry.io/otel/trace => ../../../trace

replace go.opentelemetry.io/otel/sdk => ../../../sdk

replace go.opentelemetry.io/otel/exporters/otlp/internal/retry => ../internal/retry
