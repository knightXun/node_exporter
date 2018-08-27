package collector

import (
	"fmt"
	"github.com/coreos/go-systemd/dbus"
	"github.com/prometheus/client_golang/prometheus"
	"os/exec"
	"bytes"
)

func init() {
	registerCollector("kube_controller_manager", defaultEnabled, NewControllerCollector)
}

type controllerCollector struct {
	systemdClient *dbus.Conn
	requestsDesc  *prometheus.Desc
}

func NewControllerCollector() (Collector, error) {
	systemdClient, err := dbus.NewSystemdConnection()

	if err != nil {
		return nil, fmt.Errorf("failed to open systemd dbus: %v", err)
	}

	return &controllerCollector{
		systemdClient: systemdClient,
		requestsDesc: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "kube_controller_manager", "health"),
			"kube_controller_manager health.",
			[]string{"proto", "method"}, nil,
		),
	}, nil
}

func (kube *controllerCollector) Update(ch chan<- prometheus.Metric) error {
	kube.updateStatus(ch)
	defer kube.systemdClient.Close()
	return nil
}

func (kube *controllerCollector) updateStatus(ch chan<- prometheus.Metric) {
	p, err := kube.systemdClient.GetUnitProperties("kube-controller-manager.service")
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
				prometheus.BuildFQName(namespace, "kube_controller_manager", "health"),
				"kube_controller_manager Health Status",
				nil,
				nil,
			),
			prometheus.CounterValue,
			0)
	} else {
		ch <- prometheus.MustNewConstMetric(
			prometheus.NewDesc(
				prometheus.BuildFQName(namespace, "kube_controller_manager", "health"),
				"kube_controller_manager Health Status",
				nil,
				nil,
			),
			prometheus.CounterValue,
			1)
	}

	cmd := exec.Command("kube-controller-manager", "--version")
	var out bytes.Buffer
	cmd.Stdout = &out
	buildInfo := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "kube_controller_manager_version",
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
