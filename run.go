package main

import (
	"TinyDocker/cgroup"
	subsystems "TinyDocker/cgroup/subsystem"
	"TinyDocker/container"
	log "github.com/sirupsen/logrus"
	"os"
	"strings"
)

func Run(tty bool, comArray []string, res *subsystems.ResourceConfig) {
	parent := container.NewParentProcess(tty)
	if parent == nil {
		log.Errorf("New parent process error")
		return
	}
	if err := parent.Start(); err != nil {
		log.Error(err)
	}

	cgroupManager := cgroup.NewCgroupManager("tinydocker-cgroup")
	defer cgroupManager.Destroy()
	cgroupManager.Set(res)
	cgroupManager.Apply(parent.Process.Pid)
	sendInitCommand(comArray, writePipe)
	parent.Wait()
}

func sendInitCommand(comArray []string, writePipe *os.File) {
	command := strings.Join(comArray, " ")
	log.Infof("command all is %s", command)
	writePipe.WriteString(command)
	writePipe.Close()
}
