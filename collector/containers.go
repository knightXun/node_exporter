package collector

import (
	"encoding/json"
	"fmt"
	"io"
	"time"

	dockerapi "github.com/docker/engine-api/client"
	"github.com/docker/engine-api/types"
	eventtypes "github.com/docker/engine-api/types/events"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/log"
	"golang.org/x/net/context"
	"strings"
	"bufio"
)

func init() {
	registerCollector("containers", defaultDisabled, NewContainersCollector)
}

type containersCollector struct {
	nEventsDesc                *prometheus.Desc
	nContainerDesc             *prometheus.Desc

	containerMetrics map[string]*prometheus.Desc
}

var defaultTimeout = time.Second * 5

func NewContainersCollector() (Collector, error) {
	const subsystem = "containers"

	nEventsDesc := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, subsystem, "event"),
		"container count of Restart triggers", []string{"type", "action", "name", "image", "from"}, nil)

	nContainerDesc := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, subsystem, "containers"),
		"containers", []string{"ID", "Name", "Image", "State", "Status"}, nil)

	containerMetrics := make(map[string]*prometheus.Desc)

	// CPU Stats
	containerMetrics["cpuUsagePercent"] = prometheus.NewDesc(
		prometheus.BuildFQName("container", "cpu", "usage_percent"),
		"CPU usage percent for the specified container",
		[]string{"container_id", "container_name"}, nil,
	)

	// Memory Stats
	containerMetrics["memoryUsagePercent"] = prometheus.NewDesc(
		prometheus.BuildFQName("container", "memory", "usage_percent"),
		"Current memory usage percent for the specified container",
		[]string{"container_id", "container_name"}, nil,
	)
	containerMetrics["memoryUsageBytes"] = prometheus.NewDesc(
		prometheus.BuildFQName("container", "memory", "usage_bytes"),
		"Current memory usage in bytes for the specified container",
		[]string{"container_id", "container_name"}, nil,
	)
	containerMetrics["memoryCacheBytes"] = prometheus.NewDesc(
		prometheus.BuildFQName("container", "memory", "cache_bytes"),
		"Current memory cache in bytes for the specified container",
		[]string{"container_id", "container_name"}, nil,
	)
	containerMetrics["memoryLimit"] = prometheus.NewDesc(
		prometheus.BuildFQName("container", "memory", "limit"),
		"Memory limit as configured for the specified container",
		[]string{"container_id", "container_name"}, nil,
	)

	// Network Stats
	containerMetrics["rxBytes"] = prometheus.NewDesc(
		prometheus.BuildFQName("container", "net_rx", "bytes"),
		"Network RX Bytes",
		[]string{"container_id", "container_name", "interface"}, nil,
	)
	containerMetrics["rxDropped"] = prometheus.NewDesc(
		prometheus.BuildFQName("container", "net_rx", "dropped"),
		"Network RX Dropped Packets",
		[]string{"container_id", "container_name", "interface"}, nil,
	)
	containerMetrics["rxErrors"] = prometheus.NewDesc(
		prometheus.BuildFQName("container", "net_rx", "errors"),
		"Network RX Packet Errors",
		[]string{"container_id", "container_name", "interface"}, nil,
	)
	containerMetrics["rxPackets"] = prometheus.NewDesc(
		prometheus.BuildFQName("container", "net_rx", "packets"),
		"Network RX Packets",
		[]string{"container_id", "container_name", "interface"}, nil,
	)
	containerMetrics["txBytes"] = prometheus.NewDesc(
		prometheus.BuildFQName("container", "net_tx", "bytes"),
		"Network TX Bytes",
		[]string{"container_id", "container_name", "interface"}, nil,
	)
	containerMetrics["txDropped"] = prometheus.NewDesc(
		prometheus.BuildFQName("container", "net_tx", "dropped"),
		"Network TX Dropped Packets",
		[]string{"container_id", "container_name", "interface"}, nil,
	)
	containerMetrics["txErrors"] = prometheus.NewDesc(
		prometheus.BuildFQName("container", "net_tx", "errors"),
		"Network TX Packet Errors",
		[]string{"container_id", "container_name", "interface"}, nil,
	)
	containerMetrics["txPackets"] = prometheus.NewDesc(
		prometheus.BuildFQName("container", "net_tx", "packets"),
		"Network TX Packets",
		[]string{"container_id", "container_name", "interface"}, nil,
	)

	return &containersCollector{
		nEventsDesc:                nEventsDesc,
		nContainerDesc:             nContainerDesc,
		containerMetrics:           containerMetrics,
	}, nil
}

