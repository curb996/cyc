package device

import "cycV2/internal/protocol"

type BMSDevice struct {
	cfg     DeviceConfig
	adapter protocol.ProtocolAdapter
}

func NewBMSDevice(cfg DeviceConfig, adapter protocol.ProtocolAdapter) *BMSDevice {
	return &BMSDevice{cfg: cfg, adapter: adapter}
}

func (b *BMSDevice) Collect() (map[string]interface{}, error) {
	// 通过协议适配器采集
	_, err := b.adapter.Read(b.cfg.Addr, b.cfg.Params)
	if err != nil {
		return nil, err
	}

	// 解析协议数据为结构化map
	data := make(map[string]interface{})
	// ...解析raw填充data...
	return data, nil
}

func (b *BMSDevice) Control(action string, params map[string]interface{}) error {
	// 组包，调用adapter.Write
	return b.adapter.Write(b.cfg.Addr, []byte{ /*组包*/ }, params)
}

func (b *BMSDevice) GetName() string {
	return b.cfg.Name
}
