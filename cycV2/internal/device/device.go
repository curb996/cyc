// internal/device/device.go
package device

import (
	"cycV2/internal/driver"
)

type Device struct {
	Name   string                 `json:"name"`
	Driver driver.Driver          `json:"-"`
	Config map[string]interface{} `json:"config"`
	Points []Point                `json:"points"`
}

type Point struct {
	Id        string                 `json:"id"`         // 逻辑点名
	Desc      string                 `json:"desc"`       // 描述
	Params    map[string]interface{} `json:"params"`     // 协议抽象参数，func/address/quantity/unit_id/byte_order等
	Parse     string                 `json:"parse"`      // 解析方式（int16/float32等）
	SwapReg   bool                   `json:"swap_reg"`   // 多寄存器交换
	ByteOrder string                 `json:"byte_order"` // big/little
}

func (d *Device) Collect() (map[string]interface{}, error) {
	return d.Driver.Collect()
}
