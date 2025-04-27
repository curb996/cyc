package device

import (
	"cycV2/internal/protocol"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"sync"

	"github.com/fsnotify/fsnotify"
)

//TODO
//配置合法性校验
//高速file watch降抖（同一时间段热更新概率降低）
//日志告警、回滚策略

type BusInstance struct {
	Devices []*DeviceInstance
	StopCh  chan struct{}
	Wg      *sync.WaitGroup
	BusID   string
}

type DeviceInstance struct {
	Cfg    DeviceConfig
	StopCh chan struct{}
	//CollectWg *sync.WaitGroup
}

type Manager struct {
	Buses      map[string][]*ModbusDevice // 当前所有bus分组
	RawCh      chan RawCollectResult      // 公共原始结果通道，供所有设备采集送入
	BusStop    map[string]chan struct{}   // 每个bus一个stop通道用于优雅重启
	configPath string                     // 配置文件地址
	//devices    map[string]*DeviceInstance
	mu sync.Mutex
}

func NewManager(configPath string) *Manager {
	return &Manager{
		Buses:      make(map[string][]*ModbusDevice),
		configPath: configPath,
		BusStop:    make(map[string]chan struct{}), // ← 新增
		//devices:    make(map[string]*DeviceInstance),
		RawCh: make(chan RawCollectResult, 100), // buffer依据实际业务量调整
	}
}

// 本地加载并全量替换（可做增量更新优化）
//func (m *Manager) ReloadFromFile() error {
//	m.mu.Lock()
//	defer m.mu.Unlock()
//
//	confData, err := ioutil.ReadFile(m.configPath)
//	if err != nil {
//		return err
//	}
//
//	var newDevices []DeviceConfig
//	if err = json.Unmarshal(confData, &newDevices); err != nil {
//		return err
//	}
//
//	// 设备去重与比对
//	newSet := make(map[string]DeviceConfig)
//	for _, d := range newDevices {
//		newSet[d.Name] = d
//	}
//
//	// 1. 停掉已删除的设备
//	for name, inst := range m.devices {
//		if _, ok := newSet[name]; !ok {
//			close(inst.StopCh)
//			if inst.CollectWg != nil {
//				inst.CollectWg.Wait() // 等采集goroutine退出
//			}
//			delete(m.devices, name)
//			log.Printf("设备[%s] 已下线", name)
//		}
//	}
//
//	// 2. 注册新设备、更新变更
//	for name, cfg := range newSet {
//		if _, ok := m.devices[name]; !ok {
//			inst := &DeviceInstance{
//				Cfg:    cfg,
//				StopCh: make(chan struct{}),
//			}
//			inst.CollectWg = StartCollectPipeline(inst, m.RawCh, inst.StopCh) // 你的pipeline
//			m.devices[name] = inst
//			log.Printf("新设备[%s] 已上线", name)
//		}
//		// TODO: 对已存在设备支持freq等变更（可复杂对比后重启相关采集线程）
//	}
//	return nil
//}

// 假设你已按 bus_id 分组好了 devMap: map[string][]*ModbusDevice
// 以及 Manager有
//  - m.RawCh   chan RawCollectResult  // 全局采集数据通道
//  - m.BusStop map[string]chan struct{} // 各bus的stop信号
//  - m.Buses   map[string][]*ModbusDevice // 现在的bus分组

func (m *Manager) ReloadFromFile() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 1. 加载JSON获得 []*ModbusDevice，分好 bus_id 分组
	// busDevicesMap := map[string][]*ModbusDevice // bus_id -> 同一总线设备

	busDevicesMap, err := loadDevicesByBus(m.configPath)
	if err != nil {
		return err
	}

	// 2. 关闭和移除所有“旧的bus worker”
	for busID, stopCh := range m.BusStop {
		close(stopCh) // 通知worker退出
		log.Printf("关闭旧总线worker %s", busID)
		delete(m.BusStop, busID)
	}

	// 3. 启动新的“bus worker”各自管理一个物理总线
	m.Buses = busDevicesMap
	for busID, devices := range busDevicesMap {
		if len(devices) == 0 {
			continue
		}
		log.Printf("启动采集流水线，总线bus[%s]有%d个设备", busID, len(devices))
		stopCh := make(chan struct{})
		m.BusStop[busID] = stopCh
		StartCollectPipeline(devices, m.RawCh, stopCh)
	}

	return nil
}

