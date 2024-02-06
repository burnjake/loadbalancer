module github.com/burnjake/loadbalancer

go 1.15

require (
	github.com/prometheus/client_golang v1.18.0
	gopkg.in/yaml.v2 v2.4.0
)

require (
	go.opentelemetry.io/otel v1.22.0
	go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc v0.45.0
	go.opentelemetry.io/otel/metric v1.22.0
	go.opentelemetry.io/otel/sdk/metric v1.22.0
	google.golang.org/grpc v1.61.0
)
