package protocol

// ProtocolAdapter 协议适配器基础接口
type ProtocolAdapter interface {
	Connect() error
	Disconnect() error
	Read(address string, params map[string]interface{}) ([]byte, error)
	Write(address string, data []byte, params map[string]interface{}) error
	// 支持订阅/推送型协议也可增加回调注册等
}
