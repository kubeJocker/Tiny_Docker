package main

import (
	"TinyDocker/container"
	"fmt"
	log "github.com/sirupsen/logrus"
	"os/exec"
)

// 将容器文件系统打包成${imagename}.tar文件
// 使用子目录集合制作镜像
func commitContainer(containerName, imageName string) {
	mntURL := fmt.Sprintf(container.MntUrl, containerName)
	mntURL += "/"

	imageTar := container.RootUrl + "/" + imageName + ".tar"

	if _, err := exec.Command("tar", "-czf", imageTar, "-C", mntURL, ".").CombinedOutput(); err != nil {
		log.Errorf("Tar folder %s error %v", mntURL, err)
	}
}
