package device

import "time"

type RawPoint struct {
	PointCfg PointConfig
	Bytes    []byte
	Err      error
}

//type RawDeviceData struct {
//	Device    *ModbusDevice // 哪个设备
//	RawPoints []RawPoint    // 每个点的原始采集结果
//}

type RawCollectResult struct {
	DeviceName string
	RawPoints  map[string]interface{} // key: PointConfig.Id，value: RawPoint
	Timestamp  time.Time
}
