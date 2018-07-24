//   Copyright 2016 DigitalOcean
//
//   Licensed under the Apache License, Version 2.0 (the "License");
//   you may not use this file except in compliance with the License.
//   You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
//   Unless required by applicable law or agreed to in writing, software
//   distributed under the License is distributed on an "AS IS" BASIS,
//   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//   See the License for the specific language governing permissions and
//   limitations under the License.

package collector

import (
	"encoding/json"
	"log"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/ceph/go-ceph/rados"
	"github.com/prometheus/node_exporter/utils"
	"fmt"
	"gopkg.in/alecthomas/kingpin.v2"
)

const (
	cephNamespace = "ceph"
)

// A ClusterUsageCollector is used to gather all the global stats about a given
// ceph cluster. It is sometimes essential to know how fast the cluster is growing
// or shrinking as a whole in order to zero in on the cause. The pool specific
// stats are provided separately.
type ClusterUsageCollector struct {
	conn Conn

	// GlobalCapacity displays the total storage capacity of the cluster. This
	// information is based on the actual no. of objects that are allocated. It
	// does not take overcommitment into consideration.
	GlobalCapacity prometheus.Gauge

	// UsedCapacity shows the storage under use.
	UsedCapacity prometheus.Gauge

	// AvailableCapacity shows the remaining capacity of the cluster that is left unallocated.
	AvailableCapacity prometheus.Gauge

	// Objects show the total no. of RADOS objects that are currently allocated.
	Objects prometheus.Gauge
}

func init() {
	enable_ceph := kingpin.Flag("enable-ceph", "enable ceph collector").Bool()
	kingpin.Parse()
	if *enable_ceph {
		registerCollector("ceph_cluster_usage", defaultEnabled, NewClusterUsageCollector)
	}
}

// NewClusterUsageCollector creates and returns the reference to ClusterUsageCollector
// and internally defines each metric that display cluster stats.
func NewClusterUsageCollector() (Collector,error) {
	log.Println("init Cluster Usage Collector")
	exporterConfig := "/etc/ceph/exporter.yml"
	cfg, err := utils.ParseConfig(exporterConfig)
	if err != nil {
		log.Fatalf("Error: %v", err)
		return nil, err
	}

	if len(cfg.Cluster) != 1 {
		log.Fatalln("parse /etc/ceph/exporter.yml error: more than one ceph cluster")
		return nil, fmt.Errorf("parse /etc/ceph/exporter.yml error: more than one ceph cluster")
	}

	cluster := cfg.Cluster[0]
	conn, err := rados.NewConnWithUser(cluster.User)
	if err != nil {
		log.Fatalf("cannot create new ceph connection: %s", err)
		return nil, fmt.Errorf("cannot create new ceph connection: %s", err)
	}

	err = conn.ReadConfigFile(cluster.ConfigFile)
	if err != nil {
		log.Fatalf("cannot read ceph config file: %s", err)
		return nil, fmt.Errorf("cannot read ceph config file: %s", err)
	}

	if err := conn.Connect(); err != nil {
		log.Fatalf("cannot connect to ceph cluster: %s", err)
		return nil, fmt.Errorf("cannot connect to ceph cluster: %s", err)
	}
	// defer Shutdown to program exit

	log.Println("Starting ceph exporter for cluster: %s", cluster.ClusterLabel)

	labels := make(prometheus.Labels)
	labels["cluster"] = cluster.ClusterLabel

	collector := ClusterUsageCollector{
		conn: conn,

		GlobalCapacity: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace:   cephNamespace,
			Name:        "cluster_capacity_bytes",
			Help:        "Total capacity of the cluster",
			ConstLabels: labels,
		}),
		UsedCapacity: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace:   cephNamespace,
			Name:        "cluster_used_bytes",
			Help:        "Capacity of the cluster currently in use",
			ConstLabels: labels,
		}),
		AvailableCapacity: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace:   cephNamespace,
			Name:        "cluster_available_bytes",
			Help:        "Available space within the cluster",
			ConstLabels: labels,
		}),
		Objects: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace:   cephNamespace,
			Name:        "cluster_objects",
			Help:        "No. of rados objects within the cluster",
			ConstLabels: labels,
		}),
	}
	return &collector, nil
}

func (c *ClusterUsageCollector) metricsList() []prometheus.Metric {
	return []prometheus.Metric{
		c.GlobalCapacity,
		c.UsedCapacity,
		c.AvailableCapacity,
		c.Objects,
	}
}

type cephClusterStats struct {
	Stats struct {
		TotalBytes      json.Number `json:"total_bytes"`
		TotalUsedBytes  json.Number `json:"total_used_bytes"`
		TotalAvailBytes json.Number `json:"total_avail_bytes"`
		TotalObjects    json.Number `json:"total_objects"`
	} `json:"stats"`
}

func (c *ClusterUsageCollector) collect() error {
	cmd := c.cephUsageCommand()
	buf, _, err := c.conn.MonCommand(cmd)
	if err != nil {
		return err
	}

	stats := &cephClusterStats{}
	if err := json.Unmarshal(buf, stats); err != nil {
		return err
	}

	var totBytes, usedBytes, availBytes, totObjects float64

	totBytes, err = stats.Stats.TotalBytes.Float64()
	if err != nil {
		log.Println("[ERROR] cannot extract total bytes:", err)
	}

	usedBytes, err = stats.Stats.TotalUsedBytes.Float64()
	if err != nil {
		log.Println("[ERROR] cannot extract used bytes:", err)
	}

	availBytes, err = stats.Stats.TotalAvailBytes.Float64()
	if err != nil {
		log.Println("[ERROR] cannot extract available bytes:", err)
	}

	totObjects, err = stats.Stats.TotalObjects.Float64()
	if err != nil {
		log.Println("[ERROR] cannot extract total objects:", err)
	}

	c.GlobalCapacity.Set(totBytes)
	c.UsedCapacity.Set(usedBytes)
	c.AvailableCapacity.Set(availBytes)
	c.Objects.Set(totObjects)

	return nil
}

func (c *ClusterUsageCollector) cephUsageCommand() []byte {
	cmd, err := json.Marshal(map[string]interface{}{
		"prefix": "df",
		"detail": "detail",
		"format": "json",
	})
	if err != nil {
		// panic! because ideally in no world this hard-coded input
		// should fail.
		panic(err)
	}
	return cmd
}

// Describe sends the descriptors of each metric over to the provided channel.
// The corresponding metric values are sent separately.
func (c *ClusterUsageCollector) Describe(ch chan<- *prometheus.Desc) {
	for _, metric := range c.metricsList() {
		ch <- metric.Desc()
	}
}

// Update sends the metric values for each metric pertaining to the global
// cluster usage over to the provided prometheus Metric channel.
func (c *ClusterUsageCollector) Update(ch chan<- prometheus.Metric) error {
	if err := c.collect(); err != nil {
		log.Println("[ERROR] failed collecting cluster usage metrics:", err)
		return err
	}

	for _, metric := range c.metricsList() {
		ch <- metric
	}
	return nil
}