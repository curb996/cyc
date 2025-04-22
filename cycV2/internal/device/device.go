package device

// DeviceDriver 统一设备接口
type DeviceDriver interface {
	Collect() (map[string]interface{}, error)                   // 采集数据
	Control(action string, params map[string]interface{}) error // 下发控制命令
	GetName() string
}

// DeviceConfig 设备配置
type DeviceConfig struct {
	Name     string
	Type     string
	Protocol string
	Addr     string
	Params   map[string]interface{}
}
