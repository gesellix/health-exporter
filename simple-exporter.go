package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"sync"

	"github.com/golang/glog"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	configFile = flag.String("config.file", "", "Path to config file.")
	listenAddress = flag.String("telemetry.address", ":9990", "Address on which to expose metrics.")
	metricsEndpoint = flag.String("telemetry.endpoint", "/metrics", "Path under which to expose metrics.")
)

const (
	namespace = "health"
)

type HealthCheckResponse struct {
	Status string        `json:"status"`
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
			[]string{"service_name"}),
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
	glog.Infof("result: %v", result)
	for k, v := range result {
		glog.Infof("result: %s->%v", k, v)
		serviceUp := 0.0
		if v.Status == "UP" {
			serviceUp = 1.0
		} else {
			overall = 0.0
		}
		e.statusByService.WithLabelValues(k).Set(serviceUp)
	}
	e.up.Set(overall)
	return
}

func (e *Exporter) performCheck(client *http.Client, service Service) (*HealthCheckResponse, error) {
	resp, err := client.Get(service.Uri)
	if err != nil {
		return nil, fmt.Errorf("Error reading from URI %s: %v", service.Uri, err)
	}

	data, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 400 {
		if err != nil {
			data = []byte(err.Error())
		}
		return nil, fmt.Errorf("Status %s (%d): %s", resp.Status, resp.StatusCode, data)
	}

	status := &HealthCheckResponse{}
	err = json.Unmarshal(data, &status)
	glog.Infof(fmt.Sprintf("status for %s: %v\n", service.Name, status))
	return status, nil
}

func (e *Exporter) performAllChecks() (OverallHealthCheckResult, error) {
	result := make(OverallHealthCheckResult)
	for _, service := range e.config.Services {
		status, err := e.performCheck(e.client, service)
		if err != nil {
			glog.Fatal(err)
			return nil, fmt.Errorf("Error reading from URI %s: %v", service.Uri, err)
		}
		result[service.Name] = *status
	}
	return result, nil
}

func getConfig(file string) (*Config, error) {
	config := &Config{}
	glog.Infof("reading config from %s", file)
	bytes, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, err
	}
	return config, json.Unmarshal(bytes, &config)
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
	err = http.ListenAndServe(*listenAddress, nil)
	if err != nil {
		glog.Fatal(err)
	}
}
