package protocol

// ProtocolAdapter 协议适配器基础接口
type ProtocolAdapter interface {
	Connect() error
	Disconnect() error
	Read(params map[string]interface{}) ([]byte, error)
	BatchRead(funcCode string, startAddr, quantity uint16) ([]byte, error)
	Write(address string, data []byte, params map[string]interface{}) error
	WriteModbus(funcCode string, addr uint16, value []byte) error
	//Write(funcCode string, slaveId uint8, addr uint16, value []byte) error

	// 支持订阅/推送型协议也可增加回调注册等
}
