package collector

import (
	"fmt"
	"github.com/coreos/go-systemd/dbus"
	"github.com/prometheus/client_golang/prometheus"
)

func init() {
	registerCollector("wsagent", defaultEnabled, NewWsagentCollector)
}

type wsagentCollector struct {
	systemdClient *dbus.Conn
	requestsDesc  *prometheus.Desc
}

func NewWsagentCollector() (Collector, error) {
	systemdClient, err := dbus.NewSystemdConnection()

	if err != nil {
		return nil, fmt.Errorf("failed to open systemd dbus: %v", err)
	}

	return &wsagentCollector{
		systemdClient: systemdClient,
		requestsDesc: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "wsagent", "health"),
			"wsagent health.",
			[]string{"proto", "method"}, nil,
		),
	}, nil
}

func (kube *wsagentCollector) Update(ch chan<- prometheus.Metric) error {
	kube.updateStatus(ch)
	defer kube.systemdClient.Close()
	return nil
}

func (kube *wsagentCollector) updateStatus(ch chan<- prometheus.Metric) {
	p, err := kube.systemdClient.GetUnitProperties("wsagent.service")
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
				prometheus.BuildFQName(namespace, "wsagent", "health"),
				"wsagent Health Status",
				nil,
				nil,
			),
			prometheus.CounterValue,
			0)
	} else {
		ch <- prometheus.MustNewConstMetric(
			prometheus.NewDesc(
				prometheus.BuildFQName(namespace, "wsagent", "health"),
				"wsagent Health Status",
				nil,
				nil,
			),
			prometheus.CounterValue,
			1)
	}
}
