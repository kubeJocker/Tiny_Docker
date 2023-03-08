package main

import (
	"TinyDocker/container"
	"encoding/json"
	"fmt"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"os"
	"text/tabwriter"
)

func ListContainers() {
	//找到存储容器信息的路径 /var/run/mydocker
	dirUrl := fmt.Sprintf(container.DefaultInfoLocation, "")
	dirUrl = dirUrl[:len(dirUrl)-1]
	//读取该文件夹下的所有文件
	files, err := ioutil.ReadDir(dirUrl)
	if err != nil {
		log.Errorf("Read dir %s error %v", dirUrl, err)
		return
	}
	var containers []*container.ContainerInfo
	for _, file := range files {
		//将配置文件中的信息转换为容器信息的对象
		tmpContainer, err := getConainterInfo(file)
		if err != nil {
			log.Errorf("Get container info error %v", err)
			continue
		}
		containers = append(containers, tmpContainer)
	}

	//使用控制台打印出容器信息
	w := tabwriter.NewWriter(os.Stdout, 12, 1, 3, ' ', 0)
	fmt.Fprint(w, "ID\tNAME\tPID\tSTATUS\tCOMMAND\tCREATED\n")
	for _, item := range containers {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
			item.Id,
			item.Name,
			item.Pid,
			item.Status,
			item.Command,
			item.CreatedTime)
	}

	//刷新标准输出流缓冲区，将容器列表打印出来
	if err := w.Flush(); err != nil {
		log.Errorf("Flush error %v", err)
		return
	}
}

// 将配置文件中的信息转换为容器信息对象
func getConainterInfo(file os.FileInfo) (*container.ContainerInfo, error) {
	containerName := file.Name()
	configFileDir := fmt.Sprintf(container.DefaultInfoLocation, containerName)
	configFileDir = configFileDir + container.ConfigName
	content, err := ioutil.ReadFile(configFileDir)
	if err != nil {
		log.Errorf("Read file %s error %v", configFileDir, err)
		return nil, err
	}

	var containerInfo container.ContainerInfo
	if err := json.Unmarshal(content, containerInfo); err != nil {
		log.Errorf("JSON unmarshal error %v", err)
	}

	return &containerInfo, nil
}
