package main

import (
	"flag"
	"fmt"
	"github.com/akyriako/cloudeye-exporter/config"
	"github.com/akyriako/cloudeye-exporter/handlers"
	"log/slog"
	"net/http"
	"os"
	"strings"
)

var (
	cloudConfigFlag  = flag.String("config", "./clouds.yaml", "path to the cloud configuration file")
	enableFilterFlag = flag.Bool("enable-filters", false, "enabling monitoring metric filter")
	debugFlag        = flag.Bool("debug", false, "debug mode")

	logger *slog.Logger
)

const (
	exitCodeConfigurationError  int = 1
	exitCodeListenAndServeError int = 2
)

func main() {
	flag.Parse()

	initializeLogger()
	cloudConfig, err := config.GetConfigFromFile(*cloudConfigFlag, *enableFilterFlag)
	if err != nil {
		wd, wderr := os.Getwd()
		if wderr != nil {
			slog.Error(fmt.Sprintf("parsing cloud config failed: %s", wderr.Error()))
			os.Exit(exitCodeConfigurationError)
		}

		slog.Error(fmt.Sprintf("parsing cloud config at %s%s failed: %s", wd, strings.Trim(*cloudConfigFlag, "."), err.Error()))
		os.Exit(exitCodeConfigurationError)
	}

	http.HandleFunc(cloudConfig.Global.MetricsPath, handlers.Metrics(cloudConfig))
	http.HandleFunc("/healthz", handlers.Health)
	http.HandleFunc("/livez", handlers.Health)
	http.HandleFunc("/readyz", handlers.Health)
	http.HandleFunc("/", handlers.Welcome(cloudConfig.Global.MetricsPath))

	slog.Info(fmt.Sprintf("listening at 0.0.0.0%s%s", cloudConfig.Global.Port, cloudConfig.Global.MetricsPath))
	if err := http.ListenAndServe(cloudConfig.Global.Port, nil); err != nil {
		slog.Error(fmt.Sprintf("error occur when start server %s", err.Error()))
		os.Exit(exitCodeListenAndServeError)
	}
}

func initializeLogger() {
	levelInfo := slog.LevelInfo
	if *debugFlag {
		levelInfo = slog.LevelDebug
	}

	logger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: levelInfo,
	}))

	slog.SetDefault(logger)
}
