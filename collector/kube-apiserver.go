package collector

import (
	"fmt"
	"github.com/coreos/go-systemd/dbus"
	"github.com/prometheus/client_golang/prometheus"
	"os/exec"
	"bytes"
)

func init() {
	registerCollector("kube_apiserver", defaultEnabled, NewApiserverCollector)
}

type apiserverCollector struct {
	systemdClient *dbus.Conn
	requestsDesc  *prometheus.Desc
}

func NewApiserverCollector() (Collector, error) {
	systemdClient, err := dbus.NewSystemdConnection()

	if err != nil {
		return nil, fmt.Errorf("failed to open systemd dbus: %v", err)
	}

	return &apiserverCollector{
		systemdClient: systemdClient,
		requestsDesc: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "kube_apiserver", "health"),
			"kube_apiserver health.",
			[]string{"proto", "method"}, nil,
		),
	}, nil
}

func (kube *apiserverCollector) Update(ch chan<- prometheus.Metric) error {
	kube.updateStatus(ch)
	defer kube.systemdClient.Close()
	return nil
}

func (kube *apiserverCollector) updateStatus(ch chan<- prometheus.Metric) {
	p, err := kube.systemdClient.GetUnitProperties("kube-apiserver.service")
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
				prometheus.BuildFQName(namespace, "kube_apiserver", "health"),
				"kube_apiserver Health Status",
				nil,
				nil,
			),
			prometheus.CounterValue,
			0)
	} else {
		ch <- prometheus.MustNewConstMetric(
			prometheus.NewDesc(
				prometheus.BuildFQName(namespace, "kube_apiserver", "health"),
				"kube_apiserver Health Status",
				nil,
				nil,
			),
			prometheus.CounterValue,
			1)
	}


	cmd := exec.Command("kube-apiserver", "--version")
	var out bytes.Buffer
	cmd.Stdout = &out
	buildInfo := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "kube_apiserver_version",
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
