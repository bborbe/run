package run

import (
	"context"

	"github.com/golang/glog"
	"github.com/prometheus/client_golang/prometheus"
)

// NewMetrics create prometheus metrics for the given RunFunc.
func NewMetrics(
	namespace string,
	subsystem string,
	fn RunFunc,
) func(ctx context.Context) error {
	started := prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: namespace,
		Subsystem: subsystem,
		Name:      "started",
		Help:      "started",
	})
	completed := prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: namespace,
		Subsystem: subsystem,
		Name:      "completed",
		Help:      "completed",
	})
	failed := prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: namespace,
		Subsystem: subsystem,
		Name:      "failed",
		Help:      "failed",
	})
	prometheus.MustRegister(started, completed, failed)
	return func(ctx context.Context) error {
		started.Inc()
		if err := fn(ctx); err != nil {
			failed.Inc()
			return err
		}
		completed.Inc()
		return nil
	}
}

// SkipErrors runs the given RunFunc and returns always nil.
func SkipErrors(fn RunFunc) func(ctx context.Context) error {
	return func(ctx context.Context) error {
		if err := fn(ctx); err != nil {
			glog.Warningf("run failed: %v", err)
		}
		return nil
	}
}
