package modbus

import (
	"testing"
)

func defaultTCPConfig() map[string]interface{} {
	return map[string]interface{}{
		"mode":       "tcp",
		"addr":       "127.0.0.1:502", // 可替换为本地模拟器
		"timeout_ms": 2000,
	}
}

func TestModbusConnectAndReadHR(t *testing.T) {
	cfg := defaultTCPConfig()
	adapter, err := NewModbusAdapter(cfg)
	if err != nil {
		t.Fatalf("adapter create failed: %v", err)
	}
	if err := adapter.Connect(); err != nil {
		t.Fatalf("connect failed: %v", err)
	}
	defer adapter.Disconnect()

	params := map[string]interface{}{
		"slave_id": 1,
		"func":     "hr",
		"address":  0,
		"quantity": 2,
	}
	data, err := adapter.Read("", params)
	if err != nil {
		t.Fatalf("modbus read hr failed: %v", err)
	}
	t.Logf("holding register(0x0000~0x0001): %x", data)
}

func TestModbusConnectAndWriteHR(t *testing.T) {
	cfg := defaultTCPConfig()
	adapter, err := NewModbusAdapter(cfg)
	if err != nil {
		t.Fatalf("adapter create failed: %v", err)
	}
	if err := adapter.Connect(); err != nil {
		t.Fatalf("connect failed: %v", err)
	}
	defer adapter.Disconnect()

	params := map[string]interface{}{
		"slave_id": 1,
		"func":     "hr",
		"address":  1,
		"quantity": 1,
	}
	// 写0x0002寄存器单个register，内容0x1234
	data := []byte{0x12, 0x34}
	if err := adapter.Write("", data, params); err != nil {
		t.Fatalf("modbus write hr failed: %v", err)
	}
	t.Logf("write success.")
}

func TestModbusConnectAndReadCoil(t *testing.T) {
	cfg := defaultTCPConfig()
	adapter, err := NewModbusAdapter(cfg)
	if err != nil {
		t.Fatalf("adapter create failed: %v", err)
	}
	if err := adapter.Connect(); err != nil {
		t.Fatalf("connect failed: %v", err)
	}
	defer adapter.Disconnect()

	params := map[string]interface{}{
		"slave_id": 1,
		"func":     "co",
		"address":  0,
		"quantity": 8,
	}
	data, err := adapter.Read("", params)
	if err != nil {
		t.Fatalf("modbus read coil failed: %v", err)
	}
	t.Logf("coil(0~7): % 08b", data) // Coil状态
}

func TestCreateTCP(t *testing.T) {
	cfg := map[string]interface{}{
		"mode":    "tcp",
		"addr":    "127.0.0.1:502",
		"slaveId": 1,
	}
	m, err := NewModbusAdapter(cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer m.Disconnect()
}

func TestReadWriteRegisters(t *testing.T) {
	cfg := map[string]interface{}{
		"mode":    "tcp",
		"addr":    "127.0.0.1:502",
		"slaveId": 1,
	}
	m, _ := NewModbusAdapter(cfg)

	defer m.Disconnect()
	// 假设已联通并有寄存器可写
	writeParams := map[string]interface{}{
		//"slave_id": 1,
		"func":     "hr",
		"address":  0,
		"quantity": 2,
	}
	writeData := []byte{0x12, 0x34, 0x00, 0x06}
	if err := m.Write("", writeData, writeParams); err != nil {
		t.Errorf("Write err: %v", err)
	}
	readParams := writeParams
	readRet, err := m.Read("", readParams)
	if err != nil {
		t.Errorf("Read err: %v", err)
	}
	t.Logf("Read data: %#v", readRet)
}
