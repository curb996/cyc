package config

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"sort"
)

// DeviceConfigs 所有设备连接配置信息
type DeviceConfigs struct {
	Devices []DeviceConfig `json:"devices"`
}

// DeviceConfig 设备连接配置
type DeviceConfig struct {
	ID       string                 `json:"id"`
	Name     string                 `json:"name"`
	Protocol string                 `json:"protocol"`
	Model    string                 `json:"model"`
	Params   map[string]interface{} `json:"params"`
}

type PointConfigs struct {
	Models []PointConfig `json:"models"`
}

// PointConfig 设备点位配置
type PointConfig struct {
	DeviceModel string  `json:"modelId"`
	Points      []Point `json:"points"`
}

// Point 点位信息
type Point struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Address     uint16  `json:"address"`
	RegNum      uint16  `json:"regNum"`
	RegType     string  `json:"regType"`
	DataType    string  `json:"dataType"`
	Scale       float64 `json:"scaleFactor"`
	Unit        string  `json:"unit"`
	Description string  `json:"description"`
	ReadOnly    bool    `json:"readOnly"`
	RegSwap     bool    `json:"regSwap"`
}

// 批量读取组
type BatchGroup struct {
	StartAddress uint16
	RegCount     uint16
	RegType      string
	ReadOnly     bool
	Points       []Point
}

// ConfigManager 配置管理器
type ConfigManager struct {
	deviceConfigDir string
	pointsConfigDir string
	deviceConfigs   map[string]DeviceConfig
	pointsConfigs   map[string]PointConfig
}

// NewConfigManager 创建配置管理器
func NewConfigManager(deviceConfigDir, pointsConfigDir string) *ConfigManager {
	return &ConfigManager{
		deviceConfigDir: deviceConfigDir,
		pointsConfigDir: pointsConfigDir,
		deviceConfigs:   make(map[string]DeviceConfig),
		pointsConfigs:   make(map[string]PointConfig),
	}
}

// LoadAllConfigs 加载所有配置
func (cm *ConfigManager) LoadAllConfigs() error {
	// 加载设备配置
	deviceFiles, err := ioutil.ReadDir(cm.deviceConfigDir)
	if err != nil {
		return fmt.Errorf("failed to read device configs directory: %w", err)
	}

	for _, file := range deviceFiles {
		if filepath.Ext(file.Name()) == ".json" {
			filePath := filepath.Join(cm.deviceConfigDir, file.Name())
			data, err := ioutil.ReadFile(filePath)
			if err != nil {
				return fmt.Errorf("failed to read device config file %s: %w", filePath, err)
			}

			var deviceConfigs DeviceConfigs
			if err := json.Unmarshal(data, &deviceConfigs); err != nil {
				return fmt.Errorf("failed to parse device config %s: %w", filePath, err)
			}

			for k, v := range deviceConfigs.Devices {
				fmt.Println("k:", k, " model:", v.Model)
				cm.deviceConfigs[v.ID] = v
			}

		}
	}

	// 加载点位配置
	pointsFiles, err := ioutil.ReadDir(cm.pointsConfigDir)
	if err != nil {
		return fmt.Errorf("failed to read points configs directory: %w", err)
	}

	for _, file := range pointsFiles {
		if filepath.Ext(file.Name()) == ".json" {
			filePath := filepath.Join(cm.pointsConfigDir, file.Name())
			data, err := ioutil.ReadFile(filePath)
			if err != nil {
				return fmt.Errorf("failed to read points config file %s: %w", filePath, err)
			}

			var models PointConfigs
			if err := json.Unmarshal(data, &models); err != nil {
				return fmt.Errorf("failed to parse points config %s: %w", filePath, err)
			}

			for k, v := range models.Models {
				fmt.Println("model=> k:", k, " model:", v.DeviceModel)
				cm.pointsConfigs[v.DeviceModel] = v
				batchGroups := determineBatchGroups(v.Points)
				// 使用批量读取组进行通信...
				for _, group := range batchGroups {
					if len(group.Points) > 1 {
						// 批量读取
						fmt.Printf("批量读取: 类型=%s, 地址=%d, 数量=%d\n",
							group.RegType, group.StartAddress, group.RegCount)
					} else {
						// 单独读取
						fmt.Printf("单独读取: 类型=%s, 地址=%d, 数量=%d\n",
							group.RegType, group.StartAddress, group.RegCount)
					}
				}
			}

		}
	}

	return nil
}

