package collector

import (
	"fmt"
	"github.com/coreos/go-systemd/dbus"
	"github.com/prometheus/client_golang/prometheus"
	"os/exec"
	"bytes"
	"strings"
)

func init() {
	registerCollector("etcd", defaultEnabled, NewEtcdCollector)
}

type etcdCollector struct {
	systemdClient *dbus.Conn
	requestsDesc  *prometheus.Desc
}

func NewEtcdCollector() (Collector, error) {
	systemdClient, err := dbus.NewSystemdConnection()

	if err != nil {
		return nil, fmt.Errorf("failed to open systemd dbus: %v", err)
	}

	return &etcdCollector{
		systemdClient: systemdClient,
		requestsDesc: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "etcd", "health"),
			"etcd health.",
			[]string{"proto", "method"}, nil,
		),
	}, nil
}

func (kube *etcdCollector) Update(ch chan<- prometheus.Metric) error {
	kube.updateStatus(ch)
	defer kube.systemdClient.Close()
	return nil
}

func (kube *etcdCollector) updateStatus(ch chan<- prometheus.Metric) {
	p, err := kube.systemdClient.GetUnitProperties("etcd.service")
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
				prometheus.BuildFQName(namespace, "etcd", "health"),
				"etcd Health Status",
				nil,
				nil,
			),
			prometheus.CounterValue,
			0)
	} else {
		ch <- prometheus.MustNewConstMetric(
			prometheus.NewDesc(
				prometheus.BuildFQName(namespace, "etcd", "health"),
				"etcd Health Status",
				nil,
				nil,
			),
			prometheus.CounterValue,
			1)
	}

	cmd := exec.Command("etcd", "--version")
	var out bytes.Buffer
	cmd.Stdout = &out
	buildInfo := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "etcd_version",
			Help: "A metric with a constant '1' value labeled by Version",
		},
		[]string{"etcd_Version", "Git_SHA", "Go_Version", "Go_OSArch"},
	)
	err = cmd.Run()

	if err == nil {
		version:= ""
		git := ""
		goVersion := ""
		goOs := ""
		info := out.String()
		lines := strings.Split(info, "\n")
		for _, line := range lines {
			if strings.Contains(line, "etcd Version") {
				sep := strings.Split(line, ":")
				if len(sep) == 2 {
					version = strings.Trim(sep[1], " ")
					version = strings.Trim(version, "\t")
					version = strings.Trim(version, "\n")
				}
			}
			if strings.Contains(line, "Git SHA") {
				sep := strings.Split(line, ":")
				if len(sep) == 2 {
					git = strings.Trim(sep[1], " ")
					git = strings.Trim(git, "\t")
					git = strings.Trim(git, "\n")
				}
			}
			if strings.Contains(line, "Go Version") {
				sep := strings.Split(line, ":")
				if len(sep) == 2 {
					goVersion = strings.Trim(sep[1], " ")
					goVersion = strings.Trim(goVersion, "\t")
					goVersion = strings.Trim(goVersion, "\n")
				}
			}
			if strings.Contains(line, "Go OS/Arch") {
				sep := strings.Split(line, ":")
				if len(sep) == 2 {
					goOs = strings.Trim(sep[1], " ")
					goOs = strings.Trim(goOs, "\t")
					goOs = strings.Trim(goOs, "\n")
				}
			}
		}
		ch <- buildInfo.WithLabelValues(version, git, goVersion, goOs)
	} else {
		ch <- buildInfo.WithLabelValues("Unknow", "Unknow", "Unknow","Unknow" )
	}
}
