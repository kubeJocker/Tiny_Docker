package network

import (
	"TinyDocker/container"
	"encoding/json"
	"fmt"
	"github.com/sirupsen/logrus"
	"github.com/vishvananda/netlink"
	"github.com/vishvananda/netns"
	"net"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"text/tabwriter"
)

var (
	defaultNetworkPath = "/var/run/mydocker/network/network/"
	drivers            = map[string]NetworkDriver{}
	networks           = map[string]*Network{}
)

type Network struct {
	Name    string
	IpRange *net.IPNet //网段
	Driver  string     //网络驱动名称
}

type Endpoint struct {
	ID          string           `json:id`
	Device      netlink.Veth     `josn:"dev"`
	IPAddress   net.IP           `jsohn:"ip"`
	MacAddress  net.HardwareAddr `josn:"mac"`
	PortMapping []string         `josn:"portmapping"'`
	Network     *Network
}

type NetworkDriver interface {
	Name() string
	Create(subnet string, name string) (*Network, error)
	Delete(network Network) error
	Connect(network *Network, endpoint *Endpoint) error
	Disconnect(network Network, endpoint *Endpoint) error
}

func CreateNetwork(driver, subnet, name string) error {
	//将网段的字符串转换成net.IPNet
	_, cidr, _ := net.ParseCIDR(subnet)
	//通过IPAM分配网关IP，获取到网段中的第一个IP作为网关IP
	ip, err := ipAllocator.Allocate(cidr)
	if err != nil {
		return err
	}
	cidr.IP = ip
	//调用指定的网络驱动创建网络
	//drivers为各个网络驱动的实例字典，通过调用网络驱动的create方法创建网络
	nw, err := drivers[driver].Create(cidr.String(), name)
	if err != nil {
		return err
	}
	//将网络信息保存在文件系统中，以便查询和在网络上连接端点
	return nw.dump(defaultNetworkPath)
}

func (nw *Network) dump(dumpPath string) error {
	if _, err := os.Stat(dumpPath); err != nil {
		if os.IsNotExist(err) {
			os.MkdirAll(dumpPath, 0644)
		} else {
			return err
		}
	}
	nwPath := path.Join(dumpPath, nw.Name)
	nwFile, err := os.OpenFile(nwPath, os.O_TRUNC|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		logrus.Errorf("error:", err)
		return err
	}
	defer nwFile.Close()

	nwJson, err := json.Marshal(nw)
	if err != nil {
		logrus.Errorf("error:", err)
		return err
	}
	_, err = nwFile.Write(nwJson)
	if err != nil {
		logrus.Errorf("error:", err)
		return err
	}
	return nil
}

func (nw *Network) load(dumpPath string) error {
	nwConfigFile, err := os.Open(dumpPath)
	defer nwConfigFile.Close()
	if err != nil {
		return err
	}
	nwJson := make([]byte, 2000)
	n, err := nwConfigFile.Read(nwJson)
	if err != nil {
		return err
	}
	err = json.Unmarshal(nwJson[:n], nw)
	if err != nil {
		logrus.Errorf("Error load nw info", err)
		return err
	}
	return nil
}

func Connect(networkName string, cinfo *container.ContainerInfo) error {
	//从networks字典中取到容器连接的网络信息
	network, ok := networks[networkName]
	if !ok {
		return fmt.Errorf("No Such Network: %s", networkName)
	}
	//通过调用IPAM从网络的网段中获取可用的IP作为容器的IP地址
	ip, err := ipAllocator.Allocate(network.IpRange)
	if err != nil {
		return err
	}
	//创建网络端点
	ep := &Endpoint{
		ID:          fmt.Sprintf("%s-%s", cinfo.Id, networkName),
		Device:      netlink.Veth{},
		IPAddress:   ip,
		MacAddress:  nil,
		PortMapping: cinfo.PortMapping,
		Network:     network,
	}
	//调用网络驱动的Connect方法区连接和配置容器网络设备的IP地址和路由
	if err := drivers[network.Driver].Connect(network, ep); err != nil {
		return err
	}
	//在容器的Namespace中配置容器网络。设备IP和路由信息
	if err = configEndpointIpAddressAndRoute(ep, cinfo); err != nil {
		return err
	}
	//配置i容器到宿主机的端口映射
	return configPortMapping(ep, cinfo)
}

func Init() error {
	//加载网络驱动
	var bridgeDriver = BridgeNetworkDriver{}
	drivers[bridgeDriver.Name()] = &bridgeDriver
	//判断网络的配置根目录是否存在，不存在则创建
	if _, err := os.Stat(defaultNetworkPath); err != nil {
		if os.IsNotExist(err) {
			os.MkdirAll(defaultNetworkPath, 0644)
		} else {
			return err
		}
	}
	//检查网络配置目录中的所有文件
	//filePath.walk函数会遍历指定的path目录，并执行第二个参数中的函数指针去处理目录下的每个文件
	filepath.Walk(defaultNetworkPath, func(nwPath string, info os.FileInfo, err error) error {
		//跳过目录
		if info.IsDir() {
			return nil
		}
		//加载文件名作为网络名
		_, nwName := path.Split(nwPath)
		nw := &Network{
			Name: nwName,
		}

		//调用load加载网络信息
		if err := nw.load(nwPath); err != nil {
			logrus.Errorf("error load network: %s", err)
		}
		//将网络配置信息加入network字典中
		networks[nwName] = nw
		return nil
	})
	return nil
}

