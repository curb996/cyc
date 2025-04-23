// internal/device/config.go
package device

// internal/device/config.go
type PointConfig struct {
	Name      string                 `json:"name"`
	Desc      string                 `json:"desc"`
	FuncCode  string                 `json:"funcCode"`  // 功能码   "hr", "co" 等
	RegAddr   uint16                 `json:"regAddr"`   // 寄存器地址
	RegNum    uint16                 `json:"regNum"`    // 寄存器数量
	Params    map[string]interface{} `json:"params"`    // func/address/quantity等
	DataType  string                 `json:"dataType"`  // float32/int16等
	SwapReg   bool                   `json:"swapReg"`   // 多寄存器高低字交换
	ByteOrder string                 `json:"byteOrder"` // big/little
	Rw        string                 `json:"rw"`        // "r", "w", "rw"
}

type DeviceConfig struct {
	Name        string                 `json:"name"`
	IpAddr      string                 `json:"ipAddr"`
	Protocol    string                 `json:"protocol"`
	Params      map[string]interface{} `json:"params"` // 协议全局参数
	Points      []PointConfig          `json:"points"`
	SlaveId     uint8                  `json:"slaveId"`     //从站id
	AdapterType string                 `json:"adapterType"` //适配器类型  比如:modbus、can等
	IntervalMs  int                    `json:"interval_ms"` // 采集周期（毫秒）
}

func FindPointConfigById(points []PointConfig, id string) *PointConfig {
	for i, p := range points {
		if p.Name == id {
			return &points[i]
		}
	}
	return nil
}
