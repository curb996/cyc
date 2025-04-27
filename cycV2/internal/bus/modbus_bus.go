// internal/bus/bus.go
package bus

import (
	"cycV2/internal/device"
	"cycV2/internal/protocol"
	"encoding/binary"
	"fmt"
	"sort"
	"sync"
	"time"
)

type WriteTask struct {
	DeviceName string
	PointID    string
	Value      interface{}
	RespCh     chan error
}

type PollTask struct{}

type ModbusBus struct {
	Name    string
	Adapter protocol.ProtocolAdapter
	Devices []*device.ModbusDevice

	// 队列
	ctrlQ chan *WriteTask // 控制高优队列
	pollQ chan *PollTask  // 采集低优队列
	quitQ chan struct{}
	wg    sync.WaitGroup

	CycleMs int
}

// 工厂
func NewModbusBus(name string, adapter protocol.ProtocolAdapter, devices []*device.ModbusDevice, cycleMs int) *ModbusBus {
	return &ModbusBus{
		Name: name, Adapter: adapter, Devices: devices,
		CycleMs: cycleMs,
		ctrlQ:   make(chan *WriteTask, 8), pollQ: make(chan *PollTask, 16),
		quitQ: make(chan struct{}),
	}
}

// 采集任务加入（通用场景下可由采集调度协程触发）
func (b *ModbusBus) EnqueuePoll() {
	b.pollQ <- &PollTask{}
}

// 控制外部调用接口
func (b *ModbusBus) ControlAsync(deviceName, pointId string, val interface{}) <-chan error {
	resp := make(chan error, 1)
	b.ctrlQ <- &WriteTask{
		DeviceName: deviceName, PointID: pointId, Value: val, RespCh: resp,
	}
	return resp
}

// 总线worker循环（确保串行性！）
func (b *ModbusBus) Start() {
	b.wg.Add(1)
	go func() {
		defer b.wg.Done()
		ticker := time.NewTicker(time.Millisecond * time.Duration(b.CycleMs))
		defer ticker.Stop()
		for {
			select {
			case <-b.quitQ:
				return
			case task := <-b.ctrlQ: // 控制高优先级
				b.handleControl(task)
			default:
				select {
				case task := <-b.ctrlQ: // 再次优先取控制
					b.handleControl(task)
				case <-ticker.C:
					b.doBatchCollect()
				case <-b.pollQ:
					b.doBatchCollect()
				}
			}
		}
	}()
}

//func (b *ModbusBus) handleControl(task *WriteTask) {
//	for _, dev := range b.Devices {
//		if dev.Cfg.Name == task.DeviceName {
//			// 控制实际调用协议写命令
//			point := findPointConfigById(dev.Cfg.Points, task.PointId)
//			if point == nil {
//				task.RespCh <- fmt.Errorf("point not found")
//				return
//			}
//			// 装包略...
//			err := dev.Adapter.WriteModbus()
//			task.RespCh <- err
//			return
//		}
//	}
//	task.RespCh <- fmt.Errorf("device not exist")
//}

func (b *ModbusBus) handleControl(task *WriteTask) {
	for _, dev := range b.Devices {
		if dev.Cfg.Name == task.DeviceName {
			point := device.FindPointConfigById(dev.Cfg.Points, task.PointID)
			if point == nil {
				task.RespCh <- fmt.Errorf("point %s not exist", task.PointID)
				return
			}

			// 构造写入data
			var writeData []byte
			switch v := task.Value.(type) {
			case uint16:
				writeData = make([]byte, 2)
				binary.BigEndian.PutUint16(writeData, v)
			case []byte:
				writeData = v
			default:
				task.RespCh <- fmt.Errorf("unsupported data type: %T", v)
				return
			}
			unitId := dev.Cfg.SlaveId
			funcCode := point.FuncCode
			address := point.RegAddr

			// 优先走专用接口
			if mod, ok := dev.Adapter.(interface {
				WriteModbus(funcCode string, addr uint16, data []byte) error
			}); ok {
				err := mod.WriteModbus(funcCode, address, writeData)
				task.RespCh <- err
			} else {
				// 回落到通用接口
				params := map[string]interface{}{
					"func":     funcCode,
					"slave_id": unitId,
				}
				addrStr := fmt.Sprintf("%d", address)
				err := dev.Adapter.Write(addrStr, writeData, params)
				task.RespCh <- err
			}
			return
		}
	}
	task.RespCh <- fmt.Errorf("device %s not exist", task.DeviceName)
}

