package subsystems

// 传递资源限制配置的结构体
type ResourceConfig struct {
	MemoryLimit string //内存限制
	CpuShare    string //CPU权重
	CpuSet      string //CPU核心数
}

// cgroup抽象为path，即cgruop在hierarchy的路径，也就是虚拟文件系统中的虚拟路径
type Subsystem interface {
	//返回subsystem的名字，如 cpu memory
	Name() string
	//设置某个cgroup的资源限制
	Set(path string, res *ResourceConfig) error
	//将进程添加到某个cgroup中
	Apply(path string, pid int) error
	//移除某个cgroup
	Remove(path string) error
}

// 通过不同subsystem初始化实例，创建资源限制处理链表
var (
	SubsystemsIns = []Subsystem{
		&CpusetSubSystem{},
		&MemorySubSystem{},
		&CpuSubSystem{},
	}
)
