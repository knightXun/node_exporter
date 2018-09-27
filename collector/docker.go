package collector

import (
	"bytes"
	"fmt"
	"github.com/coreos/go-systemd/dbus"
	"github.com/prometheus/client_golang/prometheus"
	"os/exec"
	"strings"
)

func init() {
	registerCollector("docker", defaultEnabled, NewDockerCollector)
}

type dockerCollector struct {
	systemdClient *dbus.Conn
	requestsDesc  *prometheus.Desc
}

func NewDockerCollector() (Collector, error) {
	systemdClient, err := dbus.NewSystemdConnection()

	if err != nil {
		return nil, fmt.Errorf("failed to open systemd dbus: %v", err)
	}

	return &dockerCollector{
		systemdClient: systemdClient,
		requestsDesc: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "docker", "health"),
			"docker health.",
			[]string{"proto", "method"}, nil,
		),
	}, nil
}

func (kube *dockerCollector) Update(ch chan<- prometheus.Metric) error {
	kube.updateStatus(ch)
	defer kube.systemdClient.Close()
	return nil
}

func (kube *dockerCollector) updateStatus(ch chan<- prometheus.Metric) {
	p, err := kube.systemdClient.GetUnitProperties("docker.service")
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
				prometheus.BuildFQName(namespace, "docker", "health"),
				"docker Health Status",
				nil,
				nil,
			),
			prometheus.CounterValue,
			0)
	} else {
		ch <- prometheus.MustNewConstMetric(
			prometheus.NewDesc(
				prometheus.BuildFQName(namespace, "docker", "health"),
				"docker Health Status",
				nil,
				nil,
			),
			prometheus.CounterValue,
			1)
	}

	cmd := exec.Command("docker", "version")
	var out bytes.Buffer
	cmd.Stdout = &out
	err = cmd.Run()

	if err == nil {
		info := out.String()
		lines := strings.Split(info, "\n")

		clientVersion := ""
		clientApiVersion := ""
		clientGoVersion := ""
		clientGitCommit := ""
		clientBuilt := ""
		clientOsArch := ""

		serverVersion := ""
		serverApiVersion := ""
		serverGoVersion := ""
		serverGitCommit := ""
		serverBuilt := ""
		serverOsArch := ""

		clientInfo := true
		for _, line := range lines {
			if strings.Contains(line, "Server") {
				clientInfo = false
			}
			if clientInfo {
				if strings.Contains(line, "Version") {
					sep := strings.Split(line, ":")
					if len(sep) == 2 {
						clientVersion = strings.Trim(sep[1], " ")
						clientVersion = strings.Trim(clientVersion, "\t")
						clientVersion = strings.Trim(clientVersion, "\n")
						break
					}
				}
				if strings.Contains(line, "API version") {
					sep := strings.Split(line, ":")
					if len(sep) == 2 {
						clientApiVersion = strings.Trim(sep[1], " ")
						clientApiVersion = strings.Trim(clientApiVersion, "\t")
						clientApiVersion = strings.Trim(clientApiVersion, "\n")
					}
				}
				if strings.Contains(line, "Go version") {
					sep := strings.Split(line, ":")
					if len(sep) == 2 {
						clientGoVersion = strings.Trim(sep[1], " ")
						clientGoVersion = strings.Trim(clientGoVersion, "\t")
						clientGoVersion = strings.Trim(clientGoVersion, "\n")
					}
				}
				if strings.Contains(line, "Git commit") {
					sep := strings.Split(line, ":")
					if len(sep) == 2 {
						clientGitCommit = strings.Trim(sep[1], " ")
						clientGitCommit = strings.Trim(clientGitCommit, "\t")
						clientGitCommit = strings.Trim(clientGitCommit, "\n")
					}
				}
				if strings.Contains(line, "Built") {
					sep := strings.SplitN(line, ":", 2)
					if len(sep) == 2 {
						clientBuilt = strings.Trim(sep[1], " ")
						clientBuilt = strings.Trim(clientBuilt, "\t")
						clientBuilt = strings.Trim(clientBuilt, "\n")
					}
				}
				if strings.Contains(line, "OS/Arch") {
					sep := strings.Split(line, ":")
					if len(sep) == 2 {
						clientOsArch = strings.Trim(sep[1], " ")
						clientOsArch = strings.Trim(clientOsArch, "\t")
						clientOsArch = strings.Trim(clientOsArch, "\n")
					}
				}
			} else {
				if strings.Contains(line, "Version") {
					sep := strings.Split(line, ":")
					if len(sep) == 2 {
						serverVersion = strings.Trim(sep[1], " ")
						serverVersion = strings.Trim(serverVersion, "\t")
						serverVersion = strings.Trim(serverVersion, "\n")
						break
					}
				}
				if strings.Contains(line, "API version") {
					sep := strings.Split(line, ":")
					if len(sep) == 2 {
						serverApiVersion = strings.Trim(sep[1], " ")
						serverApiVersion = strings.Trim(serverApiVersion, "\t")
						serverApiVersion = strings.Trim(serverApiVersion, "\n")
					}
				}
				if strings.Contains(line, "Go version") {
					sep := strings.Split(line, ":")
					if len(sep) == 2 {
						serverGoVersion = strings.Trim(sep[1], " ")
						serverGoVersion = strings.Trim(serverGoVersion, "\t")
						serverGoVersion = strings.Trim(serverGoVersion, "\n")
					}
				}
				if strings.Contains(line, "Git commit") {
					sep := strings.Split(line, ":")
					if len(sep) == 2 {
						serverGitCommit = strings.Trim(sep[1], " ")
						serverGitCommit = strings.Trim(serverGitCommit, "\t")
						serverGitCommit = strings.Trim(serverGitCommit, "\n")
					}
				}
				if strings.Contains(line, "Built") {
					sep := strings.SplitN(line, ":", 2)
					if len(sep) == 2 {
						serverBuilt = strings.Trim(sep[1], " ")
						serverBuilt = strings.Trim(serverBuilt, "\t")
						serverBuilt = strings.Trim(serverBuilt, "\n")
					}
				}
				if strings.Contains(line, "OS/Arch") {
					sep := strings.Split(line, ":")
					if len(sep) == 2 {
						serverOsArch = strings.Trim(sep[1], " ")
						serverOsArch = strings.Trim(serverOsArch, "\t")
						serverOsArch = strings.Trim(serverOsArch, "\n")
					}
				}
			}
		}

		clientBuildInfo := prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "docker_client_version",
				Help: "A metric with a constant '1' value labeled by Version, API version, Go version, git commit, Built, OS/Arch",
			},
			[]string{"Version", "API_version", "Go_version", "Git_commit", "Built", "OS_Arch"},
		)

		ch <- clientBuildInfo.WithLabelValues(clientVersion, clientApiVersion, clientGoVersion, clientGitCommit, clientBuilt, clientOsArch)

		serverBuildInfo := prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "docker_deamon_version",
				Help: "A metric with a constant '1' value labeled by Version, API version, Go version, git commit, Built, OS/Arch",
			},
			[]string{"Version", "API_version", "Go_version", "Git_commit", "Built", "OS_Arch"},
		)
		ch <- serverBuildInfo.WithLabelValues(serverVersion, serverApiVersion, serverGoVersion, serverGitCommit, serverBuilt, serverOsArch)

	} else {
		clientBuildInfo := prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "docker_client_version",
				Help: "A metric with a constant '1' value labeled by Version, API version, Go version, git commit, Built, OS/Arch",
			},
			[]string{"Version", "API_version", "Go_version", "Git_commit", "Built", "OS_Arch"},
		)
		ch <- clientBuildInfo.WithLabelValues("Unknow", "Unknow", "Unknow", "Unknow", "Unknow", "Unknow")

		serverBuildInfo := prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "docker_deamon_version",
				Help: "A metric with a constant '1' value labeled by Version, API version, Go version, git commit, Built, OS/Arch",
			},
			[]string{"Version", "API_version", "Go_version", "Git_commit", "Built", "OS_Arch"},
		)
		ch <- serverBuildInfo.WithLabelValues("Unknow", "Unknow", "Unknow", "Unknow", "Unknow", "Unknow")

	}
}
