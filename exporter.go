package main

import (
	"flag"
	"fmt"
	"net"
	"net/http"
	"sync"

	"github.com/golang/glog"
	"github.com/prometheus/client_golang/prometheus"
)

const (
	namespace = "health"
)

type HealthCheckResponse struct {
	Status string        `json:"status"`
	IsOk   bool
	Labels prometheus.Labels
}

type OverallHealthCheckResult map[string]HealthCheckResponse

type Exporter struct {
	config          *Config
	client          *http.Client
	mutex           sync.RWMutex

	up              prometheus.Gauge
	statusByService *prometheus.GaugeVec
}

func NewExporter(servicesConfig *Config) *Exporter {
	httpClient := &http.Client{}

	return &Exporter{
		client: httpClient,
		config : servicesConfig,

		up: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "overall",
				Help:      "overall service availability",
			}),
		statusByService: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "service",
				Help:      "service status by service",
			},
			servicesConfig.collectLabels()),
	}
}

func (e *Exporter) Describe(ch chan <- *prometheus.Desc) {
	ch <- e.up.Desc()
	e.statusByService.Describe(ch)
}

func (e *Exporter) Collect(ch chan <- prometheus.Metric) {
	e.mutex.Lock() // To protect metrics from concurrent collects.
	defer e.mutex.Unlock()

	sendStatus := func() {
		ch <- e.up
		e.statusByService.Collect(ch)
	}
	defer sendStatus()

	e.up.Set(0)

	result, err := e.performAllChecks()
	if err != nil {
		glog.Error(fmt.Sprintf("Error collecting stats: %s", err))
	}

	overall := 1.0
	for k, v := range result {
		glog.Infof("%s -> %v", k, v)
		serviceUp := 0.0
		if v.IsOk {
			serviceUp = 1.0
		} else {
			overall = 0.0
		}
		e.statusByService.With(v.Labels).Set(serviceUp)
	}
	e.up.Set(overall)
	return
}

func (e *Exporter) performCheck(service Service) (*HealthCheckResponse, error) {
	resp, err := e.client.Get(service.Uri)
	if err != nil {
		glog.Errorf("Error reading from URI %s: %v", service.Uri, err)
		status := &HealthCheckResponse{Status:"ERROR", IsOk:false}
		labels := prometheus.Labels{"service_name": service.Name}
		for label, value := range service.Labels {
			labels[label] = value
		}
		status.Labels = labels
		return status, err
	}

	status := &HealthCheckResponse{Status:"DOWN", IsOk:false}
	labels := prometheus.Labels{"service_name": service.Name}
	for label, value := range service.Labels {
		labels[label] = value
	}
	glog.Infof(fmt.Sprintf("labels: %v\n", labels))
	if resp.StatusCode >= 200 && resp.StatusCode < 400 {
		status.Status = "UP"
		status.IsOk = true
		status.Labels = labels
	}
	glog.Infof(fmt.Sprintf("status for %s: %v\n", service.Name, status))
	return status, nil
}

func (e *Exporter) performAllChecks() (OverallHealthCheckResult, error) {
	result := make(OverallHealthCheckResult)
	for _, service := range e.config.Services {
		status, _ := e.performCheck(service)
		result[service.Name] = *status
	}
	return result, nil
}

func main() {
	flag.Parse()

	config, err := getConfig(*configFile)
	if err != nil {
		glog.Fatal(err)
	}
	glog.Infof("using config from %s: %v", *configFile, config)

	exporter := NewExporter(config)
	prometheus.MustRegister(exporter)

	http.Handle(*metricsEndpoint, prometheus.Handler())
	http.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "OK")
	})
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, *metricsEndpoint, http.StatusMovedPermanently)
	})

	glog.Infof("Starting exporter at %s", *listenAddress)
	listener, err := net.Listen("tcp4", *listenAddress)
	if err != nil {
		glog.Fatal(err)
	}
	err = http.Serve(listener, nil)
	if err != nil {
		glog.Fatal(err)
	}
}