//func findPointConfigById(points []device.PointConfig, id string) *device.PointConfig {
//	for idx, p := range points {
//		if p.Id == id {
//			return &points[idx]
//		}
//	}
//	return nil
//}

// ----批量（按寄存器区间聚合）采集主逻辑----

func (b *ModbusBus) doBatchCollect() {
	for _, dev := range b.Devices {
		// 按功能码分组与连续区间聚合采集
		batchGroups := groupPointsByFuncAndRegion(dev.Cfg.Points)
		for _, group := range batchGroups {
			readMap, err := dev.Adapter.BatchRead(group.Func, group.StartAddr, group.Quantity)
			if err != nil {
				fmt.Printf("batch read err from %s: %v\n", dev.Cfg.Name, err)
				continue
			}
			// 按分组内偏移映射到各点
			for _, pt := range group.Points {
				val := parseValueFromBatch(readMap, pt)
				fmt.Printf("Device %s point %s = %v\n", dev.Cfg.Name, pt.Name, val)
			}
		}
	}
}

// 分组结构
type BatchGroup struct {
	Func      string
	StartAddr uint16
	Quantity  uint16
	Points    []device.PointConfig // 分组覆盖点
}

// 聚合分组算法（可按实际支持优化，只为展示结构！）
func groupPointsByFuncAndRegion(points []device.PointConfig) []BatchGroup {
	type key struct{ Func string }
	groups := map[key][]device.PointConfig{}
	for _, pt := range points {
		if pt.Rw == "r" || pt.Rw == "rw" {
			k := key{Func: pt.Params["func"].(string)}
			groups[k] = append(groups[k], pt)
		}
	}
	var batchs []BatchGroup
	for k, groupPoints := range groups {
		// 按地址排序+区段聚合
		sort.Slice(groupPoints, func(i, j int) bool {
			return groupPoints[i].Params["address"].(int) < groupPoints[j].Params["address"].(int)
		})
		if len(groupPoints) == 0 {
			continue
		}
		start, end := groupPoints[0].Params["address"].(int), groupPoints[0].Params["address"].(int)
		seg := []device.PointConfig{groupPoints[0]}
		for i := 1; i < len(groupPoints); i++ {
			addr := groupPoints[i].Params["address"].(int)
			if addr > end+1 { // 遇见断档, 划分新区间
				batchs = append(batchs, BatchGroup{
					Func: k.Func, StartAddr: uint16(start),
					Quantity: uint16(end - start + 1), Points: append([]device.PointConfig{}, seg...),
				})
				start = addr
				seg = seg[:0]
			}
			end = addr
			seg = append(seg, groupPoints[i])
		}
		batchs = append(batchs, BatchGroup{
			Func: k.Func, StartAddr: uint16(start),
			Quantity: uint16(end - start + 1), Points: append([]device.PointConfig{}, seg...),
		})
	}
	return batchs
}

// 假定协议适配器支持批量读取接口
// (实际应直接调用Grid-x Modbus的ReadHoldingRegisters地址区间)
func parseValueFromBatch(batch interface{}, pt device.PointConfig) interface{} {
	// 写转换、解包，完成解析
	return nil
}

func (b *ModbusBus) Stop() {
	close(b.quitQ)
	b.wg.Wait()
}
