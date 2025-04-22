package collector

import (
	"context"
	"cyc/internal/config"
	"fmt"
	"sync"
	"time"

	"cyc/internal/device"

	"github.com/sirupsen/logrus"
)

// CollectTask 采集任务
type CollectTask struct {
	DeviceID string
	Interval time.Duration
	Points   []config.Point
	ctx      context.Context
	cancelFn context.CancelFunc
}

// Collector 数据采集器
type Collector struct {
	deviceFactory *device.DeviceFactory
	devices       map[string]device.Device
	tasks         map[string]*CollectTask
	dataChan      chan CollectedData
	logger        *logrus.Logger
	mu            sync.Mutex
}

// CollectedData 采集的数据
type CollectedData struct {
	DeviceID  string
	Timestamp time.Time
	Points    map[string]interface{}
}

// NewCollector 创建采集器
func NewCollector(deviceFactory *device.DeviceFactory, logger *logrus.Logger) *Collector {
	return &Collector{
		deviceFactory: deviceFactory,
		devices:       make(map[string]device.Device),
		tasks:         make(map[string]*CollectTask),
		dataChan:      make(chan CollectedData, 100),
		logger:        logger,
	}
}

// AddDevice 添加设备
func (c *Collector) AddDevice(deviceID string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, exists := c.devices[deviceID]; exists {
		return fmt.Errorf("device %s already added", deviceID)
	}

	device, err := c.deviceFactory.CreateDevice(deviceID)
	if err != nil {
		return fmt.Errorf("failed to create device %s: %w", deviceID, err)
	}

	c.devices[deviceID] = device
	return nil
}

// ConnectDevice 连接设备
func (c *Collector) ConnectDevice(deviceID string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	device, exists := c.devices[deviceID]
	if !exists {
		return fmt.Errorf("device %s not found", deviceID)
	}

	if device.IsConnected() {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := device.Connect(ctx); err != nil {
		return fmt.Errorf("failed to connect device %s: %w", deviceID, err)
	}

	return nil
}

// DisconnectDevice 断开设备连接
func (c *Collector) DisconnectDevice(deviceID string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	device, exists := c.devices[deviceID]
	if !exists {
		return fmt.Errorf("device %s not found", deviceID)
	}

	if !device.IsConnected() {
		return nil
	}

	// 先停止相关任务
	for id, task := range c.tasks {
		if task.DeviceID == deviceID {
			c.stopTaskLocked(id)
		}
	}

	if err := device.Disconnect(); err != nil {
		return fmt.Errorf("failed to disconnect device %s: %w", deviceID, err)
	}

	return nil
}

// StartTask 开始采集任务
func (c *Collector) StartTask(taskID, deviceID string, points []config.Point, interval time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, exists := c.tasks[taskID]; exists {
		return fmt.Errorf("task %s already exists", taskID)
	}

	device, exists := c.devices[deviceID]
	if !exists {
		return fmt.Errorf("device %s not found", deviceID)
	}

	if !device.IsConnected() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := device.Connect(ctx); err != nil {
			return fmt.Errorf("failed to connect device %s: %w", deviceID, err)
		}
	}

	ctx, cancelFn := context.WithCancel(context.Background())

	task := &CollectTask{
		DeviceID: deviceID,
		Interval: interval,
		Points:   points,
		ctx:      ctx,
		cancelFn: cancelFn,
	}

	c.tasks[taskID] = task

	// 转换为设备点位
	//devicePoints := make([]config.Point, 0, len(points))
	//deviceInfo := device.GetInfo()

	//for _, pointID := range points {
	//	// 这里需要根据实际情况获取点位信息
	//	// 简化起见，这里假设已经有一个点位映射
	//	// 实际项目中可能需要从设备配置或其他地方获取
	//	for _, p := range c.deviceFactory.GetDevicePoints(deviceID) {
	//		if p.ID == pointID {
	//			devicePoints = append(devicePoints, p)
	//			break
	//		}
	//	}
	//}

	// 启动采集协程
	go c.collectTask(taskID, device, points, interval)

	c.logger.Infof("Started collection task %s for device %s with %d points at interval %v",
		taskID, deviceID, len(points), interval)

	return nil
}

// StopTask 停止采集任务
func (c *Collector) StopTask(taskID string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.stopTaskLocked(taskID)
}

// stopTaskLocked 停止任务的内部实现（已加锁）
func (c *Collector) stopTaskLocked(taskID string) error {
	task, exists := c.tasks[taskID]
	if !exists {
		return fmt.Errorf("task %s not found", taskID)
	}

	task.cancelFn()
	delete(c.tasks, taskID)

	c.logger.Infof("Stopped collection task %s", taskID)

	return nil
}

// collectTask 执行采集任务
func (c *Collector) collectTask(taskID string, dev device.Device, points []config.Point, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	deviceID := dev.GetInfo().ID

	for {
		select {
		case <-c.tasks[taskID].ctx.Done():
			return

		case <-ticker.C:
			ctx, cancel := context.WithTimeout(context.Background(), interval/2)

			// 检查连接状态，如果断开则尝试重连
			if !dev.IsConnected() {
				c.logger.Warnf("Device %s disconnected, trying to reconnect", deviceID)
				if err := dev.Connect(ctx); err != nil {
					c.logger.Errorf("Failed to reconnect device %s: %v", deviceID, err)
					cancel()
					continue
				}
			}

			// 采集数据
			data, err := dev.ReadMultiple(ctx, points)
			if err != nil {
				c.logger.Errorf("Failed to cyc data from device %s: %v", deviceID, err)
				cancel()
				continue
			}

			// 发送采集结果
			c.dataChan <- CollectedData{
				DeviceID:  deviceID,
				Timestamp: time.Now(),
				Points:    data,
			}

			cancel()
		}
	}
}

// GetDataChan 获取数据通道
func (c *Collector) GetDataChan() <-chan CollectedData {
	return c.dataChan
}
