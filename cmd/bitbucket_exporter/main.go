// Copyright 2022 The Prometheus Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/alecthomas/kingpin/v2"
	"github.com/nandanurseptama/bitbucket-exporter/collector"
	"github.com/nandanurseptama/bitbucket-exporter/config"
	"github.com/prometheus/client_golang/prometheus"
	versioncollector "github.com/prometheus/client_golang/prometheus/collectors/version"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/promslog"
	"github.com/prometheus/common/promslog/flag"
	"github.com/prometheus/common/version"
	"github.com/prometheus/exporter-toolkit/web"
	"github.com/prometheus/exporter-toolkit/web/kingpinflag"
)

var (
	c = config.Handler{
		Config: &config.Config{},
	}

	configFile   = kingpin.Flag("config.file", "Bitbucket exporter configuration file.").Default("config.yaml").String()
	metricsPath  = kingpin.Flag("web.telemetry-path", "Path under which to expose metrics.").Default("/metrics").Envar("BITBUCKET_EXPORTER_WEB_TELEMETRY_PATH").String()
	webConfig    = kingpinflag.AddFlags(kingpin.CommandLine, ":9171")
	logger       = promslog.NewNopLogger()
	fromPromFile = kingpin.Flag("metric.from-prom-file", "Whether to expose metric from .prom file").Default("false").Bool()
	promfile     = kingpin.Flag("metric.prom-file-path", "File path of prom file").Default("example-output.prom").String()
)

// Metric name parts.
const (
	// Namespace for all metrics.
	namespace = "bitbucket"
	// Subsystems.
	exporter = "exporter"
	// The name of the exporter.
	exporterName = "bitbucket_exporter"
	// Metric label used for static string data thats handy to send to Prometheus
	// e.g. version
	staticLabelName = "static"
	// Metric label used for server identification.
	serverLabelName = "server"
)

func main() {
	kingpin.Version(version.Print(exporterName))
	promslogConfig := &promslog.Config{}
	flag.AddFlags(kingpin.CommandLine, promslogConfig)
	kingpin.HelpFlag.Short('h')
	kingpin.Parse()
	logger = promslog.New(promslogConfig)

	if err := c.ReloadConfig(*configFile, logger); err != nil {
		logger.Warn("Error loading config", "err", err)
	}

	prometheus.MustRegister(versioncollector.NewCollector(exporterName))

	exporters := collector.NewBitbucketCollector(logger, c.Config)
	collectors := exporters.GetCollectors()
	prometheus.MustRegister(collectors...)

	if fromPromFile != nil && *fromPromFile {
		http.Handle(*metricsPath, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fileBytes, err := os.ReadFile(*promfile)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte("internal server error"))
				return
			}
			w.Header().Add("Content-Type", "text/plain; version=0.0.4; charset=utf-8; escaping=underscores")
			w.WriteHeader(http.StatusOK)
			w.Write(fileBytes)
		}))
	} else {
		http.Handle(*metricsPath, promhttp.Handler())
	}

	if *metricsPath != "/" && *metricsPath != "" {
		landingConfig := web.LandingConfig{
			Name:        "Postgres Exporter",
			Description: "Prometheus PostgreSQL server Exporter",
			Version:     version.Info(),
			Links: []web.LandingLinks{
				{
					Address: *metricsPath,
					Text:    "Metrics",
				},
			},
		}
		landingPage, err := web.NewLandingPage(landingConfig)
		if err != nil {
			logger.Error("error creating landing page", "err", err)
			os.Exit(1)
		}
		http.Handle("/", landingPage)
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	srv := &http.Server{}

	go func() {
		if err := web.ListenAndServe(srv, webConfig, logger); err != nil {
			logger.Error("Error running HTTP server", "err", err)
			os.Exit(1)
		}
	}()

	go func() {
		if fromPromFile == nil {
			return
		}
		if *fromPromFile {
			return
		}
		exporters.Exec(ctx)
	}()

	<-ctx.Done()
	logger.Info("Shutting down server...")

	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("Server forced to shutdown", "err", err)
	}

	logger.Info("Server exited gracefully")

}
