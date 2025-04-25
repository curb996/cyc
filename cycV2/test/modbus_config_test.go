package test

import (
	"cycV2/internal/protocol/modbus"
	"fmt"
	"testing"
)

func TestModbusConfig(t *testing.T) {
	demo := map[string]interface{}{
		"mode":       "tcp",
		"address":    "127.0.0.1:502",
		"slaveId":    2,
		"timeoutMs":  1000,
		"serialPort": "/dev",
		"baudRate":   9600,
		"dataBits":   8,
		"parity":     "N",
		"stopBits":   1,
		"desc":       "测试设备TCP",
	}
	cfg, err := modbus.MapToModbusConfig(demo)
	if err != nil {
		panic(err)
	}
	fmt.Printf("%+v\n", cfg)

	m, err := modbus.ModbusConfigToMap(cfg)
	if err != nil {
		panic(err)
	}
	fmt.Printf("%#v\n", m)

}
