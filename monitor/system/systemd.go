package system

import (
	"github.com/coreos/go-systemd/dbus"
	"github.com/prometheus/common/log"
)

func GetSystemdProcess(name string, systemdClient *dbus.Conn) (status string, err error){
	p, err := systemdClient.GetUnitProperties(name)
	if err != nil {
		return "unknown", err
	}
	if p["SubState"] != "running" {
		return "Existed", nil
	} else {
		return "running",nil
	}
}

func RestartSystemdProcess(name string, systemdClient *dbus.Conn) {
	ch := make(chan string)
	_, err := systemdClient.StartUnit(name, "ignore-dependencies", ch )
	if err != nil {
		log.Errorf("restart %v failed: %v", name, err)
	}

	res := <- ch
	if res != "done" {
		log.Infof("restart systemd %v finish", name)
	} else {
		log.Errorf("restart %v failed", name)
	}
}