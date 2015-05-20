package main

import (
	"flag"
	"fmt"
	"net"
	"net/http"

	"github.com/golang/glog"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	configFile = flag.String("config.file", "config.json", "Path to config file.")
	listenAddress = flag.String("telemetry.address", ":9990", "Address on which to expose metrics.")
	metricsEndpoint = flag.String("telemetry.endpoint", "/metrics", "Path under which to expose metrics.")
)

func main() {
	flag.Parse()

	config, err := readConfig(*configFile)
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

	glog.Infof("starting exporter at tcp4://%s", *listenAddress)
	listener, err := net.Listen("tcp4", *listenAddress)
	if err != nil {
		glog.Fatal(err)
	}
	err = http.Serve(listener, nil)
	if err != nil {
		glog.Fatal(err)
	}
}
