package modbus

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"time"
	"unsafe"

	"cycV2/internal/protocol"
	"github.com/grid-x/modbus"

	"github.com/mitchellh/mapstructure"
)

// 协议注册
func init() {
	protocol.Register("modbus", NewModbusAdapter)
}

// ModbusConfig 兼容TCP和RTU，未用参数可省略配置
type ModbusConfig struct {
	//公用
	Mode      string `json:"mode"`      // "tcp" | "rtu"
	SlaveID   byte   `json:"slaveId"`   // 站号
	TimeoutMS int    `json:"timeoutMs"` // 超时时间，毫秒
	//TCP
	Address string `json:"address,omitempty"` // "127.0.0.1:502"  TCP模式用
	//RTU
	SerialPort string `json:"serialPort,omitempty"` //串口设备,仅RTU模式用
	BaudRate   int    `json:"baudRate,omitempty"`   //波特率
	DataBits   int    `json:"dataBits,omitempty"`   //数据位
	Parity     string `json:"parity,omitempty"`     // "N", "E", "O"
	StopBits   int    `json:"stopBits,omitempty"`   //停止位
}

// ModbusAdapter 实现 protocol.ProtocolAdapter 接口
type ModbusAdapter struct {
	handler modbus.ClientHandler
	client  modbus.Client
	config  map[string]interface{} //TODO,可以更改成具体modbus配置信息
	opened  bool                   //判断是否打开连接了
}

// NewModbusAdapter 工厂函数，根据配置创建 Modbus Adapter（支持 TCP/RTU）
func NewModbusAdapter(cfg map[string]interface{}) (protocol.ProtocolAdapter, error) {
	mode, _ := cfg["mode"].(string) // "tcp" / "rtu"
	addr, _ := cfg["address"].(string)
	slave := parseUint8(cfg["slaveId"], 1)

	timeout := time.Second * 2
	if to, ok := cfg["timeoutMs"].(int); ok && to > 0 {
		timeout = time.Duration(to) * time.Millisecond
	}
	fmt.Println("slaveId:", slave)

	switch mode {
	case "tcp":
		handler := modbus.NewTCPClientHandler(addr)
		handler.Timeout = timeout
		//handler.SetSlave(slave)
		handler.SlaveID = slave
		return &ModbusAdapter{
			handler: handler,
			config:  cfg,
			opened:  false,
		}, nil
	case "rtu":
		baud := 9600
		if v, ok := cfg["baudrate"].(int); ok && v > 0 {
			baud = v
		}
		dataBits := 8
		if v, ok := cfg["databits"].(int); ok && v > 0 {
			dataBits = v
		}
		stopBits := 1
		if v, ok := cfg["stopbits"].(int); ok && v > 0 {
			stopBits = v
		}
		parity := "N"
		if v, ok := cfg["parity"].(string); ok && v != "" {
			parity = v
		}
		handler := modbus.NewRTUClientHandler(addr)
		handler.BaudRate = baud
		handler.DataBits = dataBits
		handler.Parity = parity
		handler.StopBits = stopBits
		handler.Timeout = timeout
		//handler.SetSlave(slave)
		handler.SlaveID = slave
		return &ModbusAdapter{
			handler: handler,
			config:  cfg,
			opened:  false,
		}, nil
	default:
		return nil, errors.New("unsupported modbus mode, should be tcp or rtu")
	}
}

// Connect 实现协议连接
func (m *ModbusAdapter) Connect() error {
	if m.opened {
		return nil
	}
	err := m.handler.Connect()
	if err == nil {
		m.client = modbus.NewClient(m.handler)
		m.opened = true
	}
	return err
}

// Disconnect 用于断开连接
func (m *ModbusAdapter) Disconnect() error {
	if !m.opened {
		return nil
	}
	if err := m.handler.Close(); err != nil {
		return err
	}
	m.opened = false
	return nil
}

