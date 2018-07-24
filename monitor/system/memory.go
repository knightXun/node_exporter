package system

import (
	"io/ioutil"
	"regexp"
	"fmt"
	"strconv"
)

var memoryCapacityRegexp = regexp.MustCompile(`MemTotal:\s*([0-9]+) kB`)
var memoryAvaliableRegexp = regexp.MustCompile(`MemAvailable:\s*([0-9]+) kB`)

func GetMemoryUsagePercent() (uint64, error) {
		totalMemory, err := GetMachineMemoryCapacity()
		if err != nil {
			return 0, err
		}
		availableMemory, err := GetMachineMemoryAvailable()
		if err != nil {
			return 0, err
		}
		return availableMemory * 100 / totalMemory, nil
}

func GetMachineMemoryCapacity() (uint64, error) {
	out, err := ioutil.ReadFile("/proc/meminfo")
	if err != nil {
		return 0, err
	}

	memoryCapacity, err := parseCapacity(out, memoryCapacityRegexp)
	if err != nil {
		return 0, err
	}
	return memoryCapacity, err
}

func GetMachineMemoryAvailable() (uint64, error) {
	out, err := ioutil.ReadFile("/proc/meminfo")
	if err != nil {
		return 0, err
	}

	memoryCapacity, err := parseCapacity(out, memoryAvaliableRegexp)
	if err != nil {
		return 0, err
	}
	return memoryCapacity, err
}

func parseCapacity(b []byte, r *regexp.Regexp) (uint64, error) {
	matches := r.FindSubmatch(b)
	if len(matches) != 2 {
		return 0, fmt.Errorf("failed to match regexp in output: %q", string(b))
	}
	m, err := strconv.ParseUint(string(matches[1]), 10, 64)
	if err != nil {
		return 0, err
	}

	// Convert to bytes.
	return m * 1024, err
}

func GetMemoryUsage() (pentage float64, err error) {
	total, err := GetMachineMemoryCapacity()
	if err != nil {
		return
	}

	avail, err := GetMachineMemoryAvailable()
	if err != nil {
		return
	}

	return 1 - float64(avail) / float64(total), nil
}