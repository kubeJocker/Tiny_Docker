package network

import (
	"encoding/json"
	log "github.com/sirupsen/logrus"
	"net"
	"os"
	"path"
	"strings"
)

const ipamDefaultAllocatorPath = "/var/run/mydocker/network/ipam/subnet.json"

type IPAM struct {
	//分配文件存放地址
	SubnetAllocatorPath string
	//网段和位图算法的数组，key为网段，value为位图
	Subnets *map[string]string
}

// 默认使用/var/run/mydocker/network/ipam/subnet.json作为分配信息存储地址
var ipAllocator = &IPAM{
	SubnetAllocatorPath: ipamDefaultAllocatorPath,
}

// 加载网络地址分配信息
func (ipam *IPAM) load() error {
	if _, err := os.Stat(ipam.SubnetAllocatorPath); err != nil {
		if os.IsNotExist(err) {
			return nil
		} else {
			return err
		}
	}
	//打开并读取存储文件
	subnetConfigFile, err := os.Open(ipam.SubnetAllocatorPath)
	defer subnetConfigFile.Close()
	if err != nil {
		return err
	}
	subnetConfigFile, err = os.Open(ipam.SubnetAllocatorPath)
	defer subnetConfigFile.Close()
	if err != nil {
		return err
	}
	subnetJson := make([]byte, 2000)
	n, err := subnetConfigFile.Read(subnetJson)
	if err != nil {
		return err
	}
	//反序列化出IP分配信息
	err = json.Unmarshal(subnetJson[:n], ipam.Subnets)
	if err != nil {
		log.Errorf("Error dump allocation info, %v", err)
		return err
	}
	return nil
}

// 存储网段地址分配信息
func (ipam *IPAM) dump() error {
	ipamConfigFileDir, _ := path.Split(ipam.SubnetAllocatorPath)
	if _, err := os.Stat(ipamConfigFileDir); err != nil {
		if os.IsNotExist(err) {
			os.MkdirAll(ipamConfigFileDir, 0644)
		} else {
			return err
		}
	}
	subnetConfigFile, err := os.OpenFile(ipam.SubnetAllocatorPath, os.O_TRUNC|os.O_WRONLY|os.O_CREATE, 0644)
	defer subnetConfigFile.Close()
	if err != nil {
		return err
	}
	ipamConfigJson, err := json.Marshal(ipam.Subnets)
	if err != nil {
		return err
	}
	_, err = subnetConfigFile.Write(ipamConfigJson)
	if err != nil {
		return err
	}
	return nil
}

// 在网段中分配一个可用的IP地址，并记录
func (ipam *IPAM) Allocate(subnet *net.IPNet) (ip net.IP, err error) {
	ipam.Subnets = &map[string]string{}
	//从文件加载已经分配的网段信息
	err = ipam.load()
	if err != nil {
		log.Errorf("Error load allocation info, %v", err)
	}
	//返回网段的子网掩码总长度和网段前面固定位的长度
	one, size := subnet.Mask.Size()
	//如果没有分配过这个网段，则初始化网段的分配配置
	if _, exist := (*ipam.Subnets)[subnet.String()]; !exist {
		//用0填满可用地址
		(*ipam.Subnets)[subnet.String()] = strings.Repeat("0", 1<<uint8(size-one))

	}
	//遍历网段的位图数组
	for c := range (*ipam.Subnets)[subnet.String()] {
		//找到数组中为0的序号，分配这个IP
		if (*ipam.Subnets)[subnet.String()][c] == '0' {
			ipalloc := []byte((*ipam.Subnets)[subnet.String()])
			ipalloc[c] = '1'
			(*ipam.Subnets)[subnet.String()] = string(ipalloc)
			//初始IP
			ip = subnet.IP
			//通过网段IP与偏移相加计算分配的IP地址
			//在初始的IP上依次相加[uint8(c >> 24)、uint8(c >> 16)、uint8(c >> 8)、uint8(c >> 0)]
			for t := uint(4); t > 0; t-- {
				[]byte(ip)[4-t] += uint8((c >> ((t - 1) * 8)))
			}
			ip[3] += 1
			break
		}
	}
	ipam.dump()
	return
}

// 释放地址
func (ipam *IPAM) Release(subnet *net.IPNet, ipaddr *net.IP) error {
	ipam.Subnets = &map[string]string{}
	err := ipam.load()
	if err != nil {
		log.Errorf("Error dump allocacation info, %v", err)
	}
	c := 0
	releaseIP := ipaddr.To4()
	releaseIP[3]--
	for t := uint(4); t > 0; t-- {
		c += int(releaseIP[t-1]-subnet.IP[t-1]) << ((4 - t) * 8)
	}

	ipalloc := []byte((*ipam.Subnets)[subnet.String()])
	ipalloc[c] = '0'
	(*ipam.Subnets)[subnet.String()] = string(ipalloc)
	ipam.dump()
	return nil
}
