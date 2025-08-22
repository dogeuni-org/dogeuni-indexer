package metrics

import (
	"sync/atomic"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	CardityDeployTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{Name: "cardity_deploy_total", Help: "Cardity deploy ops count"},
		[]string{"op"},
	)
	CardityInvokeTotal = prometheus.NewCounter(
		prometheus.CounterOpts{Name: "cardity_invoke_total", Help: "Cardity invoke count"},
	)
	CardityErrorsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{Name: "cardity_errors_total", Help: "Cardity errors count"},
		[]string{"stage"},
	)
	CardityAssembleDuration = prometheus.NewHistogram(
		prometheus.HistogramOpts{Name: "cardity_assemble_duration_seconds", Help: "Assemble duration", Buckets: []float64{0.01, 0.05, 0.1, 0.25, 0.5, 1, 2, 5}},
	)
	CardityDecodeDuration = prometheus.NewHistogram(
		prometheus.HistogramOpts{Name: "cardity_decode_duration_seconds", Help: "Decode duration", Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5}},
	)
	CardityBundlesIncomplete = prometheus.NewGauge(
		prometheus.GaugeOpts{Name: "cardity_bundles_incomplete", Help: "Incomplete bundles"},
	)
	CardityLastBlock = prometheus.NewGauge(
		prometheus.GaugeOpts{Name: "cardity_last_block", Help: "Last processed block height"},
	)

	lastBlock int64
)

func MustRegister() {
	prometheus.MustRegister(
		CardityDeployTotal,
		CardityInvokeTotal,
		CardityErrorsTotal,
		CardityAssembleDuration,
		CardityDecodeDuration,
		CardityBundlesIncomplete,
		CardityLastBlock,
	)
}

func IncDeploy(op string) { CardityDeployTotal.WithLabelValues(op).Inc() }
func IncInvoke()            { CardityInvokeTotal.Inc() }
func IncError(stage string) { CardityErrorsTotal.WithLabelValues(stage).Inc() }

func ObserveAssemble(seconds float64) { CardityAssembleDuration.Observe(seconds) }
func ObserveDecode(seconds float64)   { CardityDecodeDuration.Observe(seconds) }

func SetBundlesIncomplete(n int) { CardityBundlesIncomplete.Set(float64(n)) }
func SetLastBlock(height int64) {
	atomic.StoreInt64(&lastBlock, height)
	CardityLastBlock.Set(float64(height))
}
