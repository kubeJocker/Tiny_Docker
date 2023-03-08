package network

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"github.com/vishvananda/netlink"
	"net"
	"os/exec"
	"strings"
)

type BridgeNetworkDriver struct {
}

func (d *BridgeNetworkDriver) Name() string {
	return "bridge"
}

func (d *BridgeNetworkDriver) Create(subnet string, name string) (*Network, error) {
	ip, ipRange, _ := net.ParseCIDR(subnet)
	ipRange.IP = ip
	n := &Network{
		Name:    name,
		IpRange: ipRange,
	}
	err := d.initBridge(n)
	if err != nil {
		log.Errorf("error init bridge: %v", err)
	}
	return n, err
}

// 初始化Bridge设备
func (d *BridgeNetworkDriver) initBridge(n *Network) error {
	//创建bridge虚拟设备
	bridgeName := n.Name
	if err := createBridgeInterface(bridgeName); err != nil {
		return fmt.Errorf("Error add bridge: %s, Error: %v", bridgeName, err)
	}
	//设置Bridge设备的地址和路由
	gatewatIP := *n.IpRange
	gatewatIP.IP = n.IpRange.IP
	if err := setInterfaceIP(bridgeName, gatewatIP.String()); err != nil {
		return fmt.Errorf("Error assigning address: %s on bridge: %s with an error of: %v", gatewatIP, bridgeName, err)
	}

	//启动Bridge设备
	if err := setInterfaceUP(bridgeName); err != nil {
		return fmt.Errorf("Error set bridge up: %s, Error: %v", bridgeName, err)
	}
	//设置ipta的SNAT规则
	if err := setupIPTables(bridgeName, n.IpRange); err != nil {
		return fmt.Errorf("Error setting iptables for %s: %v", bridgeName, err)
	}
	return nil
}

// 创建linuxbridge设备
func createBridgeInterface(bridgeName string) error {
	_, err := net.InterfaceByName(bridgeName)
	if err != nil || !strings.Contains(err.Error(), "no such network interface") {
		return err
	}

	//初始化一个netlink的Link基础对象
	la := netlink.NewLinkAttrs()
	la.Name = bridgeName

	//使用刚才创建的Link的属性创建netlink的Bridge对象
	br := &netlink.Bridge{LinkAttrs: la}
	//调用netlink的linkadd方法，创建bridge虚拟网络设备
	//相当于ip link add XXXX
	if err := netlink.LinkAdd(br); err != nil {
		return fmt.Errorf("Bridge creation failed for bridge %s: %v", bridgeName, err)
	}
	return nil
}

// 设置网络接口的IP地址
func setInterfaceIP(name string, rawIP string) error {
	//找到需要设置的网络接口
	iface, err := netlink.LinkByName(name)
	if err != nil {
		return fmt.Errorf("error get interface: %v", err)
	}
	//ipNet包含网段和原始IP
	ipNet, err := netlink.ParseIPNet(rawIP)
	if err != nil {
		return err
	}
	//通过AddrAdd给网络配置地址，相当于ip addr add XXX
	//如果配置了地址所在网段信息，还会将路由表转发到这个testbridge的网络接口上
	addr := &netlink.Addr{IPNet: ipNet}
	return netlink.AddrAdd(iface, addr)
}

// 设置网络接口为up状态
func setInterfaceUP(interfaceName string) error {

	iface, err := netlink.LinkByName(interfaceName)
	if err != nil {
		return fmt.Errorf("Error retrieving a link named [ %s ]: %v", interfaceName, err)
	}
	//等价于 ip link set XXX UP
	if err := netlink.LinkSetUp(iface); err != nil {
		return fmt.Errorf("Error enabling interface for %s: %v", interfaceName, err)
	}
	return nil
}

// 设置iptables对应bridge的MASQUERADE规则
func setupIPTables(bridgeName string, subnet *net.IPNet) error {
	//通过命令方式配置
	iptablesCmd := fmt.Sprintf("-t nat -A POSTROUTING -s %s ! -o %s -j MASQUERADE", subnet.String(), bridgeName)
	cmd := exec.Command("iptables", strings.Split(iptablesCmd, "")...)
	//执行iptalbes命令配置SNAT规则
	output, err := cmd.Output()
	if err != nil {
		log.Errorf("iptable Output, %v", output)
	}
	return nil
}

func (d *BridgeNetworkDriver) Delete(network Network) error {
	bridgeName := network.Name
	//通过linkbyname找到网络对应设备
	br, err := netlink.LinkByName(bridgeName)
	if err != nil {
		return err
	}
	return netlink.LinkDel(br)
}

// 连接容器到之前创建的网络
func (d *BridgeNetworkDriver) Connect(network *Network, endpoint *Endpoint) error {
	//获取网络名，并即Bridge接口，然后获取接口对象和属性
	bridgeName := network.Name
	br, err := netlink.LinkByName(bridgeName)
	if err != nil {
		return err
	}
	//创建Veth接口配置
	la := netlink.NewLinkAttrs()
	la.Name = endpoint.ID[:5]
	//通过设置Veth接口的master属性，设置这个Veht的一端挂载到网路对应的Bridge上
	la.MasterIndex = br.Attrs().Index
	//创建Veth对象，通过PeerName配置Veth另一端的接口名
	endpoint.Device = netlink.Veth{
		LinkAttrs: la,
		PeerName:  "cif-" + endpoint.ID[:5],
	}

	//调用linkAdd创建这个Veth，另一端同时被挂载到Bridge上
	if err := netlink.LinkAdd(&endpoint.Device); err != nil {
		return fmt.Errorf("Error Add Endpoint Device: %v", err)
	}
	//启动Veth
	if err = netlink.LinkSetUp(&endpoint.Device); err != nil {
		return fmt.Errorf("Error set Endpoint Device up: %v", err)
	}
	return nil
}

func (d *BridgeNetworkDriver) Disconnect(network Network, endpoint *Endpoint) error {
	return nil
}
