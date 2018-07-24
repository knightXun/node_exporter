// Copyright 2015 The Prometheus Authors
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
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"sort"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/log"
	"github.com/prometheus/common/version"
	"github.com/prometheus/node_exporter/collector"
	"gopkg.in/alecthomas/kingpin.v2"
	"github.com/prometheus/node_exporter/monitor"
	"github.com/prometheus/node_exporter/utils"
	"os"
)


func init() {
	prometheus.MustRegister(version.NewCollector("node_exporter"))
}

func handler(w http.ResponseWriter, r *http.Request) {
	filters := r.URL.Query()["collect[]"]
	log.Debugln("collect query:", filters)

	nc, err := collector.NewNodeCollector(filters...)
	if err != nil {
		log.Warnln("Couldn't create", err)
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(fmt.Sprintf("Couldn't create %s", err)))
		return
	}

	registry := prometheus.NewRegistry()
	err = registry.Register(nc)
	if err != nil {
		log.Errorln("Couldn't register collector:", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(fmt.Sprintf("Couldn't register collector: %s", err)))
		return
	}

	gatherers := prometheus.Gatherers{
		prometheus.DefaultGatherer,
		registry,
	}
	// Delegate http serving to Prometheus client library, which will call collector.Collect.
	h := promhttp.InstrumentMetricHandler(
		registry,
		promhttp.HandlerFor(gatherers,
			promhttp.HandlerOpts{
				ErrorLog:      log.NewErrorLogger(),
				ErrorHandling: promhttp.ContinueOnError,
			}),
	)
	h.ServeHTTP(w, r)
}

func cephHandler(w http.ResponseWriter, r *http.Request) {
	filters := []string{"ceph_cluster_usage","ceph_pool_usage","ceph_cluster_health","ceph_monitor","ceph_OSD"}
	//filters := []string{"ceph_cluster_usage"}
	log.Infoln("collect query:", filters)

	nc, err := collector.NewNodeCollector(filters...)
	if err != nil {
		log.Warnln("Couldn't create", err)
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(fmt.Sprintf("Couldn't create %s", err)))
		return
	}

	registry := prometheus.NewRegistry()
	err = registry.Register(nc)
	if err != nil {
		log.Errorln("Couldn't register collector:", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(fmt.Sprintf("Couldn't register collector: %s", err)))
		return
	}

	gatherers := prometheus.Gatherers{
		prometheus.DefaultGatherer,
		registry,
	}
	// Delegate http serving to Prometheus client library, which will call collector.Collect.
	h := promhttp.InstrumentMetricHandler(
		registry,
		promhttp.HandlerFor(gatherers,
			promhttp.HandlerOpts{
				ErrorLog:      log.NewErrorLogger(),
				ErrorHandling: promhttp.ContinueOnError,
			}),
	)
	h.ServeHTTP(w, r)
}

func main() {
	var (
		listenAddress = kingpin.Flag("web.listen-address", "Address on which to expose metrics and web interface.").Default(":9100").String()
		metricsPath   = kingpin.Flag("web.telemetry-path", "Path under which to expose metrics.").Default("/metrics").String()
		MonitConfigPath = kingpin.Flag("monit-config-path", "monit config file path.").Default("/etc/monit.json").String()
		enable_monit = kingpin.Flag("enable-monit", "enable monitor").Bool()
		enable_ceph = kingpin.Flag("enable-ceph", "enable ceph collector").Bool()
		exporterConfig = kingpin.Flag("exporter.config", "Path to ceph exporter config.").Default("/etc/ceph/exporter.yml").String()
	)

	log.AddFlags(kingpin.CommandLine)
	kingpin.Version(version.Print("node_exporter"))
	kingpin.HelpFlag.Short('h')
	kingpin.Parse()

	log.Infoln("Starting node_exporter", version.Info())
	log.Infoln("Build context", version.BuildContext())

	// monit handler
	if *enable_monit == true {
		monitCfg, err := monitor.EncodeFromFile(*MonitConfigPath)
		if err != nil {
			fmt.Println("MonitConfigPath: ", *MonitConfigPath)
			log.Fatalf("Couldn't Parse monit config file: %s", err)
			os.Exit(1)
		}
		go monitCfg.MainLoop()
		http.HandleFunc("/monit_status", monitCfg.Status)
	}

	nc, err := collector.NewNodeCollector()
	if *enable_ceph == true {
		if utils.FileExists(*exporterConfig) {
			cluster_usage, err := collector.NewClusterUsageCollector()
			if err == nil {
				log.Infoln("make cluster usage collector successfully")
				nc.Collectors["ceph_cluster_usage"] = cluster_usage
			} else {
				log.Infoln(err.Error())
			}

			pool_usage, err := collector.NewPoolUsageCollector()
			if err == nil {
				nc.Collectors["ceph_pool_usage"] = pool_usage
			} else {
				log.Infoln(err.Error())
			}

			cluster_health, err := collector.NewClusterHealthCollector()
			if err == nil {
				nc.Collectors["ceph_cluster_health"] = cluster_health
			} else {
				log.Infoln(err.Error())
			}

			ceph_monitors, err := collector.NewMonitorCollector()
			if err == nil {
				nc.Collectors["ceph_monitor"] = ceph_monitors
			} else {
				log.Infoln(err.Error())
			}

			osd , err := collector.NewOSDCollector()
			if err == nil {
				nc.Collectors["ceph_OSD"] = osd
			} else {
				log.Infoln(err.Error())
			}

		} else {
			log.Fatalln("Couldn't create ceph collector: lost /etc/ceph/exporter.yml")
			os.Exit(1)
		}
	}
	// This instance is only used to check collector creation and logging.
	if err != nil {
		log.Fatalf("Couldn't create collector: %s", err)
	}
	log.Infof("Enabled collectors:")
	collectors := []string{}
	for n := range nc.Collectors {
		collectors = append(collectors, n)
	}
	sort.Strings(collectors)
	for _, n := range collectors {
		log.Infof(" - %s", n)
	}

	http.HandleFunc(*metricsPath, handler)
	http.HandleFunc("/ceph", cephHandler)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html>
			<head><title>Node Exporter</title></head>
			<body>
			<h1>Node Exporter</h1>
			<p><a href="` + *metricsPath + `">Metrics</a></p>
			</body>
			</html>`))
	})

	fmt.Println("begin to Listen")
	log.Infoln("Listening on", *listenAddress)
	err = http.ListenAndServe(*listenAddress, nil)
	if err != nil {
		log.Fatal(err)
	}
}