// internal/driver/driver.go
package driver

import (
	"cycV2/internal/protocol"
	"fmt"
)

type Driver interface {
	// 生命周期管理
	Start() error
	Stop() error
	// 执行一次数据采集，点ID->解析后的值
	Collect() (map[string]interface{}, error)
	// 写控制命令
	Write(point string, value interface{}) error
}

// 驱动工厂注册
type DriverFactory func(cfg map[string]interface{}, adapter protocol.ProtocolAdapter) (Driver, error)

var driverFactories = map[string]DriverFactory{}

func Register(name string, factory DriverFactory) {
	driverFactories[name] = factory
}
func Create(name string, cfg map[string]interface{}, adapter protocol.ProtocolAdapter) (Driver, error) {
	f, ok := driverFactories[name]
	if !ok {
		return nil, fmt.Errorf("driver not registered: %s", name)
	}
	return f(cfg, adapter)
}
