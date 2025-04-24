package device

import (
	"cycV2/internal/protocol/modbus"
	"testing"
	"time"
)

func TestStartParseWorkerPool(t *testing.T) {

	// 1. 配置两个设备（不同slaveId即可区分）
	dev1Cfg := DeviceConfig{
		BusId:       "1",
		Name:        "device1",
		SlaveId:     1,
		IpAddr:      "127.0.0.1:502",
		AdapterName: "modbus",
		IntervalMs:  1000,
		Points: []PointConfig{
			{
				Name:      "Volatage",
				FuncCode:  "hr",
				RegAddr:   1,
				RegNum:    2,
				Desc:      "电压",
				Rw:        "r",
				ByteOrder: "big",
				DataType:  "int16",
				Params: map[string]interface{}{
					"func":     "hr",
					"address":  1,
					"quantity": 1,
				},
			},
		},
	}
	dev2Cfg := DeviceConfig{
		BusId:       "1",
		Name:        "device2",
		SlaveId:     2,
		IpAddr:      "127.0.0.1:502",
		AdapterName: "modbus",
		IntervalMs:  5000,
		Points: []PointConfig{
			{
				Name:      "Current",
				FuncCode:  "hr",
				RegAddr:   1,
				RegNum:    2,
				Desc:      "电流",
				Rw:        "r",
				ByteOrder: "big",
				DataType:  "int16",
				Params: map[string]interface{}{
					"func":     "hr",
					"address":  1,
					"quantity": 1,
				},
			},
		},
	}

	// 2. 实例化协议适配器
	adapter1, err := modbus.NewModbusAdapter(map[string]interface{}{
		"mode":      "tcp",
		"address":   "127.0.0.1:502",
		"slaveId":   1,
		"timeoutMs": 1000,
	})
	if err != nil {
		t.Fatalf("adapter1 err: %v", err)
	}
	adapter2, err := modbus.NewModbusAdapter(map[string]interface{}{
		"mode":      "tcp",
		"address":   "127.0.0.1:502",
		"slaveId":   2,
		"timeoutMs": 1000,
	})
	if err != nil {
		t.Fatalf("adapter2 err: %v", err)
	}

	// 3. 注入设备对象
	d1 := &ModbusDevice{
		Cfg:     dev1Cfg,
		Adapter: adapter1,
	}
	d2 := &ModbusDevice{
		Cfg:     dev2Cfg,
		Adapter: adapter2,
	}

	devices1 := []*ModbusDevice{d1}
	devices2 := []*ModbusDevice{d2}

	rawCh := make(chan RawCollectResult, 100)
	stopCh := make(chan struct{})

	StartCollectPipeline(devices1, rawCh, stopCh)
	StartCollectPipeline(devices2, rawCh, stopCh)

	//wg := StartParseWorkerPool(rawCh, 4, func(dev string, parsed map[string]interface{}) {
	//	// 你的上传、落库、转发等
	//	fmt.Printf("[%s] 解析结果: %+v\n", dev, parsed)
	//}, stopCh)

	wg := StartParseWorkerPool(rawCh, 4, parsedHandler, stopCh)

	time.Sleep(time.Hour)
	// 示例：某个时机可调用
	close(stopCh)
	wg.Wait() //安全回收
}
