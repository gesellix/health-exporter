package main

import (
	"fmt"
	"net/http"
	"sync"

	"github.com/golang/glog"
	"github.com/prometheus/client_golang/prometheus"
)

const (
	namespace = "health"
)

type HealthCheckResult struct {
	Status string
	IsOk   bool
	Labels prometheus.Labels
}

type OverallHealthCheckResult map[string]HealthCheckResult

type Exporter struct {
	config        *Config
	client        *http.Client
	mutex         sync.RWMutex

	up            prometheus.Gauge
	serviceStatus *prometheus.GaugeVec
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
		serviceStatus: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "service",
				Help:      "service status summary",
			},
			servicesConfig.collectUniqueLabelNames()),
	}
}

func (e *Exporter) Describe(ch chan <- *prometheus.Desc) {
	ch <- e.up.Desc()
	e.serviceStatus.Describe(ch)
}

func (e *Exporter) Collect(ch chan <- prometheus.Metric) {
	e.mutex.Lock() // To protect metrics from concurrent collects.
	defer e.mutex.Unlock()

	sendResult := func() {
		ch <- e.up
		e.serviceStatus.Collect(ch)
	}
	defer sendResult()

	result := e.performAllChecks()

	overall := 1.0
	for k, v := range result {
		glog.Infof("service status %s at %s", v.Status, k)
		serviceUp := 0.0
		if v.IsOk {
			serviceUp = 1.0
		} else {
			overall = 0.0
		}
		e.serviceStatus.With(v.Labels).Set(serviceUp)
	}
	e.up.Set(overall)
	return
}

func (e *Exporter) performCheck(service Service) (*HealthCheckResult, error) {
	labels := prometheus.Labels{}
	for label, value := range service.Labels {
		labels[label] = value
	}
	labels["uri"] = service.Uri

	resp, err := e.client.Get(service.Uri)
	if err != nil {
		glog.Errorf("Error reading from URI %s: %v", service.Uri, err)
		status := &HealthCheckResult{Status:"ERROR", IsOk:false, Labels:labels}
		return status, err
	}
	defer resp.Body.Close()

	isStatusOk := resp.StatusCode >= 200 && resp.StatusCode < 400
	status := &HealthCheckResult{Status:fmt.Sprintf("%d", resp.StatusCode), IsOk:isStatusOk, Labels:labels}
	return status, nil
}

func (e *Exporter) performAllChecks() (OverallHealthCheckResult) {
	result := make(OverallHealthCheckResult)
	for _, service := range e.config.Services {
		status, _ := e.performCheck(service)
		result[service.Uri] = *status
	}
	return result
}
