package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// TCPConnectionsCounter stores the total number of incoming tcp connections
	TCPConnectionsCounter = promauto.NewCounter(prometheus.CounterOpts{
		Name: "incoming_tcp_connections_total",
		Help: "The total number of incoming tcp connections",
	})

	// TCPConnectionErrorsCounter stores the total number of incoming tcp connection errors
	TCPConnectionErrorsCounter = promauto.NewCounter(prometheus.CounterOpts{
		Name: "incoming_tcp_connection_errors_total",
		Help: "The total number of incoming tcp connection errors",
	})

	// NumTargets stores the number of targets
	NumTargets = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "num_targets",
		Help: "The current number of healthy targets",
	})

	// NumHealthyTargets stores the number of healthy targets
	NumHealthyTargets = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "num_healthy_targets",
		Help: "The number of healthy targets",
	})
)
