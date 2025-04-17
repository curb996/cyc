package modbus

import (
	"context"
	"cyc/internal/config"
	"fmt"
	"time"

	"github.com/grid-x/modbus"
)

// ModbusDevice 实现Modbus协议的设备
type ModbusDevice struct {
	info      config.DeviceConfig
	points    []config.Point
	client    modbus.Client
	connected bool
	pointsMap map[string]config.Point
	handler   modbus.ClientHandler
}

// NewModbusDevice 创建Modbus设备
func NewModbusDevice(info config.DeviceConfig, points []config.Point) (*ModbusDevice, error) {
	d := &ModbusDevice{
		info:      info,
		points:    points,
		connected: false,
		pointsMap: make(map[string]config.Point),
	}

	// 构建点位map，方便查找
	for _, p := range points {
		d.pointsMap[p.ID] = p
	}

	return d, nil
}

// Connect 连接设备
func (d *ModbusDevice) Connect(ctx context.Context) error {
	if d.connected {
		return nil
	}

	// 获取连接参数
	host := d.info.Params["host"].(string)
	port := d.info.Params["port"].(float64)
	//port, err := strconv.Atoi(portStr)
	//if err != nil {
	//	return fmt.Errorf("invalid port number: %w", err)
	//}

	// 创建连接
	handler := modbus.NewTCPClientHandler(fmt.Sprintf("%s:%d", host, int(port)))
	handler.Timeout = 10 * time.Second

	// 设置从站地址（如果有）
	if slaveID, ok := d.info.Params["slaveId"].(float64); ok {
		//id, err := strconv.Atoi(slaveID)
		//if err == nil {
		handler.SetSlave(byte(slaveID))
		fmt.Println("slaveId:", slaveID)
		//handler.SlaveId = byte(id)
		//}
	}

	// 连接服务器
	if err := handler.Connect(); err != nil {
		return fmt.Errorf("failed to connect to modbus server: %w", err)
	}
	d.handler = handler

	d.client = modbus.NewClient(handler)
	d.connected = true

	return nil
}

// Disconnect 断开连接
func (d *ModbusDevice) Disconnect() error {
	if !d.connected {
		return nil
	}

	err := d.handler.Close()
	if err != nil {
		return err
	}

	//if handler, ok := d.client.(*modbus.TCPClientHandler); ok {
	//	handler.Close()
	//}

	d.connected = false
	return nil
}

// IsConnected 检查是否已连接
func (d *ModbusDevice) IsConnected() bool {
	return d.connected
}

// Read 读取单个点位数据
func (d *ModbusDevice) Read(ctx context.Context, point config.Point) (interface{}, error) {
	if !d.connected {
		return nil, fmt.Errorf("device not connected")
	}

	// 解析地址
	address := point.Address
	//address, err := strconv.ParseUint(point.Address, 0, 16)
	//if err != nil {
	//	return nil, fmt.Errorf("invalid address: %w", err)
	//} else {
	//	//fmt.Println("address:", address, " num:", point.RegNum)
	//}
	fmt.Println("address:", address, " num:", point.RegNum, " regType:", point.RegType)
	// 根据数据类型选择读取方法
	var result interface{}

	switch point.RegType {
	case "coil":
		response, err := d.client.ReadCoils(address, 1)
		if err != nil {
			return nil, fmt.Errorf("failed to read coil: %w", err)
		}
		result = response[0] == 1

	case "discrete":
		response, err := d.client.ReadDiscreteInputs(uint16(address), 1)
		if err != nil {
			return nil, fmt.Errorf("failed to read discrete input: %w", err)
		}
		result = response[0] == 1

	case "holding":
		response, err := d.client.ReadHoldingRegisters(uint16(address), 1)
		if err != nil {
			return nil, fmt.Errorf("failed to read holding register: %w", err)
		}
		value := uint16(response[0])<<8 | uint16(response[1])
		result = float64(value) * point.Scale

	case "input":
		response, err := d.client.ReadInputRegisters(uint16(address), 1)
		if err != nil {
			return nil, fmt.Errorf("failed to read input register: %w", err)
		}
		value := uint16(response[0])<<8 | uint16(response[1])
		result = float64(value) * point.Scale

	default:
		return nil, fmt.Errorf("unsupported data type: %s", point.DataType)
	}

	return result, nil
}

// ReadMultiple 批量读取多个点位数据
func (d *ModbusDevice) ReadMultiple(ctx context.Context, points []config.Point) (map[string]interface{}, error) {
	result := make(map[string]interface{})

	for _, point := range points {
		value, err := d.Read(ctx, point)
		if err != nil {
			return nil, fmt.Errorf("failed to read point %s: %w", point.ID, err)
		}
		result[point.ID] = value
	}

	return result, nil
}

// Write 写入数据到指定点位
func (d *ModbusDevice) Write(ctx context.Context, point config.Point, value interface{}) error {
	if !d.connected {
		return fmt.Errorf("device not connected")
	}

	// 解析地址
	address := point.Address
	var err error
	//address, err := strconv.ParseUint(point.Address, 0, 16)
	//if err != nil {
	//	return fmt.Errorf("invalid address: %w", err)
	//}

	switch point.DataType {
	case "coil":
		boolValue, ok := value.(bool)
		if !ok {
			return fmt.Errorf("value must be boolean for coil")
		}

		if boolValue {
			_, err = d.client.WriteSingleCoil(uint16(address), 0xFF00)
		} else {
			_, err = d.client.WriteSingleCoil(uint16(address), 0x0000)
		}

	case "holding":
		var floatValue float64

		switch v := value.(type) {
		case float64:
			floatValue = v
		case float32:
			floatValue = float64(v)
		case int:
			floatValue = float64(v)
		case int64:
			floatValue = float64(v)
		default:
			return fmt.Errorf("unsupported value type for holding register")
		}

		// 应用缩放因子并转换为整数
		intValue := uint16(floatValue / point.Scale)
		_, err = d.client.WriteSingleRegister(uint16(address), intValue)

	default:
		return fmt.Errorf("writing to %s is not supported", point.DataType)
	}

	if err != nil {
		return fmt.Errorf("failed to write value: %w", err)
	}

	return nil
}

// GetInfo 获取设备信息
func (d *ModbusDevice) GetInfo() config.DeviceConfig {
	return d.info
}
