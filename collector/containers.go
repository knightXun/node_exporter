package collector

import (
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"time"

	dockerapi "github.com/docker/engine-api/client"
	"github.com/docker/engine-api/types"
	eventtypes "github.com/docker/engine-api/types/events"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/log"
	"golang.org/x/net/context"
	"gopkg.in/alecthomas/kingpin.v2"
	"strings"
)

var (
	containersWhitelist = kingpin.Flag("collector.containers.whitelist", "Regexp of docker containers to whitelist. Containers must both match whitelist and not match blacklist to be included.").Default(".+").String()
	containersBlacklist = kingpin.Flag("collector.containers.blacklist", "Regexp of docker containers units to blacklist. Containers must both match whitelist and not match blacklist to be included.").Default(".+\\.scope").String()
)

func init() {
	registerCollector("containers", defaultDisabled, NewContainersCollector)
}

type containersCollector struct {
	nEventsDesc                *prometheus.Desc
	containersWhitelistPattern *regexp.Regexp
	containersBlacklistPattern *regexp.Regexp
}

var defaultTimeout = time.Second * 5

func NewContainersCollector() (Collector, error) {
	const subsystem = "containers"

	nEventsDesc := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, subsystem, "host_docker_event"),
		"container count of Restart triggers", []string{"type", "action", "name", "image", "from"}, nil)

	containersWhitelistPattern := regexp.MustCompile(fmt.Sprintf("^(?:%s)$", *unitWhitelist))
	containersBlacklistPattern := regexp.MustCompile(fmt.Sprintf("^(?:%s)$", *unitBlacklist))

	return &containersCollector{
		nEventsDesc:                nEventsDesc,
		containersWhitelistPattern: containersWhitelistPattern,
		containersBlacklistPattern: containersBlacklistPattern,
	}, nil
}

func (c *containersCollector) Update(ch chan<- prometheus.Metric) error {
	events, err := c.getAllEvents()
	if err != nil {
		return fmt.Errorf("couldn't get containers events: %s", err)
	}

	events = filterEvents(events, c.containersWhitelistPattern, c.containersBlacklistPattern)

	c.collectEventsMetrics(ch, events)
	//containers, err := c.getAllContainers()
	//if err != nil {
	//	return fmt.Errorf("couldn't get containers: %s", err)
	//}
	//containers := filterContainers(containers, c.containersWhitelistPattern, c.containersBlacklistPattern)
	return nil
}

func (c *containersCollector) collectEventsMetrics(ch chan<- prometheus.Metric, events []eventtypes.Message) {
	for _, event := range events {
		if strings.HasPrefix(event.Actor.Attributes["name"], "k8s_") {
			fmt.Println("containerName", event.Actor.Attributes["name"])
			continue
		}

		if event.Type == "container" {
			ch <- prometheus.MustNewConstMetric(
				c.nEventsDesc, prometheus.CounterValue,
				1, event.Type, event.Action, event.Actor.Attributes["name"], event.Actor.Attributes["image"], event.From)
		}
	}
}

func filterEvents(events []eventtypes.Message, whitelistPattern, blacklistPattern *regexp.Regexp) []eventtypes.Message {
	filtered := make([]eventtypes.Message, 0, len(events))
	for _, event := range events {
		if whitelistPattern.MatchString(event.Actor.Attributes["image"]) &&
			!blacklistPattern.MatchString(event.Actor.Attributes["image"]) {
			log.Debugf("Adding unit: %s", event.ID)
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
			log.Debugf("Adding unit: %s", c.ID)
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

func (c *containersCollector) getAllEvents() ([]eventtypes.Message, error) {
	client, err := dockerapi.NewEnvClient()
	if err != nil {
		return nil, fmt.Errorf("couldn't get docker connection: %s", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	lastTime := 15 * time.Second

	opts := types.EventsOptions{
		Since: time.Now().Add(-lastTime).Format("2018-03-00T10:01:01"),
		Until: time.Now().Format("2018-03-00T10:01:01"),
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
