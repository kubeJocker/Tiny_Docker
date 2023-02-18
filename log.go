package main

import (
	"TinyDocker/container"
	"fmt"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"os"
)

func logContainer(containerName string) {
	dirUrl := fmt.Sprintf(container.DefaultInfoLocation, containerName)
	logFileLocation := dirUrl + container.ContainerLogFile
	file, err := os.Open(logFileLocation)
	defer file.Close()
	if err != nil {
		log.Errorf("Log contaienr open file %s error %v", logFileLocation, err)
		return
	}
	content, err := ioutil.ReadAll(file)
	if err != nil {
		log.Errorf("Log contaienr read file %s error %v", logFileLocation, err)
		return
	}
	fmt.Fprintf(os.Stdout, string(content))
}
