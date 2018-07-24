package collector

import (
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/coreos/go-systemd/dbus"
)

func init() {
	registerCollector("kube_component", defaultEnabled, NewKubeNodeCollector)
}

const (
	kubeSlave = "kubeslave"
)


type kubeSlaveCollector struct {
	systemdClient *dbus.Conn
	requestsDesc *prometheus.Desc
}


func NewKubeNodeCollector() (Collector, error) {
	systemdClient, err := dbus.NewSystemdConnection()

	if err != nil {
		return nil, fmt.Errorf("failed to open systemd dbus: %v", err)
	}


	return &kubeSlaveCollector{
		systemdClient: systemdClient,
		requestsDesc: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, kubeSlave, "health"),
			"kube slave node health.",
			[]string{"proto", "method"}, nil,
		),
	}, nil
}

func (kube *kubeSlaveCollector) Update(ch chan<- prometheus.Metric)  error {
	kube.updateKubeletStatus(ch)
	kube.updateKubeProxyStatus(ch)
	kube.updateDockerdStatus(ch)
	kube.updateWsagentStatus(ch)
	kube.updateFlannelStatus(ch)
	return nil
}

func (kube *kubeSlaveCollector) updateFlannelStatus(ch chan<- prometheus.Metric) {
	p, err := kube.systemdClient.GetUnitProperties("flannel.service")
	var health bool
	if err != nil {
		health = false
	} else if p["SubState"] != "running" {
		health = false
	} else {
		health = true
	}

	if health {
		ch <- prometheus.MustNewConstMetric(
			prometheus.NewDesc(
				prometheus.BuildFQName(namespace, kubeSlave, "flannel_health"),
				"flannel Health Status",
				nil,
				nil,
			),
			prometheus.CounterValue,
			0)
	} else {
		ch <- prometheus.MustNewConstMetric(
			prometheus.NewDesc(
				prometheus.BuildFQName(namespace, kubeSlave, "flannel_health"),
				"flannel Health Status",
				nil,
				nil,
			),
			prometheus.CounterValue,
			1)
	}
}

func (kube *kubeSlaveCollector) updateWsagentStatus(ch chan<- prometheus.Metric) {
	p, err := kube.systemdClient.GetUnitProperties("wsagent.service")
	var health bool
	if err != nil {
		health = false
	} else if p["SubState"] != "running" {
		health = false
	} else {
		health = true
	}

	if health {
		ch <- prometheus.MustNewConstMetric(
			prometheus.NewDesc(
				prometheus.BuildFQName(namespace, kubeSlave, "wsagent_health"),
				"wsagent Health Status",
				nil,
				nil,
			),
			prometheus.CounterValue,
			0)
	} else {
		ch <- prometheus.MustNewConstMetric(
			prometheus.NewDesc(
				prometheus.BuildFQName(namespace, kubeSlave, "wsagent_health"),
				"wsagent Health Status",
				nil,
				nil,
			),
			prometheus.CounterValue,
			1)
	}
}

func (kube *kubeSlaveCollector) updateDockerdStatus(ch chan<- prometheus.Metric) {
	p, err := kube.systemdClient.GetUnitProperties("docker.service")
	var health bool
	if err != nil {
		health = false
	} else if p["SubState"] != "running" {
		health = false
	} else {
		health = true
	}

	if health {
		ch <- prometheus.MustNewConstMetric(
			prometheus.NewDesc(
				prometheus.BuildFQName(namespace, kubeSlave, "docker_health"),
				"docker Health Status",
				nil,
				nil,
			),
			prometheus.CounterValue,
			0)
	} else {
		ch <- prometheus.MustNewConstMetric(
			prometheus.NewDesc(
				prometheus.BuildFQName(namespace, kubeSlave, "docker_health"),
				"docker Health Status",
				nil,
				nil,
			),
			prometheus.CounterValue,
			1)
	}
}

func (kube *kubeSlaveCollector) updateKubeProxyStatus(ch chan<- prometheus.Metric) {
	p, err := kube.systemdClient.GetUnitProperties("kube-proxy.service")
	var health bool
	if err != nil {
		health = false
	} else if p["SubState"] != "running" {
		health = false
	} else {
		health = true
	}

	if health {
		ch <- prometheus.MustNewConstMetric(
			prometheus.NewDesc(
				prometheus.BuildFQName(namespace, kubeSlave, "kubeproxy_health"),
				"Kube-Proxy Health Status",
				nil,
				nil,
			),
			prometheus.CounterValue,
			0)
	} else {
		ch <- prometheus.MustNewConstMetric(
			prometheus.NewDesc(
				prometheus.BuildFQName(namespace, kubeSlave, "kubeproxy_health"),
				"Kube-Proxy Health Status",
				nil,
				nil,
			),
			prometheus.CounterValue,
			1)
	}
}

func (kube *kubeSlaveCollector) updateKubeletStatus(ch chan<- prometheus.Metric) {
	p, err := kube.systemdClient.GetUnitProperties("kubelet.service")
	var health bool
	if err != nil {
		health = false
	} else if p["SubState"] != "running" {
		health = false
	} else {
		health = true
	}

	if health {
		ch <- prometheus.MustNewConstMetric(
			prometheus.NewDesc(
				prometheus.BuildFQName(namespace, kubeSlave, "kubelet_health"),
				"Kubelet Health Status",
				nil,
				nil,
			),
			prometheus.CounterValue,
			0)
	} else {
		ch <- prometheus.MustNewConstMetric(
			prometheus.NewDesc(
				prometheus.BuildFQName(namespace, kubeSlave, "kubelet_health"),
				"Kubelet Health Status",
				nil,
				nil,
			),
			prometheus.CounterValue,
			1)
	}
}