func (c *containersCollector) Update(ch chan<- prometheus.Metric) error {
	events, err := c.getAllEvents()
	if err != nil {
		return fmt.Errorf("couldn't get containers events: %s", err)
	}

	events = filterEvents(events)
	containers, err  := c.getAllContainers()

	metrics, err := c.asyncRetrieveMetrics()

	if err != nil {
		return err
	}

	for _, b := range metrics {
		c.setPrometheusMetrics(b, ch)
	}

	c.collectEventsMetrics(ch, events)
	c.collectContainersMetrics(ch, containers)
	return nil
}

func (c *containersCollector) collectEventsMetrics(ch chan<- prometheus.Metric, events []eventtypes.Message) {
	for _, event := range events {
		log.Debugln(event.From, event.Action, event.Type, event.Time, event.Status)
		if strings.HasPrefix(event.Actor.Attributes["name"], "k8s_") {
			log.Debugln("containerName", event.Actor.Attributes["name"])
			continue
		}

		if event.Type == "container" {
			log.Debugln("send events: ", event.From, event.Action, event.Type, event.Time, event.Status)
			ch <- prometheus.MustNewConstMetric(
				c.nEventsDesc, prometheus.CounterValue,
				1, event.Type, event.Action, event.Actor.Attributes["name"], event.Actor.Attributes["image"], event.From)
		}
	}
}


func (c *containersCollector) collectContainersMetrics(ch chan <- prometheus.Metric, containers []types.Container) {
	for _, container := range containers {
		ch <- prometheus.MustNewConstMetric(
			c.nContainerDesc, prometheus.CounterValue, 1,
			container.ID, container.Names[0], container.Image, container.State, container.Status)
	}
}

func filterEvents(events []eventtypes.Message) []eventtypes.Message {
	filtered := make([]eventtypes.Message, 0, len(events))
	for _, event := range events {
		log.Debugf("Adding unit: %s", event.ID)
		if !strings.HasPrefix(event.Action, "exec") {
			filtered = append(filtered, event)
		}
	}

	return filtered
}

// DecodeEvents decodes event from input stream
func DecodeEvents(input io.Reader) ([]eventtypes.Message, error) {
	dec := json.NewDecoder(input)
	events := []eventtypes.Message{}
	for {
		var event eventtypes.Message
		err := dec.Decode(&event)
		if err != nil && err == io.EOF {
			break
		}

		events = append(events, event)
	}
	return events, nil
}

