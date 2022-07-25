package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/ory/graceful"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type arrayFlags []string

func (i *arrayFlags) String() string {
	return fmt.Sprint(*i)
}

func (i *arrayFlags) Set(value string) error {
	*i = append(*i, value)
	return nil
}

var urlTargets arrayFlags

var addr = flag.String("addr", ":8080", "The address to listen on for HTTP requests.")
var interval = flag.Int("interval", 15, "Fetch Interval in second")
var timeout = flag.Int("timeout", 120, "Fetch Timeout in second")
var location = flag.String("location", "default", "Location of exporter")

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

func healthHandler(w http.ResponseWriter, req *http.Request) {
	fmt.Fprintf(w, "OK")
}

func main() {
	flag.Var(&urlTargets, "target", "Url Target")
	flag.Parse()

	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}

	router := http.NewServeMux()
	srv := &http.Server{
		Addr:    *addr,
		Handler: router,
	}
	r := prometheus.NewRegistry()
	r.MustRegister(downloadTotalSize)
	r.MustRegister(downloadDuration)
	handler := promhttp.HandlerFor(r, promhttp.HandlerOpts{})
	router.Handle("/metrics", handler)
	router.HandleFunc("/health", healthHandler)

	quit := make(chan bool)
	for _, target := range urlTargets {
		job := NewJob(target, time.Duration(*interval)*time.Second)
		jobResult := make(chan JobResult)
		job.Loop(jobResult, quit)
		go func(job *Job, results chan JobResult) {
			for result := range results {
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
		}(job, jobResult)
	}

	log.Println("main: Starting the server")
	if err := graceful.Graceful(srv.ListenAndServe, srv.Shutdown); err != nil {
		close(quit)
		log.Fatalln("main: Failed to gracefully shutdown")
	}
	close(quit)
	log.Println("main: Server was shutdown gracefully")
}
