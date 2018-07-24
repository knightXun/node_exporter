package main

import (
	"github.com/prometheus/node_exporter/monitor"
	"fmt"
	"os"
)

func main() {
	path := "/home/xufei/workspace/src/github.com/prometheus/node_exporter/monitor/main/monit.json"
	cfg ,err := monitor.EncodeFromFile(path)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	cfg.MainLoop()
	fmt.Println(err.Error())
}
/*
import (
	"github.com/coreos/go-systemd/dbus"
	"fmt"
	dClient "github.com/docker/engine-api/client"
	"github.com/prometheus/node_exporter/monitor/docker"
	"github.com/prometheus/node_exporter/email"
	"syscall"
)

type MonitorConfig struct {
	alertEmails    	[]string
	daemonTime    	int
	emailServer   	string
	emailUserName 	string
	emailPassword 	string
	Process 		[]Process
	Host 			Host
}

type Host struct {
	Cpu 			Cpu
	Memory 			Memory
	FileSys			FileSys
}

type Cpu struct {
	UserTimeLevel int
	SysTimeLevel int
}

type Memory struct {
	TotalUsage 		int
}

type FileSys struct {
	TotalUsage      int
}


type Process struct {
	Type string
	Name string
	Match string
	AlertMessage string
}

var systemdClient *dbus.Conn
var dockerClient *dClient.Client

func init() {
	systemdClient, _ = dbus.NewSystemdConnection()
	dockerClient, _  = docker.NewDockerClient()
}
//type Cache
//
func CheckProcessStatus(p Process, cfg MonitorConfig) (alert bool, msg string, err error) {
	//if p.Type == "systemd" {
		property , err := systemdClient.GetUnitProperty(p.Name, "ActiveState")
		if err != nil || property.Value.String() != "active" {
			if err != nil {
				return true,  "", err
			} else {
				return false, p.Name + " is Not active", err
			}
		}
		property , err = systemdClient.GetUnitProperty(p.Name, "LoadState")
		if err != nil || property.Value.String() != "loaded" {
			if err != nil {
				return true, "",  err
			} else {
				return false, p.Name + " is Not loaded", err
			}
		}

		property , err = systemdClient.GetUnitProperty(p.Name, "SubState")
		if err != nil || property.Value.String() != "running" {
			if err != nil {
				return true, "" ,  err
			} else {
				go func() {
					ch := make(chan string)
					systemdClient.StartUnit(p.Name, "", ch )
					select {
					case <-ch:
						cfg := email.EmailConfig{
							To: cfg.alertEmails,
							Body: "restart " + p.Name,
							EmailServer: cfg.emailServer,
							EmailUserName: cfg.emailUserName,
							EmailPassword: cfg.emailPassword,
						}
						email.SendEmail(cfg)
					}
				}()
				return false, p.Name + " is Not running", err
			}
		}
		return true, "", nil
	//} else if p.Type == "docker" {
	//	dockerClient.ContainerList()
	//}
}

type Filesystem struct {
	Path 			string
	AlertMessage 	string
}

type CpuWarning struct {
	UsageLimit string
}

type MemoryWaring struct {
	UsageLimit string
}

type DiskUsage struct {
	Name 		string
	Size 		uint64
	Used 		uint64
	Avail 		uint64
	MountedPath string
}

func getVfsStats(path string) (total uint64, free uint64, avail uint64, inodes uint64, inodesFree uint64, err error) {
	var s syscall.Statfs_t
	if err = syscall.Statfs(path, &s); err != nil {
		return 0, 0, 0, 0, 0, err
	}
	total = uint64(s.Frsize) * s.Blocks
	free = uint64(s.Frsize) * s.Bfree
	avail = uint64(s.Frsize) * s.Bavail
	inodes = uint64(s.Files)
	inodesFree = uint64(s.Ffree)
	fmt.Println(total, free, avail)
	return total, free, avail, inodes, inodesFree, nil
}

func main() {
	systemd, _ := dbus.NewSystemdConnection()
	po , _ := systemd.GetUnitProperty("docker.service", "ConditionTimestamp")
	fmt.Println(po.Name, po.Value)

	ch := make(chan string)

	s, err := systemd.StartUnit("docker.service", "ignore-dependencies" , ch )
	if err != nil {
		fmt.Println("error : ", err.Error())
	}
	fmt.Println(s)

	res := <- ch
	fmt.Println(res )


	//fmt.Println( time.Unix(0, int64(t)).Format("2006-01-02 15:04:05"))

	//po , _ = systemd.GetUnitProperty("docker.service", "ActiveState")
	//fmt.Println(po.Name, po.Value)
	//po , _ = systemd.GetUnitProperty("docker.service", "LoadState")
	//fmt.Println(po.Name, po.Value)
	//
	//getVfsStats("/var")
	//
	//status ,err := docker.GetContainerStatus("monit-scrapy")
	//if err != nil {
	//	fmt.Println(err.Error())
	//} else {
	//	fmt.Println("monit-scrapy ", status)
	//}
}
*/