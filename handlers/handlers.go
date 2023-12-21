package handlers

import (
	"fmt"
	"github.com/akyriako/cloudeye-exporter/collector"
	"github.com/akyriako/cloudeye-exporter/config"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"log/slog"
	"net/http"
	"strings"
)

func Health(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, err := w.Write([]byte("pong"))
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func Metrics(cloudConfig *config.CloudConfig) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		target := r.URL.Query().Get("services")
		if target == "" {
			http.Error(w, "'target' parameter must be specified", http.StatusBadRequest)
			return
		}

		targets := strings.Split(target, ",")
		registry := prometheus.NewRegistry()

		slog.Info("starting cloudeye collector", "targets", targets)
		cloudEyeCollector, err := collector.NewCloudEyeCollector(cloudConfig, targets)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			_, err := w.Write([]byte(err.Error()))
			if err != nil {
				slog.Error(fmt.Sprintf("writing response body failed: %s", err.Error()))
				return
			}
			return
		}
		registry.MustRegister(cloudEyeCollector)
		if err != nil {
			slog.Error(fmt.Sprintf("registering cloudeye collector in prometheus failed: %+v, err: %s", targets, err.Error()))
			return
		}

		h := promhttp.HandlerFor(registry, promhttp.HandlerOpts{})
		h.ServeHTTP(w, r)
	}
}

func Welcome(metricsPath string) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte(`<html>
             <head><title>Open Telekom Cloud CloudEye Exporter</title></head>
             <body>
             <h1>Open Telekom Cloud CloudEye Exporter</h1>
             <p><a href='` + metricsPath + "?services=SYS.ELB" + `'>ELB Metrics</a></p>
             <p><a href='` + metricsPath + "?services=SYS.RDS" + `'>RDS Metrics</a></p>
             <p><a href='` + metricsPath + "?services=SYS.DCS" + `'>DCS Metrics</a></p>
             <p><a href='` + metricsPath + "?services=SYS.NAT" + `'>NAT Metrics</a></p>
             <p><a href='` + metricsPath + "?services=SYS.VPC" + `'>VPC Metrics</a></p>
             <p><a href='` + metricsPath + "?services=SYS.ECS" + `'>ECS Metrics</a></p>
             </body>
             </html>`))
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}
}
