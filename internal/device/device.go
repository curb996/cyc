package device

import (
	"context"
	"cyc/internal/config"
)

// Point 表示一个数据点位
//type Point struct {
//	ID          string  // 点位ID
//	Name        string  // 点位名称
//	Address     string  // 点位地址
//	DataType    string  // 数据类型
//	Scale       float64 // 缩放因子
//	Unit        string  // 单位
//	Description string  // 描述
//}

// Device 定义设备通用接口
type Device interface {
	// Connect 连接设备
	Connect(ctx context.Context) error

	// Disconnect 断开连接
	Disconnect() error

	// IsConnected 检查是否已连接
	IsConnected() bool

	// Read 读取指定点位的数据
	Read(ctx context.Context, point config.Point) (interface{}, error)

	// ReadMultiple 批量读取多个点位数据
	ReadMultiple(ctx context.Context, points []config.Point) (map[string]interface{}, error)

	// Write 写入数据到指定点位
	Write(ctx context.Context, point config.Point, value interface{}) error

	// GetInfo 获取设备信息
	GetInfo() config.DeviceConfig
}

// DeviceConfig 设备信息
//type DeviceConfig struct {
//	ID       string            // 设备ID
//	Name     string            // 设备名称
//	Protocol string            // 设备协议
//	Model    string            // 设备型号
//	Params   map[string]string // 连接参数
//}