func ListNetwork() {
	//使用控制台打印出网络信息
	w := tabwriter.NewWriter(os.Stdout, 12, 1, 3, ' ', 0)
	fmt.Fprintf(w, "NAME\tIpRange\tDriver\n")
	for _, nw := range networks {
		fmt.Fprintf(w, "%s\t%s\t%s\n",
			nw.Name,
			nw.IpRange,
			nw.Driver,
		)
	}
	if err := w.Flush(); err != nil {
		logrus.Errorf("Flush error %v", err)
		return
	}
}

func DeleteNetwork(networkName string) error {
	//查找网络是否存在
	nw, ok := networks[networkName]
	if !ok {
		return fmt.Errorf("No Such Network: %s", networkName)
	}

	//调用IPAM实例ipAllocator释放网络网关IP
	if err := ipAllocator.Release(nw.IpRange, &nw.IpRange.IP); err != nil {
		return fmt.Errorf("Error Remove Network gateway ip: %s", err)
	}

	//调用网络驱动delete删除网络创建的设置与配置
	if err := drivers[nw.Driver].Delete(*nw); err != nil {
		return fmt.Errorf("Error Remove Network DriverError: %s", err)
	}
	//删除配置目录中该网络对应的配置文件
	return nw.remove(defaultNetworkPath)
}

func (nw *Network) remove(dumpPath string) error {
	if _, err := os.Stat(path.Join(dumpPath, nw.Name)); err != nil {
		if os.IsNotExist(err) {
			return nil
		} else {
			return err
		}
	} else {
		return os.Remove(path.Join(dumpPath, nw.Name))
	}
}

func configEndpointIpAddressAndRoute(ep *Endpoint, cinfo *container.ContainerInfo) error {
	//获取网络端点中Veth的另一端
	peerLink, err := netlink.LinkByName(ep.Device.PeerName)
	if err != nil {
		return fmt.Errorf("fail config endpoint: %v", err)
	}
	//将容器的网络端点加入到容器的网络空间中
	//并使这个函数下面的操作都在这个网络空间进行
	//执行函数后，恢复为默认的网络空间
	defer enterContainerNetns(&peerLink, cinfo)()

	//获取容器的IP地址以及网段，用于配置容器内部接口地址
	interfaceIP := *ep.Network.IpRange
	interfaceIP.IP = ep.IPAddress

	//设置容器内Veth的端点IP
	if err = setInterfaceIP(ep.Device.PeerName, interfaceIP.String()); err != nil {
		return fmt.Errorf("%v, %s", ep.Network, err)
	}

	//启动容器内Veth端点
	if err = setInterfaceUP(ep.Device.PeerName); err != nil {
		return err
	}
	//启动net Namespace中默认本地地址127.0.0.1的lo网卡
	if err = setInterfaceUP("lo"); err != nil {
		return err
	}
	//设置容器内的外部请求都通过容器内的Veth端点访问
	_, cidr, _ := net.ParseCIDR("0.0.0.0/0")
	defaultRoute := &netlink.Route{
		LinkIndex: peerLink.Attrs().Index,
		Gw:        ep.Network.IpRange.IP,
		Dst:       cidr,
	}
	//调用RouteAdd，添加路由到网络空间
	if err = netlink.RouteAdd(defaultRoute); err != nil {
		return err
	}
	return nil
}

// 将容器网络端点加入到容器的网络空间中，并锁定当前程序执行的线程，使当前线程进入容器的网络空间
// 返回值是函数指针，执行这个返回函数才会退出容器的网路空间
func enterContainerNetns(enLink *netlink.Link, cinfo *container.ContainerInfo) func() {
	//找到容器的NetNamespace
	f, err := os.OpenFile(fmt.Sprintf("/proc/%s/ns/net", cinfo.Pid), os.O_RDONLY, 0)
	if err != nil {
		logrus.Errorf("error get container net namespace, %v", err)
	}
	//取到文件描述符
	nsFD := f.Fd()
	//锁定当前程序锁执行的线程，如果不锁定的话，go的Goroutine可能会被调度到别的线程，
	//就不能保证一直在所需要的网络空间中了
	runtime.LockOSThread()
	//修改网络端点Veth的另一端，将其移动到容器的Net Namespace
	//以便后面从容器的Net Namespace退出
	if err = netlink.LinkSetNsFd(*enLink, int(nsFD)); err != nil {
		logrus.Errorf("error set link netns, %v", err)
	}
	//将当前进程加入到容器的Net Namespace中
	origns, err := netns.Get()
	if err != nil {
		logrus.Errorf("error get currnet netus, %v", err)
	}
	//返回之前Net Namespace的函数
	return func() {
		//恢复到上面获取的之前的Namespace
		netns.Set(origns)
		origns.Close()
		runtime.UnlockOSThread()
		f.Close()
	}
}

// 配置端口映射
func configPortMapping(ep *Endpoint, cinfo *container.ContainerInfo) error {
	//遍历容器端口映射列表
	for _, pm := range ep.PortMapping {
		portMapping := strings.Split(pm, ":")
		if len(portMapping) != 2 {
			logrus.Errorf("port mapping format error, %v", pm)
			continue
		}
		//在iptables的PREROUTING中添加DNAT规则
		//将宿主机的端口请求转发到容器的地址和端口上
		iptableCmd := fmt.Sprintf("-t nat -A PREROUTING -p tcp -m tcp --dport %s -j DNAT --to-destination %s:%s",
			portMapping[0], ep.IPAddress.String(), portMapping[1])
		//执行命令，添加端口映射转发规则
		cmd := exec.Command("iptables", strings.Split(iptableCmd, " ")...)
		output, err := cmd.Output()
		if err != nil {
			logrus.Errorf("iptables Output, %v", output)
			continue
		}
	}
	return nil
}

func Disconnect(networkName string, cinfo *container.ContainerInfo) error {
	return nil
}
