package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"cyc/internal/collector"
	"cyc/internal/config"
	"cyc/internal/device"
	"github.com/sirupsen/logrus"
)

func main() {
	// 命令行参数
	configDir := flag.String("config", "./configs", "配置文件目录")
	logLevel := flag.String("log-level", "info", "日志级别")
	flag.Parse()

	// 设置日志
	logger := logrus.New()
	level, err := logrus.ParseLevel(*logLevel)
	if err != nil {
		logger.SetLevel(logrus.InfoLevel)
	} else {
		logger.SetLevel(level)
	}

	logger.SetFormatter(&logrus.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "2006-01-02 15:04:05",
	})

	// 加载配置
	deviceConfigDir := filepath.Join(*configDir, "devices")
	pointsConfigDir := filepath.Join(*configDir, "points")

	configManager := config.NewConfigManager(deviceConfigDir, pointsConfigDir)
	if err := configManager.LoadAllConfigs(); err != nil {
		logger.Fatalf("Failed to load configurations: %v", err)
	}

	// 创建设备工厂
	deviceFactory := device.NewDeviceFactory(configManager)

	// 创建采集器
	collector := collector.NewCollector(deviceFactory, logger)

	// 初始化所有设备
	deviceConfigs := configManager.GetAllDeviceConfigs()
	for deviceID := range deviceConfigs {
		if err := collector.AddDevice(deviceID); err != nil {
			logger.Errorf("Failed to add device %s: %v", deviceID, err)
			continue
		}

		logger.Infof("Added device: %s", deviceID)
	}

	// 启动数据处理协程
	go func() {
		for data := range collector.GetDataChan() {
			logger.Infof("Received data from device %s at %v with %d points",
				data.DeviceID, data.Timestamp, len(data.Points))

			for k, v := range data.Points {
				logger.Infof("Point ID: %s, Value: %v", k, v)
			}

			// 这里可以添加数据处理、存储等逻辑
			// 例如写入数据库、转发到消息队列等
		}
	}()

	// 启动默认采集任务
	for deviceID := range deviceConfigs {
		device, _ := deviceFactory.CreateDevice(deviceID)
		deviceInfo := device.GetInfo()
		//deviceInfo, _ := deviceFactory.GetDeviceInfo(deviceID)
		pointsConfig, exists := configManager.GetPointsConfig(deviceInfo.Model)
		if !exists {
			logger.Warnf("No points configuration found for device %s (model: %s)",
				deviceID, deviceInfo.Model)
			continue
		}

		// 获取所有点位ID
		//pointIDs := make([]string, 0, len(pointsConfig.Points))
		//for _, p := range pointsConfig.Points {
		//	pointIDs = append(pointIDs, p.ID)
		//}

		// 创建默认采集任务
		taskID := fmt.Sprintf("task_%s", deviceID)
		interval := 5 * time.Second // 默认5秒采集一次

		if err := collector.StartTask(taskID, deviceID, pointsConfig.Points, interval); err != nil {
			logger.Errorf("Failed to start task for device %s: %v", deviceID, err)
		}
	}

	// 等待中断信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	// 关闭所有设备连接
	for deviceID := range deviceConfigs {
		if err := collector.DisconnectDevice(deviceID); err != nil {
			logger.Errorf("Failed to disconnect device %s: %v", deviceID, err)
		}
	}

	logger.Info("Program exited")
}
