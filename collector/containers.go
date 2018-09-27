package collector

import (
	"fmt"
	"time"
	"encoding/json"
	"io"
	"regexp"

	"gopkg.in/alecthomas/kingpin.v2"
	dockerapi "github.com/docker/engine-api/client"
	eventtypes "github.com/docker/engine-api/types/events"
	"github.com/docker/engine-api/types"
	"golang.org/x/net/context"
	"github.com/prometheus/common/log"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	containersWhitelist  = kingpin.Flag("collector.containers.whitelist", "Regexp of docker containers to whitelist. Containers must both match whitelist and not match blacklist to be included.").Default(".+").String()
	containersBlacklist  = kingpin.Flag("collector.containers.blacklist", "Regexp of docker containers units to blacklist. Containers must both match whitelist and not match blacklist to be included.").Default(".+\\.scope").String()
)

func init() {
	registerCollector("containers", defaultDisabled, NewContainersCollector)
}

type containersCollector struct {
	nRestartsDesc                 		*prometheus.Desc
	nStopDesc 							*prometheus.Desc
	nStartDesc                          *prometheus.Desc
	containersWhitelistPattern          *regexp.Regexp
	containersBlacklistPattern          *regexp.Regexp
}

var defaultTimeout = time.Second * 5

func NewContainersCollector() (Collector, error) {
	const subsystem = "containers"

	nRestartsDesc := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, subsystem, "container_restart_total"),
		"container count of Restart triggers", []string{"state"}, nil)

	nStopDesc := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, subsystem, "container_stop_total"),
		"container count of Restart triggers", []string{"state"}, nil)

	nStartDesc := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, subsystem, "container_start_total"),
		"container count of Restart triggers", []string{"state"}, nil)

	containersWhitelistPattern := regexp.MustCompile(fmt.Sprintf("^(?:%s)$", *unitWhitelist))
	containersBlacklistPattern := regexp.MustCompile(fmt.Sprintf("^(?:%s)$", *unitBlacklist))

	return &containersCollector{
		nRestartsDesc:                 	nRestartsDesc,
		nStopDesc:						nStopDesc,
		nStartDesc:						nStartDesc,
		containersWhitelistPattern:     containersWhitelistPattern,
		containersBlacklistPattern:     containersBlacklistPattern,
	}, nil
}

func (c *containersCollector) Update(ch chan<- prometheus.Metric) error {
	events, err := c.getAllEvents()
	if err != nil {
		return fmt.Errorf("couldn't get containers events: %s", err)
	}

	events = filterEvents(events, c.containersWhitelistPattern, c.containersBlacklistPattern)

	c.collectStopMetrics(ch, events)
	c.collectRestartMetrics(ch, events)
	c.collectStartMetrics(ch, events)
	containers, err := c.getAllContainers()
	if err != nil {
		return fmt.Errorf("couldn't get containers: %s", err)
	}

	containers := filterContainers(containers, c.containersWhitelistPattern, c.containersBlacklistPattern)


	return nil
}

func (c *containersCollector) collectStopMetrics(ch chan<- prometheus.Metric, events []eventtypes.Message) {

}

func (c *containersCollector) collectStartMetrics(ch chan<- prometheus.Metric, events []eventtypes.Message) {

}

func (c *containersCollector) collectRestartMetrics(ch chan<- prometheus.Metric, events []eventtypes.Message) {

}

func filterEvents(events []eventtypes.Message, whitelistPattern, blacklistPattern *regexp.Regexp) []eventtypes.Message {
	filtered := make([]eventtypes.Message, 0, len(events))
	for _, event := range events {
		if whitelistPattern.MatchString(event.Actor.Attributes["image"]) &&
		 !blacklistPattern.MatchString(event.Actor.Attributes["image"]) {
			log.Debugf("Adding unit: %s", event.ID )
			filtered = append(filtered, event)
		} else {
			log.Debugf("Ignoring unit: %s", event.ID)
		}
	}

	return filtered
}

func filterContainers(containers []types.Container, whitelistPattern, blacklistPattern *regexp.Regexp) []types.Container {
	filtered := make([]types.Container, 0, len(containers))
	for _, c := range containers {
		if whitelistPattern.MatchString(c.Image) &&
			!blacklistPattern.MatchString(c.Image) {
			log.Debugf("Adding unit: %s", c.ID )
			filtered = append(filtered, c)
		} else {
			log.Debugf("Ignoring unit: %s", c.ID)
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

func (c *containersCollector) getAllEvents() ([]eventtypes.Message, error){
	client, err := dockerapi.NewEnvClient()
	if err != nil {
		return nil, fmt.Errorf("couldn't get docker connection: %s", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	lastTime := 5* time.Second
	opts := types.EventsOptions{
		Since: lastTime.String(),
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