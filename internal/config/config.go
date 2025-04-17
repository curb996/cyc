package config

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"
)

// DeviceConfigs 所有设备连接配置信息
type DeviceConfigs struct {
	Devices []DeviceConfig `json:"devices"`
}

// DeviceConfig 设备连接配置
type DeviceConfig struct {
	ID       string            `json:"id"`
	Name     string            `json:"name"`
	Protocol string            `json:"protocol"`
	Model    string            `json:"model"`
	Params   map[string]string `json:"params"`
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
	Address     string  `json:"address"`
	DataType    string  `json:"dataType"`
	Scale       float64 `json:"scaleFactor"`
	Unit        string  `json:"unit"`
	Description string  `json:"description"`
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
