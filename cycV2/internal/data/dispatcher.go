package data

import (
	"sync"
)

// DataDispatcher 接口。上传/分发数据都必须实现该接口
type DataDispatcher interface {
	Dispatch(deviceName string, points map[string]interface{}) error
}

var (
	dispatcherMu sync.RWMutex
	dispatchers  = make(map[string]DataDispatcher)
	defaultType  = "log" // 默认用日志型
)

// Register 注册分发实现
func Register(name string, d DataDispatcher) {
	dispatcherMu.Lock()
	defer dispatcherMu.Unlock()
	dispatchers[name] = d
}

// GetDispatcherByName 默认获得指定类型，如未注册返回nil
func GetDispatcherByName(name string) DataDispatcher {
	dispatcherMu.RLock()
	defer dispatcherMu.RUnlock()
	return dispatchers[name]
}

// SetDefaultType 变更默认类型（可选）
func SetDefaultType(name string) {
	defaultType = name
}

// GetDefaultDispatcher 获取默认实现，如未注册返回nil
func GetDefaultDispatcher() DataDispatcher {
	return GetDispatcherByName(defaultType)
}
