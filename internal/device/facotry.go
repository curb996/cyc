package device

import (
	"fmt"

	"cyc/internal/config"
	"cyc/pkg/protocol/modbus"
)

// DeviceFactory 设备工厂
type DeviceFactory struct {
	configManager *config.ConfigManager
}

// NewDeviceFactory 创建设备工厂
func NewDeviceFactory(configManager *config.ConfigManager) *DeviceFactory {
	return &DeviceFactory{
		configManager: configManager,
	}
}

// CreateDevice 根据设备ID创建设备实例
func (f *DeviceFactory) CreateDevice(deviceID string) (Device, error) {
	deviceConfig, exists := f.configManager.GetDeviceConfig(deviceID)
	if !exists {
		return nil, fmt.Errorf("device config not found for ID: %s", deviceID)
	}

	pointsConfig, exists := f.configManager.GetPointsConfig(deviceConfig.Model)
	if !exists {
		return nil, fmt.Errorf("points config not found for model: %s", deviceConfig.Model)
	}

	// 转换点位配置
	points := make([]config.Point, 0, len(pointsConfig.Points))
	for _, p := range pointsConfig.Points {
		points = append(points, config.Point{
			ID:          p.ID,
			Name:        p.Name,
			Address:     p.Address,
			RegType:     p.RegType,
			RegNum:      p.RegNum,
			DataType:    p.DataType,
			Scale:       p.Scale,
			Unit:        p.Unit,
			Description: p.Description,
			ReadOnly:    p.ReadOnly,
			RegSwap:     p.RegSwap,
		})
	}

	// 根据协议类型创建对应的设备实例
	var device Device
	var err error

	switch deviceConfig.Protocol {
	case "modbus":
		device, err = modbus.NewModbusDevice(config.DeviceConfig{
			ID:       deviceConfig.ID,
			Name:     deviceConfig.Name,
			Protocol: deviceConfig.Protocol,
			Model:    deviceConfig.Model,
			Params:   deviceConfig.Params,
		}, points)

	case "iec104":
		//device, err = iec104.NewIEC104Device(DeviceConfig{
		//	ID:       deviceConfig.ID,
		//	Name:     deviceConfig.Name,
		//	Protocol: deviceConfig.Protocol,
		//	Model:    deviceConfig.Model,
		//	Params:   deviceConfig.Params,
		//}, points)

	case "iec61850":
		//device, err = iec61850.NewIEC61850Device(DeviceConfig{
		//	ID:       deviceConfig.ID,
		//	Name:     deviceConfig.Name,
		//	Protocol: deviceConfig.Protocol,
		//	Model:    deviceConfig.Model,
		//	Params:   deviceConfig.Params,
		//}, points)

	case "can":
		//device, err = can.NewCANDevice(DeviceConfig{
		//	ID:       deviceConfig.ID,
		//	Name:     deviceConfig.Name,
		//	Protocol: deviceConfig.Protocol,
		//	Model:    deviceConfig.Model,
		//	Params:   deviceConfig.Params,
		//}, points)

	default:
		return nil, fmt.Errorf("unsupported protocol: %s", deviceConfig.Protocol)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create device: %w", err)
	}

	return device, nil
}
