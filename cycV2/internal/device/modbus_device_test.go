// internal/device/modbus_device_test.go
package device

import (
	"testing"
)

type mockAdapter struct{}

func (m *mockAdapter) Read(params map[string]interface{}) ([]byte, error) {
	//TODO implement me
	//panic("implement me")
	return []byte{0x41, 0x20, 0x00, 0x00}, nil // float32=10.0
}

func (m *mockAdapter) BatchRead(funcCode string, startAddr, quantity uint16) ([]byte, error) {
	//TODO implement me
	//panic("implement me")
	return []byte{0x41, 0x20, 0x00, 0x00}, nil // float32=10.0
}

func (m *mockAdapter) WriteModbus(funcCode string, addr uint16, value []byte) error {
	//TODO implement me
	//panic("implement me")
	return nil // float32=10.0
}

func (m *mockAdapter) Write(_ string, _ []byte, _ map[string]interface{}) error { return nil }

func (m *mockAdapter) Connect() error {
	//TODO implement me
	panic("implement me")
}

func (m *mockAdapter) Disconnect() error {
	//TODO implement me
	panic("implement me")
}

func TestModbusDevice_CollectAllParallel(t *testing.T) {
	cfg := DeviceConfig{
		Name:     "dev1",
		Protocol: "modbus",
		Points: []PointConfig{
			{Name: "volt", DataType: "float32", Rw: "r"},
			{Name: "curr", DataType: "float32", Rw: "r"},
		},
	}
	dev := NewModbusDevice(cfg, &mockAdapter{})
	data, err := dev.CollectAllParallel()
	if err != nil {
		t.Fatalf("Collect failed: %v", err)
	}
	if data["volt"] != float32(10.0) {
		t.Errorf("expect volt==10.0, got %v", data["volt"])
	}
}

func TestModbusDevice_ControlAsync(t *testing.T) {
	cfg := DeviceConfig{Name: "dev1"}
	dev := NewModbusDevice(cfg, &mockAdapter{})
	errCh := dev.ControlAsync("volt", []byte{0, 1}, map[string]interface{}{})
	err := <-errCh
	if err != nil {
		t.Fatalf("ControlAsync failed: %v", err)
	}
}

// 测试批量设备管理
func TestDeviceManager_CollectAll(t *testing.T) {
	mgr := NewDeviceManager()
	cfg := DeviceConfig{
		Name:     "dev1",
		Protocol: "modbus",
		Points: []PointConfig{
			{Name: "volt", DataType: "float32", Rw: "r"},
		},
	}
	err := mgr.Register(cfg, &mockAdapter{})
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}
	all := mgr.CollectAll()
	if v, ok := all["dev1"]["volt"].(float32); !ok || v != 10.0 {
		t.Fatalf("CollectAll error, got %v", all)
	}
}