// GetDeviceConfig 获取设备配置
func (cm *ConfigManager) GetDeviceConfig(deviceID string) (DeviceConfig, bool) {
	config, exists := cm.deviceConfigs[deviceID]
	return config, exists
}

// GetPointsConfig 获取点位配置
func (cm *ConfigManager) GetPointsConfig(deviceModel string) (PointConfig, bool) {
	config, exists := cm.pointsConfigs[deviceModel]
	return config, exists
}

// GetAllDeviceConfigs 获取所有设备配置
func (cm *ConfigManager) GetAllDeviceConfigs() map[string]DeviceConfig {
	return cm.deviceConfigs
}

// 获取数据类型实际占用的寄存器数量
func getRegNumByDataType(dataType string) uint16 {
	switch dataType {
	case "float32", "int32", "uint32":
		return 2
	case "float64", "int64", "uint64":
		return 4
	default: // 默认为uint16, int16等16位类型
		return 1
	}
}

// 验证并修正点位配置
func validatePoints(points []Point) []Point {
	correctedPoints := make([]Point, len(points))
	copy(correctedPoints, points)

	for i := range correctedPoints {
		expectedRegNum := getRegNumByDataType(correctedPoints[i].DataType)
		if correctedPoints[i].RegNum != expectedRegNum {
			fmt.Printf("警告: 点位 %s 配置的RegNum=%d 但数据类型 %s 需要 %d 个寄存器. 已自动调整.\n",
				correctedPoints[i].Name, correctedPoints[i].RegNum, correctedPoints[i].DataType, expectedRegNum)
			correctedPoints[i].RegNum = expectedRegNum
		}
	}

	return correctedPoints
}

// 判断两个点是否可以批量读取
func canBatchRead(p1, p2 Point) bool {
	// 判断条件：相同寄存器类型、相同读写属性、地址连续、相同寄存器交换方式
	if p1.RegType != p2.RegType || p1.ReadOnly != p2.ReadOnly || p1.RegSwap != p2.RegSwap {
		return false
	}

	// 判断地址是否连续
	expectedNextAddress := p1.Address + p1.RegNum
	return expectedNextAddress == p2.Address
}

// 确定批量读取组
func determineBatchGroups(points []Point) []BatchGroup {
	// 首先验证和调整点位配置
	correctedPoints := validatePoints(points)

	// 按地址排序
	sort.Slice(correctedPoints, func(i, j int) bool {
		return correctedPoints[i].Address < correctedPoints[j].Address
	})

	var batchGroups []BatchGroup

	if len(correctedPoints) == 0 {
		return batchGroups
	}

	// 创建第一个组
	currentGroup := BatchGroup{
		StartAddress: correctedPoints[0].Address,
		RegCount:     correctedPoints[0].RegNum,
		RegType:      correctedPoints[0].RegType,
		ReadOnly:     correctedPoints[0].ReadOnly,
		Points:       []Point{correctedPoints[0]},
	}

	// 遍历所有点位
	for i := 1; i < len(correctedPoints); i++ {
		currentPoint := correctedPoints[i]
		lastPoint := currentGroup.Points[len(currentGroup.Points)-1]

		// 检查是否可以批量读取
		if canBatchRead(lastPoint, currentPoint) {
			// 可以批量读取，添加到当前组
			currentGroup.RegCount += currentPoint.RegNum
			currentGroup.Points = append(currentGroup.Points, currentPoint)
		} else {
			// 不能批量读取，创建新组
			batchGroups = append(batchGroups, currentGroup)
			currentGroup = BatchGroup{
				StartAddress: currentPoint.Address,
				RegCount:     currentPoint.RegNum,
				RegType:      currentPoint.RegType,
				ReadOnly:     currentPoint.ReadOnly,
				Points:       []Point{currentPoint},
			}
		}
	}

	// 添加最后一个组
	batchGroups = append(batchGroups, currentGroup)

	return batchGroups
}
