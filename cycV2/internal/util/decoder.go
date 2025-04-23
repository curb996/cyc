package util

import (
	"encoding/binary"
	"errors"
	"math"
)

func DecodeRegisterValue(raw []byte, typ, byteOrder string, swapReg bool) (interface{}, error) {
	reg := make([]byte, len(raw))
	copy(reg, raw)
	// 按需交换高低寄存器
	if swapReg && len(reg) == 4 {
		reg[0], reg[2] = reg[2], reg[0]
		reg[1], reg[3] = reg[3], reg[1]
	}
	var bo binary.ByteOrder
	switch byteOrder {
	case "little":
		bo = binary.LittleEndian
	case "big":
		bo = binary.BigEndian
	default:
		return nil, errors.New("unknown byte order")
	}

	switch typ {
	case "uint16":
		return bo.Uint16(reg), nil
	case "int16":
		return int16(bo.Uint16(reg)), nil
	case "uint32":
		return bo.Uint32(reg), nil
	case "float32":
		bits := bo.Uint32(reg)
		return math.Float32frombits(bits), nil
	case "bool":
		return reg[0] != 0, nil
	default:
		return nil, errors.New("unsupported type")
	}
}