//func (m *Manager) ReloadFromFile() error {
//	m.mu.Lock()
//	defer m.mu.Unlock()
//	// 1. 读取配置并组装 bus_id -> []*DeviceInstance
//	confData, err := ioutil.ReadFile(m.configPath)
//	if err != nil {
//		return err
//	}
//	var newDevices []DeviceConfig
//	if err := json.Unmarshal(confData, &newDevices); err != nil {
//		return err
//	}
//	busMap := make(map[string][]*DeviceInstance)
//	for _, devCfg := range newDevices {
//		inst := &DeviceInstance{
//			Cfg:    devCfg,
//			StopCh: make(chan struct{}),
//		}
//		busMap[devCfg.BusId] = append(busMap[devCfg.BusId], inst)
//	}
//	// 2. 关闭和移除旧的bus采集worker
//	for busID, bus := range m.Buses {
//		close(bus.StopCh)
//		bus.Wg.Wait()
//		delete(m.Buses, busID)
//		log.Printf("关闭Bus[%s]", busID)
//	}
//	// 3. 启动每条bus的轮询worker
//	for busID, devices := range busMap {
//		bus := &BusInstance{
//			Devices: devices,
//			StopCh:  make(chan struct{}),
//			Wg:      &sync.WaitGroup{},
//			BusID:   busID,
//		}
//		bus.Wg.Add(1)
//		go func(busIns *BusInstance) {
//			defer busIns.Wg.Done()
//			StartCollectPipeline(busIns, m.RawCh)
//		}(bus)
//		m.Buses[busID] = bus
//		log.Printf("启动Bus[%s] with %d devices", busID, len(devices))
//	}
//	return nil
//}

// 热加载控制
func (m *Manager) WatchAndReload() {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatalf("创建文件监听失败: %v", err)
	}
	defer watcher.Close()

	configDir := "."
	if abs, _ := os.Stat(m.configPath); !abs.IsDir() {
		configDir = getParentDir(m.configPath)
	}
	if err := watcher.Add(configDir); err != nil {
		log.Fatalf("添加监听失败: %v", err)
	}

	log.Printf("开始监听配置文件: %s", m.configPath)
	_ = m.ReloadFromFile()
	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}
			if event.Op&fsnotify.Write == fsnotify.Write && event.Name == m.configPath {
				log.Printf("检测到配置文件变更，自动热加载")
				if err := m.ReloadFromFile(); err != nil {
					log.Printf("热加载失败: %v", err)
				}
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			log.Printf("监听错误: %v", err)
		}
	}
}

func getParentDir(filePath string) string {
	d := filePath
	for len(d) > 0 && d[len(d)-1] != '/' && d[len(d)-1] != '\\' {
		d = d[:len(d)-1]
	}
	if len(d) == 0 {
		return "."
	}
	return d
}

func loadDevicesByBus(configPath string) (map[string][]*ModbusDevice, error) {
	dat, err := ioutil.ReadFile(configPath)
	if err != nil {
		return nil, err
	}
	var devCfgs []*DeviceConfig
	err = json.Unmarshal(dat, &devCfgs)
	if err != nil {
		return nil, err
	}
	// map[bus_id][]*ModbusDevice
	busGroup := make(map[string][]*ModbusDevice)
	for _, cfg := range devCfgs {
		busID := cfg.BusId
		adapter, err := protocol.GetAdapter(cfg.AdapterName, cfg.Params)
		if err != nil {
			fmt.Println("protocol.GetAdapter Failed...name:", cfg.AdapterName, " err:", err)
			continue
		}
		md := NewModbusDevice(*cfg, adapter)
		busGroup[busID] = append(busGroup[busID], md)
	}
	return busGroup, nil
}

//func main() {
//	configPath := "./devices_config.json"
//	manager := device.NewManager(configPath)
//
//	// 开一个goroutine监听配置、自动热加载
//	go manager.WatchAndReload()
//
//	select {} // 保持主进程
//}
