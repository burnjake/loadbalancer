package otel

import (
	"context"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"google.golang.org/grpc/credentials/insecure"
)

type Metrics struct {
	TCPConnectionsCounter      metric.Int64Counter
	TCPConnectionErrorsCounter metric.Int64Counter
	NumTargets                 metric.Int64ObservableGauge
	NumHealthyTargets          metric.Int64ObservableGauge
}

func InitMetrics(ctx context.Context, endpoint string) (Metrics, func(context.Context) error, error) {
	var metrics Metrics

	exporter, err := otlpmetricgrpc.New(
		ctx,
		otlpmetricgrpc.WithEndpoint(endpoint),
		otlpmetricgrpc.WithTLSCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return metrics, nil, err
	}

	meterProvider := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(
			sdkmetric.NewPeriodicReader(exporter, sdkmetric.WithInterval(time.Second*5)),
		),
	)
	if err != nil {
		return metrics, nil, err
	}

	otel.SetMeterProvider(meterProvider)
	mtr := otel.Meter("loadbalancer")
	metrics.TCPConnectionsCounter, _ = mtr.Int64Counter("tcp_connections_counter")
	metrics.TCPConnectionErrorsCounter, _ = mtr.Int64Counter("tcp_connection_errors_counter")
	metrics.NumTargets, _ = mtr.Int64ObservableGauge("num_targets")
	metrics.NumHealthyTargets, _ = mtr.Int64ObservableGauge("num_healthy_targets")
	return metrics, meterProvider.Shutdown, nil
}
