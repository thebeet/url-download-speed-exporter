package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/ory/graceful"
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
var timeout = flag.Int("timeout", 60, "Fetch Timeout in second")
var location = flag.String("location", "default", "Location of exporter")
var insecureskipverify = flag.Bool("insecureskipverify", false, "Skip InSecure verify")

func main() {
	flag.Var(&urlTargets, "target", "Url Target")
	flag.Parse()
	if *insecureskipverify {
		http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}

	router := http.NewServeMux()
	srv := &http.Server{
		Addr:    *addr,
		Handler: router,
	}

	router.Handle("/metrics", initPrometheusHandler())
	router.HandleFunc("/health", healthHandler)

	var jobs []*Job
	for _, target := range urlTargets {
		job := NewJob(target, time.Duration(*interval)*time.Second)
		jobs = append(jobs, job)
		job.Loop()
		go jobResultMetrics(job)
	}

	log.Println("main: Starting the server")
	if err := graceful.Graceful(srv.ListenAndServe, srv.Shutdown); err != nil {
		log.Fatalln("main: Failed to gracefully shutdown")
	}
	for _, job := range jobs {
		close(job.Quit)
	}
	for _, job := range jobs {
		<-job.Done
	}
	log.Println("main: Server was shutdown gracefully")
}

func healthHandler(w http.ResponseWriter, req *http.Request) {
	fmt.Fprintf(w, "OK")
}
