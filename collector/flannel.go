package collector

import (
	"fmt"
	"github.com/coreos/go-systemd/dbus"
	"github.com/prometheus/client_golang/prometheus"
)

func init() {
	registerCollector("flannel", defaultEnabled, NewFlannelCollector)
}

type flannelCollector struct {
	systemdClient *dbus.Conn
	requestsDesc  *prometheus.Desc
}

func NewFlannelCollector() (Collector, error) {
	systemdClient, err := dbus.NewSystemdConnection()

	if err != nil {
		return nil, fmt.Errorf("failed to open systemd dbus: %v", err)
	}

	return &flannelCollector{
		systemdClient: systemdClient,
		requestsDesc: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "flannel", "health"),
			"flannel health.",
			[]string{"proto", "method"}, nil,
		),
	}, nil
}

func (kube *flannelCollector) Update(ch chan<- prometheus.Metric) error {
	kube.updateStatus(ch)
	defer kube.systemdClient.Close()
	return nil
}

func (kube *flannelCollector) updateStatus(ch chan<- prometheus.Metric) {
	p, err := kube.systemdClient.GetUnitProperties("flannel.service")
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
				prometheus.BuildFQName(namespace, "flannel", "flannel_health"),
				"flannel Health Status",
				nil,
				nil,
			),
			prometheus.CounterValue,
			0)
	} else {
		ch <- prometheus.MustNewConstMetric(
			prometheus.NewDesc(
				prometheus.BuildFQName(namespace, "flannel", "flannel_health"),
				"flannel Health Status",
				nil,
				nil,
			),
			prometheus.CounterValue,
			1)
	}


	//cmd := exec.Command("flanneld", "--version")
	//var out bytes.Buffer
	//cmd.Stderr = &out
	//buildInfo := prometheus.NewGaugeVec(
	//	prometheus.GaugeOpts{
	//		Name: "flannel_version",
	//		Help: "A metric with a constant '1' value labeled by Version",
	//	},
	//	[]string{"Version"},
	//)
	//err = cmd.Run()
	//
	//if err == nil {
	//	version := strings.Trim(out.String(), "\n")
	//	ch <- buildInfo.WithLabelValues(version)
	//} else {
	//	ch <- buildInfo.WithLabelValues("Unknow")
	//}
}
