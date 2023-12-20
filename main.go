package main

import (
	"flag"
	"github.com/akyriako/cloudeye-exporter/collector"
	"github.com/akyriako/cloudeye-exporter/logging"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"net/http"
	"os"
	"strings"
)

var (
	clientConfig = flag.String("config", "./clouds.yaml", "Path to the cloud configuration file")
	filterEnable = flag.Bool("filter-enable", false, "Enabling monitoring metric filter")
	debug        = flag.Bool("debug", false, "If debug the code.")
)

func main() {
	flag.Parse()
	logging.InitLogger(*debug)
	config, err := collector.NewCloudConfigFromFile(*clientConfig)
	if err != nil {
		logging.Logger.Error("New Cloud Config From File error: %s", err.Error())
		return
	}
	err = collector.InitFilterConfig(*filterEnable)
	if err != nil {
		logging.Logger.Error("Init filter Config error: %s", err.Error())
		return
	}

	http.HandleFunc(config.Global.MetricPath, metrics)
	http.HandleFunc("/health", health)
	http.HandleFunc("/ping", health)

	logging.Logger.Info("Start server", "port", config.Global.Port)
	if err := http.ListenAndServe(config.Global.Port, nil); err != nil {
		logging.Logger.Error("Error occur when start server %s", err.Error())
		os.Exit(1)
	}
}

func metrics(w http.ResponseWriter, r *http.Request) {
	target := r.URL.Query().Get("services")
	if target == "" {
		http.Error(w, "'target' parameter must be specified", http.StatusBadRequest)
		return
	}

	targets := strings.Split(target, ",")
	registry := prometheus.NewRegistry()

	logging.Logger.Info("Start to monitor services: %s", targets)
	exporter, err := collector.GetMonitoringCollector(*clientConfig, targets)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, err := w.Write([]byte(err.Error()))
		if err != nil {
			logging.Logger.Error("Fail to write response body, error: %s", err.Error())
			return
		}
		return
	}
	registry.MustRegister(exporter)
	if err != nil {
		logging.Logger.Error("Fail to start to morning services: %+v, err: %s", targets, err.Error())
		return
	}

	h := promhttp.HandlerFor(registry, promhttp.HandlerOpts{})
	h.ServeHTTP(w, r)
}

func health(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, err := w.Write([]byte("pong"))
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}