func (c *containersCollector) getAllEvents() ([]eventtypes.Message, error) {
	client, err := dockerapi.NewEnvClient()
	if err != nil {
		return nil, fmt.Errorf("couldn't get docker connection: %s", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	lastTime := 15 * time.Second

	since := time.Unix(time.Now().Add(-lastTime).Unix(), 0).Format("2006-01-02T15:04:05.999999999Z07:00")
	until := time.Unix(time.Now().Unix(), 0).Format("2006-01-02T15:04:05.999999999Z07:00")
	opts := types.EventsOptions{
		Since: since,
		Until: until,
	}
	response, err := client.Events(ctx, opts)

	if err != nil {
		return nil, err
	}

	events, err := DecodeEvents(response)

	if err != nil {
		return nil, err
	}
	return events, nil
}


func (c *containersCollector) getAllContainers() ([]types.Container, error) {
	client, err := dockerapi.NewEnvClient()
	if err != nil {
		return nil, fmt.Errorf("couldn't get docker connection: %s", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()
	options := types.ContainerListOptions{}
	containers, err := client.ContainerList(ctx, options)
	if err != nil {
		return nil, err
	}
	return containers, nil
}

func (c *containersCollector) setPrometheusMetrics(stats *ContainerMetrics, ch chan<- prometheus.Metric) {
	// Set CPU metrics
	if strings.HasPrefix(stats.Name, "k8s"){
		return
	}

	ch <- prometheus.MustNewConstMetric(c.containerMetrics["cpuUsagePercent"], prometheus.GaugeValue, calcCPUPercent(stats), stats.ID, stats.Name)
	// Set Memory metrics
	ch <- prometheus.MustNewConstMetric(c.containerMetrics["memoryUsagePercent"], prometheus.GaugeValue, calcMemoryPercent(stats), stats.ID, stats.Name)
	ch <- prometheus.MustNewConstMetric(c.containerMetrics["memoryUsageBytes"], prometheus.GaugeValue, float64(stats.MemoryStats.Usage), stats.ID, stats.Name)
	ch <- prometheus.MustNewConstMetric(c.containerMetrics["memoryCacheBytes"], prometheus.GaugeValue, float64(stats.MemoryStats.Stats.Cache), stats.ID, stats.Name)
	ch <- prometheus.MustNewConstMetric(c.containerMetrics["memoryLimit"], prometheus.GaugeValue, float64(stats.MemoryStats.Limit), stats.ID, stats.Name)

	if len(stats.NetIntefaces) == 0 {
		log.Errorf("No network interfaces detected for container %s", stats.Name)
	}
	// Network interface stats (loop through the map of returned interfaces)
	for key, net := range stats.NetIntefaces {
		ch <- prometheus.MustNewConstMetric(c.containerMetrics["rxBytes"], prometheus.GaugeValue, float64(net.RxBytes), stats.ID, stats.Name, key)
		ch <- prometheus.MustNewConstMetric(c.containerMetrics["rxDropped"], prometheus.GaugeValue, float64(net.RxDropped), stats.ID, stats.Name, key)
		ch <- prometheus.MustNewConstMetric(c.containerMetrics["rxErrors"], prometheus.GaugeValue, float64(net.RxErrors), stats.ID, stats.Name, key)
		ch <- prometheus.MustNewConstMetric(c.containerMetrics["rxPackets"], prometheus.GaugeValue, float64(net.RxPackets), stats.ID, stats.Name, key)
		ch <- prometheus.MustNewConstMetric(c.containerMetrics["txBytes"], prometheus.GaugeValue, float64(net.TxBytes), stats.ID, stats.Name, key)
		ch <- prometheus.MustNewConstMetric(c.containerMetrics["txDropped"], prometheus.GaugeValue, float64(net.TxDropped), stats.ID, stats.Name, key)
		ch <- prometheus.MustNewConstMetric(c.containerMetrics["txErrors"], prometheus.GaugeValue, float64(net.TxErrors), stats.ID, stats.Name, key)
		ch <- prometheus.MustNewConstMetric(c.containerMetrics["txPackets"], prometheus.GaugeValue, float64(net.TxPackets), stats.ID, stats.Name, key)
	}
}

func (c *containersCollector) asyncRetrieveMetrics() ([]*ContainerMetrics, error) {
	// Create new docker API client for passed down to the async requests
	cli, err := dockerapi.NewEnvClient()
	if err != nil {
		log.Errorf("Error creating Docker client %v", err)
		return nil, err
	}
	containers, err := cli.ContainerList(context.Background(), types.ContainerListOptions{All: false})
	if err != nil {
		log.Errorf("Error obtaining container listing: %v", err)
		return nil, err
	}

	// Channels used to enable concurrent requests
	ch := make(chan *ContainerMetrics, len(containers))
	ContainerMetrics := []*ContainerMetrics{}

	// Check that there are indeed containers running we can obtain stats for
	if len(containers) == 0 {
		log.Errorf("No Containers returnedx from Docker socket, error: %v", err)
		return ContainerMetrics, err
	}
	// range through the returned containers to obtain the statistics
	// Done due to there not yet being a '--all' option for the cli.ContainerMetrics function in the engine
	for _, c := range containers {
		go func(cli *dockerapi.Client, id, name string) {
			retrieveContainerMetrics(*cli, id, name, ch)
		}(cli, c.ID, c.Names[0][1:])

	}
	for {
		select {
		case r := <-ch:
			ContainerMetrics = append(ContainerMetrics, r)
			if len(ContainerMetrics) == len(containers) {
				return ContainerMetrics, nil
			}
		}
	}
}

// ContainerMetrics is used to track the core JSON response from the stats API
type ContainerMetrics struct {
	ID           string
	Name         string
	NetIntefaces map[string]struct {
		RxBytes   int `json:"rx_bytes"`
		RxDropped int `json:"rx_dropped"`
		RxErrors  int `json:"rx_errors"`
		RxPackets int `json:"rx_packets"`
		TxBytes   int `json:"tx_bytes"`
		TxDropped int `json:"tx_dropped"`
		TxErrors  int `json:"tx_errors"`
		TxPackets int `json:"tx_packets"`
	} `json:"networks"`
	MemoryStats struct {
		Usage int `json:"usage"`
		Limit int `json:"limit"`
		Stats struct {
			Cache int `json:"cache"`
		} `json:"stats"`
	} `json:"memory_stats"`
	CPUStats struct {
		CPUUsage struct {
			PercpuUsage       []int `json:"percpu_usage"`
			UsageInUsermode   int   `json:"usage_in_usermode"`
			TotalUsage        int   `json:"total_usage"`
			UsageInKernelmode int   `json:"usage_in_kernelmode"`
		} `json:"cpu_usage"`
		SystemCPUUsage int64 `json:"system_cpu_usage"`
	} `json:"cpu_stats"`
	PrecpuStats struct {
		CPUUsage struct {
			PercpuUsage       []int `json:"percpu_usage"`
			UsageInUsermode   int   `json:"usage_in_usermode"`
			TotalUsage        int   `json:"total_usage"`
			UsageInKernelmode int   `json:"usage_in_kernelmode"`
		} `json:"cpu_usage"`
		SystemCPUUsage int64 `json:"system_cpu_usage"`
	} `json:"precpu_stats"`
}

func retrieveContainerMetrics(cli dockerapi.Client, id, name string, ch chan<- *ContainerMetrics) {

	stats, err := cli.ContainerStats(context.Background(), id, false)
	if err != nil {
		log.Errorf("Error obtaining container stats for %s, error: %v", id, err)
		return
	}

	s := bufio.NewScanner(stats)

	for s.Scan() {

		var c *ContainerMetrics

		err := json.Unmarshal(s.Bytes(), &c)
		if err != nil {
			log.Errorf("Could not unmarshal the response from the docker engine for container %s. Error: %v", id, err)
			continue
		}

		// Set the container name and ID fields of the ContainerMetrics struct
		// so we can correctly report on the container when looping through later
		c.ID = id
		c.Name = name

		ch <- c
	}

	if s.Err() != nil {
		log.Errorf("Error handling Stats.body from Docker engine. Error: %v", s.Err())
		return
	}
}


func calcCPUPercent(stats *ContainerMetrics) float64 {

	var CPUPercent float64

	cpuDelta := float64(stats.CPUStats.CPUUsage.TotalUsage - stats.PrecpuStats.CPUUsage.TotalUsage)
	sysDelta := float64(stats.CPUStats.SystemCPUUsage - stats.PrecpuStats.SystemCPUUsage)

	if sysDelta > 0.0 && cpuDelta > 0.0 {
		CPUPercent = (cpuDelta / sysDelta) * float64(len(stats.CPUStats.CPUUsage.PercpuUsage)) * 100.0
	}

	return CPUPercent
}

func calcMemoryPercent(stats *ContainerMetrics) float64 {
	return float64(stats.MemoryStats.Usage) * 100.0 / float64(stats.MemoryStats.Limit)
}