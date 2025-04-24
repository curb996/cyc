// internal/device/manager.go
package device

import (
	"cycV2/internal/protocol"
	"fmt"
	"sync"
)

type DeviceManager struct {
	devices map[string]*ModbusDevice
	mu      sync.RWMutex
}

func NewDeviceManager() *DeviceManager {
	return &DeviceManager{devices: make(map[string]*ModbusDevice)}
}

func (dm *DeviceManager) Register(cfg DeviceConfig, adapter protocol.ProtocolAdapter) error {
	dm.mu.Lock()
	defer dm.mu.Unlock()
	if _, ok := dm.devices[cfg.Name]; ok {
		return fmt.Errorf("device already exists: %s", cfg.Name)
	}
	switch cfg.AdapterName {
	case "modbus":
		dm.devices[cfg.Name] = NewModbusDevice(cfg, adapter)
	default:
		fmt.Println("Register Failed....  AdapterName:", cfg.AdapterName)
	}

	return nil
}

func (dm *DeviceManager) GetDevice(name string) (*ModbusDevice, bool) {
	dm.mu.RLock()
	defer dm.mu.RUnlock()
	dev, ok := dm.devices[name]
	return dev, ok
}

func (dm *DeviceManager) CollectAll() map[string]map[string]interface{} {
	dm.mu.RLock()
	defer dm.mu.RUnlock()
	res := make(map[string]map[string]interface{})
	var wg sync.WaitGroup
	var mu sync.Mutex
	for name, dev := range dm.devices {
		wg.Add(1)
		go func(dev *ModbusDevice, name string) {
			defer wg.Done()
			values, _ := dev.CollectAllParallel()
			mu.Lock()
			res[name] = values
			mu.Unlock()
		}(dev, name)
	}
	wg.Wait()
	return res
}
