package collector

import (
	"bytes"
	"fmt"
	"github.com/coreos/go-systemd/dbus"
	"github.com/prometheus/client_golang/prometheus"
	"os/exec"
)

func init() {
	registerCollector("kube_scheduler", defaultEnabled, NewSchedulerCollector)
}

type schedulerCollector struct {
	systemdClient *dbus.Conn
	requestsDesc  *prometheus.Desc
}

func NewSchedulerCollector() (Collector, error) {
	systemdClient, err := dbus.NewSystemdConnection()

	if err != nil {
		return nil, fmt.Errorf("failed to open systemd dbus: %v", err)
	}

	return &schedulerCollector{
		systemdClient: systemdClient,
		requestsDesc: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "kube_scheduler", "health"),
			"kube_scheduler health.",
			[]string{"proto", "method"}, nil,
		),
	}, nil
}

func (kube *schedulerCollector) Update(ch chan<- prometheus.Metric) error {
	kube.updateStatus(ch)
	defer kube.systemdClient.Close()
	return nil
}

func (kube *schedulerCollector) updateStatus(ch chan<- prometheus.Metric) {
	p, err := kube.systemdClient.GetUnitProperties("kube-scheduler.service")
	var health bool
	if err != nil {
		health = false
	} else if p["FragmentPath"] == "" {
		return
	} else if p["SubState"] != "running" {
		health = false
	} else {
		health = true
	}

	if health {
		ch <- prometheus.MustNewConstMetric(
			prometheus.NewDesc(
				prometheus.BuildFQName(namespace, "kube_scheduler", "health"),
				"kube_scheduler Health Status",
				nil,
				nil,
			),
			prometheus.CounterValue,
			0)
	} else {
		ch <- prometheus.MustNewConstMetric(
			prometheus.NewDesc(
				prometheus.BuildFQName(namespace, "kube_scheduler", "health"),
				"kube_scheduler Health Status",
				nil,
				nil,
			),
			prometheus.CounterValue,
			1)
	}

	cmd := exec.Command("kube-scheduler", "--version")
	var out bytes.Buffer
	cmd.Stdout = &out
	buildInfo := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "kube_scheduler_version",
			Help: "A metric with a constant '1' value labeled by Version",
		},
		[]string{"Version"},
	)
	err = cmd.Run()

	if err == nil {
		ch <- buildInfo.WithLabelValues(out.String())
	} else {
		ch <- buildInfo.WithLabelValues("Unknow")
	}
}
