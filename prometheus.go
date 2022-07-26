package main

import (
	"fmt"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	downloadError = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "url_download_error_total",
			Help: "Download error total",
		},
		[]string{"url", "location"})
	downloadTotalSize = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "url_download_size_bytes_total",
			Help: "Download size total",
		},
		[]string{"url", "code", "location"})
	downloadDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "url_download_duration_seconds",
			Help:    "Download Duration",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"url", "code", "location"})
)

func initPrometheusHandler() http.Handler {
	r := prometheus.NewRegistry()
	r.MustRegister(downloadTotalSize)
	r.MustRegister(downloadDuration)
	r.MustRegister(downloadError)
	return promhttp.HandlerFor(r, promhttp.HandlerOpts{})
}

func jobResultMetrics(job *Job) {
	for result := range job.Result {
		if result.Err == nil {
			label := prometheus.Labels{
				"url":      job.Url,
				"location": *location,
				"code":     fmt.Sprintf("%d", result.Code)}
			downloadTotalSize.With(label).Add(float64(result.Size))
			downloadDuration.With(label).Observe(result.Duration.Seconds())
		} else {
			downloadError.With(prometheus.Labels{
				"url":      job.Url,
				"location": *location}).Inc()
		}
	}
}
