package device

import (
	"cycV2/internal/protocol"
	"encoding/binary"
	"fmt"
	"math"
	"sync"
)

type WriteTask struct {
	Id     string
	Data   []byte
	Params map[string]interface{}
	RespCh chan error
}

type ModbusDevice struct {
	Cfg        DeviceConfig
	Adapter    protocol.ProtocolAdapter
	mu         sync.Mutex
	writeQueue chan *WriteTask
}

// 构造器
func NewModbusDevice(cfg DeviceConfig, adapter protocol.ProtocolAdapter) *ModbusDevice {
	//return &ModbusDevice{Cfg: Cfg, Adapter: Adapter}
	dev := &ModbusDevice{
		Cfg:        cfg,
		Adapter:    adapter,
		writeQueue: make(chan *WriteTask, 100),
	}
	go dev.writeWorker()
	return dev
}

func (d *ModbusDevice) writeWorker() {
	for task := range d.writeQueue {
		d.mu.Lock()
		err := d.Adapter.Write(task.Id, task.Data, task.Params)
		d.mu.Unlock()
		if task.RespCh != nil {
			task.RespCh <- err
		}
	}
}

// 发起异步控制写入，结果通过chan返回
func (d *ModbusDevice) ControlAsync(id string, data []byte, params map[string]interface{}) <-chan error {
	task := &WriteTask{
		Id:     id,
		Data:   data,
		Params: params,
		RespCh: make(chan error, 1),
	}
	d.writeQueue <- task
	return task.RespCh
}

// 采集所有点数据
func (d *ModbusDevice) Collect() (map[string]interface{}, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	result := make(map[string]interface{})
	for _, pt := range d.Cfg.Points {
		param := mergeParams(d.Cfg.Params, pt.Params)
		raw, err := d.Adapter.Read(param)
		if err != nil {
			result[pt.Name] = fmt.Sprintf("read error: %v", err)
			continue
		}
		result[pt.Name] = RawPoint{PointCfg: pt, Bytes: raw, Err: err} //这里只进行采集，将原始数据传输出去进行解析
	}
	return result, nil
}

func (d *ModbusDevice) CollectAllParallel() (map[string]interface{}, error) {
	results := make(map[string]interface{})
	var wg sync.WaitGroup
	mu := sync.Mutex{}
	errs := []error{}
	for _, pt := range d.Cfg.Points {
		if pt.Rw == "w" { // 跳过只写点
			continue
		}
		wg.Add(1)
		ptCopy := pt
		go func() {
			defer wg.Done()
			d.mu.Lock()
			param := mergeParams(d.Cfg.Params, ptCopy.Params)
			raw, err := d.Adapter.Read(param)
			d.mu.Unlock()
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				errs = append(errs, fmt.Errorf("%s: %v", ptCopy.Name, err))
				results[ptCopy.Name] = nil
			} else {
				results[ptCopy.Name] = parseRaw(raw, ptCopy)
			}
		}()
	}
	wg.Wait()
	if len(errs) > 0 {
		return results, fmt.Errorf("some points failed: %v", errs)
	}
	return results, nil
}

// 控制命令
func (d *ModbusDevice) Control(id string, data []byte, params map[string]interface{}) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.Adapter.Write(id, data, params)
}

func mergeParams(global, point map[string]interface{}) map[string]interface{} {
	out := make(map[string]interface{})
	for k, v := range global {
		out[k] = v
	}
	for k, v := range point {
		out[k] = v
	}
	return out
}

// 简单的解析函数，可按点表配置扩展
func parseRaw(data []byte, pt PointConfig) interface{} {
	switch pt.DataType {
	case "float32":
		d := data
		if pt.SwapReg && len(d) == 4 {
			d = []byte{d[2], d[3], d[0], d[1]}
		}
		if len(d) != 4 {
			return fmt.Sprintf("invalid len %d for float32", len(d))
		}
		if pt.ByteOrder == "little" {
			return math.Float32frombits(binary.LittleEndian.Uint32(d))
		}
		return math.Float32frombits(binary.BigEndian.Uint32(d))
	case "int16":
		d := data
		if len(d) != 2 {
			return fmt.Sprintf("invalid len %d for int16", len(d))
		}
		if pt.ByteOrder == "little" {
			return int16(binary.LittleEndian.Uint16(d))
		}
		return int16(binary.BigEndian.Uint16(d))
	case "uint16":
		d := data
		if len(d) != 2 {
			return fmt.Sprintf("invalid len %d for uint16", len(d))
		}
		if pt.ByteOrder == "little" {
			return binary.LittleEndian.Uint16(d)
		}
		return binary.BigEndian.Uint16(d)
	default:
		return data // 默认返回原始数据
	}
}
