package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	namespace = "diskusage"
)

// Exporter collects metrics
type Exporter struct {
	paths []string

	usageBytes *prometheus.Desc
}

// NewExporter returns an initialized exporter.
func NewExporter(paths []string) *Exporter {
	return &Exporter{
		paths: paths,
		usageBytes: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "usage_bytes"),
			"Disk usage in bytes",
			[]string{"path"},
			nil,
		),
	}
}

// Describe describes all the metrics. It implements prometheus.Collector.
func (e *Exporter) Describe(ch chan<- *prometheus.Desc) {
	ch <- e.usageBytes
}

// Collect delivers them as Prometheus metrics. It implements prometheus.Collector.
func (e *Exporter) Collect(ch chan<- prometheus.Metric) {
	for _, path := range e.paths {
		out, err := exec.Command("du", "-s", path).Output()
		if err != nil {
			log.Printf("Failed to get disk usage of %s: %s", path, err)
			continue
		}

		splited := strings.SplitN(string(out), "\t", 2)
		usageBytes, err := strconv.ParseFloat(splited[0], 64)
		if err != nil {
			log.Printf("Failed to get disk usage of %s: %s", path, err)
			continue
		}

		ch <- prometheus.MustNewConstMetric(e.usageBytes, prometheus.GaugeValue, usageBytes, path)
	}
}

func main() {
	listenAddress := flag.String("web.listen-address", ":9550", "Address to listen on")
	metricsPath := flag.String("web.telemetry-path", "/metrics", "Path under which to expose metrics")
	flag.Parse()

	//http.Handle(*metricsPath, promhttp.Handler())
	http.HandleFunc(*metricsPath, func(writer http.ResponseWriter, request *http.Request) {
		paths := request.URL.Query()["paths[]"]

		r := prometheus.NewRegistry()
		r.MustRegister(prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{}))
		r.MustRegister(prometheus.NewGoCollector())
		r.MustRegister(NewExporter(paths))
		promhttp.HandlerFor(r, promhttp.HandlerOpts{}).ServeHTTP(writer, request)
	})

	log.Printf("Listening on %s", *listenAddress)
	if err := http.ListenAndServe(*listenAddress, nil); err != nil {
		os.Exit(1)
	}
}
