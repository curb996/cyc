package test

import (
	"cycV2/internal/device"
	"cycV2/internal/protocol/modbus"
	"sync"
	"testing"
	"time"
)

func TestConcurrentCollectMultipleModbusDevices(t *testing.T) {
	// 1. 配置两个设备（不同slaveId即可区分）
	dev1Cfg := device.DeviceConfig{
		Name:        "bms1",
		SlaveId:     1,
		IpAddr:      "127.0.0.1:502",
		AdapterName: "modbus",
		Points: []device.PointConfig{
			{
				Name:      "voltage",
				FuncCode:  "hr",
				RegAddr:   0,
				Desc:      "总压",
				Rw:        "r",
				ByteOrder: "big",
			},
		},
	}
	dev2Cfg := device.DeviceConfig{
		Name:        "bms2",
		SlaveId:     2,
		IpAddr:      "127.0.0.1:502",
		AdapterName: "modbus",
		Points: []device.PointConfig{
			{
				Name:      "voltage",
				FuncCode:  "hr",
				RegAddr:   1,
				Desc:      "总压",
				Rw:        "r",
				ByteOrder: "big",
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
	devObj1 := &device.ModbusDevice{
		Cfg:     dev1Cfg,
		Adapter: adapter1,
	}
	devObj2 := &device.ModbusDevice{
		Cfg:     dev2Cfg,
		Adapter: adapter2,
	}

	// 4. 并发采集
	for {
		var wg sync.WaitGroup
		wg.Add(2)
		for _, dev := range []*device.ModbusDevice{devObj1, devObj2} {
			go func(d *device.ModbusDevice) {
				defer wg.Done()
				res, err := d.Collect()
				if err != nil {
					t.Logf("[Device: %s] Collect error: %v", d.Cfg.Name, err)
				} else {
					t.Logf("[Device: %s] Collect data: %#v", d.Cfg.Name, res)
				}
			}(dev)
		}
		wg.Wait()
		time.Sleep(time.Second)
	}

}