// Read 用于 Modbus 点读取
// params: slave_id, func (hr,ir,co,di), address, quantity
func (m *ModbusAdapter) Read(params map[string]interface{}) ([]byte, error) {
	if !m.opened {
		if err := m.Connect(); err != nil {
			return nil, fmt.Errorf("connect failed: %w", err)
		}
	}
	// 单元号 - 支持 int/float64/uint8（兼容 json 解码）
	//unit := parseUint8(params["slave_id"], 1)
	//m.setSlaveId(unit)
	// 功能码
	funcStr, _ := params["func"].(string)
	if funcStr == "" {
		funcStr = "hr"
	}
	address := parseUint16(params["address"], 0)
	quantity := parseUint16(params["quantity"], 1)

	//fmt.Println("funcStr:", funcStr, " address:", address, " quantity:", quantity)

	switch funcStr {
	case "hr":
		return m.client.ReadHoldingRegisters(address, quantity)
	case "ir":
		return m.client.ReadInputRegisters(address, quantity)
	case "co":
		return m.client.ReadCoils(address, quantity)
	case "di":
		return m.client.ReadDiscreteInputs(address, quantity)
	default:
		return nil, fmt.Errorf("unknown func type: %s", funcStr)
	}
}

func (m *ModbusAdapter) BatchRead(funcCode string, startAddr, quantity uint16) ([]byte, error) {
	// 如果你管理了多个unitId，需 SetSlave
	// 下面以最常用功能码举例
	switch funcCode {
	case "hr", "03": // holding registers
		return m.client.ReadHoldingRegisters(startAddr, quantity)
	case "ir", "04": // input registers
		return m.client.ReadInputRegisters(startAddr, quantity)
	case "co", "01": // coils
		return m.client.ReadCoils(startAddr, quantity)
	case "di", "02": // discrete inputs
		return m.client.ReadDiscreteInputs(startAddr, quantity)
	default:
		return nil, fmt.Errorf("not support funcCode: %s", funcCode)
	}
}

// Write 用于 Modbus 点写入
// params:  func (hr/co), address, quantity
func (m *ModbusAdapter) Write(_ string, data []byte, params map[string]interface{}) error {
	if !m.opened {
		if err := m.Connect(); err != nil {
			return fmt.Errorf("connect failed: %w", err)
		}
	}
	//unit := parseUint8(params["slave_id"], 1)
	//m.setSlaveId(unit)
	funcStr, _ := params["func"].(string)
	if funcStr == "" {
		funcStr = "hr"
	}
	address := parseUint16(params["address"], 0)
	quantity := parseUint16(params["quantity"], 1)
	// 注意：data长度与quantity需匹配
	switch funcStr {
	case "hr":
		if quantity == 1 {
			if len(data) != 2 {
				return errors.New("data length mismatch: should be 2 bytes for one register")
			}
			val := binary.BigEndian.Uint16(data) // 默认大端
			_, err := m.client.WriteSingleRegister(address, val)
			return err
		}
		if len(data) != int(quantity)*2 {
			return fmt.Errorf("data length mismatch: %d, expect %d", len(data), quantity*2)
		}
		_, err := m.client.WriteMultipleRegisters(address, quantity, data)
		return err
	case "co":
		if quantity == 1 {
			if len(data) != 1 {
				return errors.New("data length mismatch: should be 1 byte for one coil")
			}
			val := uint16(0x0000)
			if data[0] != 0 {
				val = 0xFF00
			}
			_, err := m.client.WriteSingleCoil(address, val)
			return err
		}
		_, err := m.client.WriteMultipleCoils(address, quantity, data)
		return err
	default:
		return fmt.Errorf("write not supported for func=%s", funcStr)
	}
}

