package monitor

import (
	"time"
	"encoding/json"
	"github.com/prometheus/node_exporter/monitor/system"
	"github.com/prometheus/node_exporter/email"
	"github.com/coreos/go-systemd/dbus"
	"github.com/prometheus/node_exporter/monitor/docker"
	"fmt"
	"io/ioutil"
	"net/http"
	"github.com/prometheus/common/log"
)

type MonitConfig struct {
	Host 				string 			`json:"host,omitempty"`
	AlertEmails 		[]string		`json:"alertEmails,omitempty"`
	DaemonTime 			int				`json:"daemonTime,omitempty"`
	EmailServer  		string			`json:"emailServer,omitempty"`
	EmailUserName 		string			`json:"emailUserName,omitempty"`
	EmailPassword 		string			`json:"emailPassword,omitempty"`
	Processes 			[]Process		`json:"processes,omitempty"`
	Cpu 				Cpu				`json:"cpu,omitempty"`
	Memory 				Memory			`json:"memory,omitempty"`
	Files				[]Filesystem	`json:"files,omitempty"`
}

type Filesystem struct {
	Path 				string			`json:"path,omitempty"`
	Limit          	 	float32			`json:"limit,omitempty"`
	Warning 			bool
}

type Cpu struct {
	Limit 				float32			`json:"limit,omitempty"`
	Warning 			bool
}

type Memory struct {
	Limit 				float32			`json:"limit,omitempty"`
	Warning 			bool
}

type Process struct {
	Type 				string			`json:"type,omitempty"`
	Name 				string			`json:"name,omitempty"`
	Match 				string			`json:"match,omitempty"`
	RestartCmd   		string			`json:"restartCmd,omitempty"`
	Status          	string			`json:"status,omitempty"`
	Warning				bool			//`json:"warning,omitempty"`
	AlertMsg			string 			//`json:"alertMsg,omitempty"`
}

func EncodeFromFile(path string) (cfg MonitConfig, err error) {
	bts, err := ioutil.ReadFile(path)
	if err != nil {
		return
	}
	err = json.Unmarshal(bts, &cfg)
	if err != nil {
		return
	}
	return cfg, nil
}

func (cfg *MonitConfig) Status(w http.ResponseWriter, r *http.Request) {
	msg , err := json.Marshal(cfg)
	if err != nil {
		log.Errorln(err.Error())
		w.Header().Add("ErrMsg", string(msg))
		w.WriteHeader(400)
		return
	}
	w.Write(msg)
	w.WriteHeader(200)
	return
}

func (config *MonitConfig) MainLoop() error {
	ticker := time.NewTicker(time.Second * time.Duration(config.DaemonTime))
	systemdClient, _ := dbus.NewSystemdConnection()
	docker.NewDockerClient()

	emailConfg := email.EmailConfig{
		To: config.AlertEmails,
		EmailPassword: config.EmailPassword,
		EmailUserName: config.EmailUserName,
		EmailServer:   config.EmailServer,
		Subject:       "node_export warnning",
	}

	for {
		select {
		case <- ticker.C:
			go func() {
				log.Infoln("begin monitor")
				emailBody := ""
				for index, process := range config.Processes {
					log.Infoln("Check Process ", process.Name, process.Type)
					if process.Type == "systemd" {
						status, err := system.GetSystemdProcess(process.Name, systemdClient)
						if err != nil {
							if !config.Processes[index].Warning {
								emailBody += fmt.Sprintf("%s Status Unknown ;", process.Name)
								emailBody += "\r\n"
								config.Processes[index].Warning = true
							}
							config.Processes[index].Status = "Unknown"
							log.Errorf("systemd meet error: %v", err)
						}
						config.Processes[index].Status = status
						if status != "running" {
							go system.RestartSystemdProcess(process.Name, systemdClient)
							if !config.Processes[index].Warning  {
								emailBody += fmt.Sprintf("%s Not Running   ;", process.Name)
								emailBody += "\r\n"
								config.Processes[index].Warning  = true
							}
							config.Processes[index].Status = "NotRunning"
						} else {
							config.Processes[index].Warning = false
							config.Processes[index].Status = "Running"
						}
					} else if process.Type == "docker" {
						containerStatus, err := docker.GetContainerStatus(process.Name)
						if err != nil {
							if !process.Warning {
								emailBody += fmt.Sprintf("Container %s status unknown %s     ;", process.Name, err.Error())
								emailBody += "\r\n"
								config.Processes[index].Warning  = true
							}
							config.Processes[index].Status = "Unknown"
							log.Errorln("Container %s status unknown %s", process.Name, err)
						} else if containerStatus.Status != "running" {
							config.Processes[index].Status = containerStatus.Status
							if !process.Warning {
								emailBody += fmt.Sprintf("docker %s not running     ;", process.Name)
								emailBody += "\r\n"
								config.Processes[index].Warning  = true
							}
							config.Processes[index].Status = "NotRunning"
						} else {
							config.Processes[index].Status = containerStatus.Status
							config.Processes[index].Warning  = false
							config.Processes[index].Status = "Running"
						}
					}
				}

				log.Infoln("Check Memory")
				memoryUsage, err := system.GetMemoryUsage()
				if err != nil {
					if ! config.Memory.Warning {
						emailBody += fmt.Sprintf("MemoryUsage Unknown   ;")
						emailBody += "\r\n"
						config.Memory.Warning = true
					}
				} else if float32(memoryUsage) >= config.Memory.Limit {
					if !config.Memory.Warning {
						config.Memory.Warning = true
						emailBody += fmt.Sprintf("MemoryUsage Exceed     ;", config.Memory.Limit)
						emailBody += "\r\n"
					}
				} else {
					config.Memory.Warning = false
				}

				log.Infoln("Check Filesystem")
				for index, file := range config.Files {
					log.Infoln("Check Path " + file.Path)
					diskStats, err := system.UpdateFsStats(file.Path)
					if err != nil {
						if !config.Files[index].Warning {
							log.Errorf("Get Path %s usage meet error: %v", file.Path, err)
							emailBody += fmt.Sprintf("%s Unknown     ;", config.Files[index].Path)
							emailBody += "\r\n"
							config.Files[index].Warning = true
						}
						continue
					}
					perc := 1 - float64(diskStats.Avail) / float64(diskStats.Size)
					if perc >= float64(file.Limit) {
						if !config.Files[index].Warning {
							emailBody += fmt.Sprintf("Path %s Usage Exceed %f    ;", file.Path, file.Limit)
							emailBody += "\r\n"
							config.Files[index].Warning = true
						}
					} else {
						config.Files[index].Warning = false
					}
				}

				log.Infoln("Check Cpu")
				system.UpdateCpuUsage()
				if system.CpuUsagePercent >= config.Cpu.Limit {
					if !config.Cpu.Warning {
						emailBody += fmt.Sprintf("CpuUsage Exceed %s     ;", config.Cpu.Limit)
						emailBody += "\r\n"
						config.Cpu.Warning = true
					}
				} else {
					config.Cpu.Warning = false
				}

				log.Infoln("EmailBody is: ", emailBody)
				if emailBody != "" {
					emailConfg.Subject = config.Host + " warning"
					emailConfg.Body = emailBody
					log.Infoln(emailBody)
					email.SendEmail(emailConfg)
					fmt.Println(*config)
				}
			}()
		}
	}
}
