package system

import (
	"github.com/opencontainers/runc/libcontainer/cgroups"
	cgroupfs "github.com/opencontainers/runc/libcontainer/cgroups/fs"
	libcontainerconfigs "github.com/opencontainers/runc/libcontainer/configs"
	"fmt"
)

var cgroupManager cgroups.Manager
var preCupTotalTime uint64 = 0
var preCpuUserTime uint64 = 0
var preCpuSystemTime uint64 = 0

var CpuUsagePercent float32 = 0.0
var CpuUserPercent float32 = 0.0
var CpuSystemPercent float32 = 0.0

// Cgroup subsystems we support listing (should be the minimal set we need stats from).
var supportedSubsystems map[string]struct{} = map[string]struct{}{
	"cpu":     {},
	"cpuacct": {},
	"memory":  {},
	"cpuset":  {},
	"blkio":   {},
}

type CgroupSubsystems struct {
	// Cgroup subsystem mounts.
	// e.g.: "/sys/fs/cgroup/cpu" -> ["cpu", "cpuacct"]
	Mounts []cgroups.Mount

	// Cgroup subsystem to their mount location.
	// e.g.: "cpu" -> "/sys/fs/cgroup/cpu"
	MountPoints map[string]string
}

func init() {
	mountPoints, err := GetCgroupSubsystems()
	if err != nil {
		cgroupManager = nil
	}
	cgroupPaths := make(map[string]string, len(mountPoints.MountPoints))
	cgroupManager = &cgroupfs.Manager{
		Cgroups: &libcontainerconfigs.Cgroup{
			Name: "/",
		},
		Paths: cgroupPaths,
	}
}

// Get information about the cgroup subsystems.
func GetCgroupSubsystems() (CgroupSubsystems, error) {
	// Get all cgroup mounts.
	allCgroups, err := cgroups.GetCgroupMounts(true)
	if err != nil {
		return CgroupSubsystems{}, err
	}
	if len(allCgroups) == 0 {
		return CgroupSubsystems{}, fmt.Errorf("failed to find cgroup mounts")
	}

	// Trim the mounts to only the subsystems we care about.
	supportedCgroups := make([]cgroups.Mount, 0, len(allCgroups))
	mountPoints := make(map[string]string, len(allCgroups))
	for _, mount := range allCgroups {
		for _, subsystem := range mount.Subsystems {
			if _, ok := supportedSubsystems[subsystem]; ok {
				supportedCgroups = append(supportedCgroups, mount)
				mountPoints[subsystem] = mount.Mountpoint
			}
		}
	}

	return CgroupSubsystems{
		Mounts:      supportedCgroups,
		MountPoints: mountPoints,
	}, nil
}

func UpdateCpuUsage() error {
	stats, err := cgroupManager.GetStats()
	if err != nil {
		CpuUsagePercent = 0
		return err
	}
	if preCupTotalTime == 0 {
		preCupTotalTime = stats.CpuStats.CpuUsage.TotalUsage
		preCpuUserTime = stats.CpuStats.CpuUsage.UsageInUsermode
		preCpuSystemTime = stats.CpuStats.CpuUsage.UsageInKernelmode
	}

	stats, err = cgroupManager.GetStats()
	if err != nil {
		CpuUsagePercent = 0
		return err
	}
	tmp := (stats.CpuStats.CpuUsage.TotalUsage - preCupTotalTime) / (1 * 1000 * 1000 * 1000) * 100
	CpuUsagePercent = float32(tmp) / 100.0
	tmp = (stats.CpuStats.CpuUsage.UsageInUsermode - preCpuUserTime) / (1 * 1000 * 1000 * 1000) * 100
	CpuUserPercent = float32(tmp) / 100.0
	tmp = (stats.CpuStats.CpuUsage.UsageInKernelmode - preCpuSystemTime) / (1 * 1000 * 1000 * 1000) * 100
	CpuSystemPercent = float32(tmp) / 100.0

	return nil
}