func (m *ModbusAdapter) WriteModbus(funcCode string, addr uint16, data []byte) error {
	if !m.opened {
		if err := m.Connect(); err != nil {
			return fmt.Errorf("connect failed: %w", err)
		}
	}
	switch funcCode {
	case "co", "01": // single or multiple coil
		if len(data) == 2 {
			// 单线圈，data需能转uint16
			value := binary.BigEndian.Uint16(data)
			_, err := m.client.WriteSingleCoil(addr, value)
			return err
		} else {
			// 多线圈，Modbus要求bit packed
			quantity := uint16(len(data) * 8)
			_, err := m.client.WriteMultipleCoils(addr, quantity, data)
			return err
		}
	case "hr", "03": // single or multiple holding register
		if len(data) == 2 {
			value := binary.BigEndian.Uint16(data)
			_, err := m.client.WriteSingleRegister(addr, value)
			return err
		} else {
			quantity := uint16(len(data) / 2)
			_, err := m.client.WriteMultipleRegisters(addr, quantity, data)
			return err
		}
	default:
		return fmt.Errorf("unsupported funcCode: %s", funcCode)
	}
}

// 内部工具函数：设置从站号
func (m *ModbusAdapter) setSlaveId(salveId uint8) {
	switch h := m.handler.(type) {
	case *modbus.TCPClientHandler:
		h.SetSlave(salveId)
	case *modbus.RTUClientHandler:
		h.SetSlave(salveId)
	}
}

// ------- 参数类型转换工具 --------
// 兼容前端/配置json传int/float64
func parseUint8(raw interface{}, def uint8) uint8 {
	switch v := raw.(type) {
	case uint8:
		return v
	case int:
		return uint8(v)
	case float64:
		return uint8(v)
	case nil:
		return def
	default:
		return def
	}
}
func parseUint16(raw interface{}, def uint16) uint16 {
	switch v := raw.(type) {
	case uint16:
		return v
	case int:
		return uint16(v)
	case float64:
		return uint16(v)
	case nil:
		return def
	default:
		return def
	}
}

// ---- 辅助解析高低字交换和字节序可选 ----

func SwapRegisterWords(data []byte, swapReg bool) []byte {
	copyData := make([]byte, len(data))
	copy(copyData, data)
	if swapReg && len(copyData) == 4 {
		copyData[0], copyData[1], copyData[2], copyData[3] = copyData[2], copyData[3], copyData[0], copyData[1]
	}
	return copyData
}

func ParseFloat32(data []byte, byteOrder string, swapReg bool) float32 {
	v := SwapRegisterWords(data, swapReg)
	var u uint32
	if byteOrder == "little" {
		u = binary.LittleEndian.Uint32(v)
	} else {
		u = binary.BigEndian.Uint32(v)
	}
	return float32FromBits(u)
}

func float32FromBits(u uint32) float32 {
	return *(*float32)(unsafe.Pointer(&u))
}

func ParseUint16(data []byte, byteOrder string) uint16 {
	if byteOrder == "little" {
		return binary.LittleEndian.Uint16(data)
	}
	return binary.BigEndian.Uint16(data)
}

// MapToModbusConfig 将map转换为ModbusConfig结构体
func MapToModbusConfig(m map[string]interface{}) (*ModbusConfig, error) {
	var cfg ModbusConfig
	decoderConfig := &mapstructure.DecoderConfig{
		Metadata:         nil,
		Result:           &cfg,
		TagName:          "json", // 支持小驼峰的json标签
		WeaklyTypedInput: true,   // 支持字符串数值自动转类型
	}
	decoder, err := mapstructure.NewDecoder(decoderConfig)
	if err != nil {
		return nil, err
	}
	if err := decoder.Decode(m); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// ModbusConfigToMap 将ModbusConfig结构体转为map[string]interface{}
func ModbusConfigToMap(cfg *ModbusConfig) (map[string]interface{}, error) {
	byts, err := json.Marshal(cfg)
	if err != nil {
		return nil, err
	}
	var m map[string]interface{}
	if err := json.Unmarshal(byts, &m); err != nil {
		return nil, err
	}
	return m, nil
}
