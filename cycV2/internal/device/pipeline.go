package device

import (
	"cycV2/internal/data"
	"fmt"
	"log"
	"sync"
	"time"
)

// 假定RawPoint和ModbusDevice已在device包内定义，例如：
// type RawPoint struct {
//     PointCfg PointConfig
//     Bytes    []byte
//     Err      error
// }
// type ModbusDevice struct {...}

//type RawCollectResult struct {
//	DeviceName string
//	RawPoints  map[string]interface{} // key: PointConfig.Id，value: RawPoint
//	Timestamp  time.Time
//}

// StartCollectPipeline 启动采集流水线：每台设备定时采集，将原始点映射送入out通道，不阻塞
//func StartCollectPipeline(devices []*ModbusDevice, out chan<- RawCollectResult, stopCh <-chan struct{}) {
//	for _, dev := range devices {
//		d := dev // goroutine闭包变量捕获
//		go func() {
//			interval := time.Duration(d.Cfg.IntervalMs) * time.Millisecond
//			if interval <= 0 {
//				interval = 1000 * time.Millisecond // 默认1s
//			}
//			ticker := time.NewTicker(interval)
//
//			defer ticker.Stop()
//			for {
//				select {
//				case <-ticker.C:
//					raw, err := d.Collect() // map[string]interface{}
//					if err != nil {
//						log.Printf("[采集流水线] 设备 %s 采集错误: %v", d.Cfg.Name, err)
//						continue
//					}
//					out <- RawCollectResult{
//						DeviceName: d.Cfg.Name,
//						RawPoints:  raw,
//						Timestamp:  time.Now(),
//					}
//				case <-stopCh:
//					log.Printf("停止采集设备 %s, pipeline退出", d.Cfg.Name)
//					return
//				}
//			}
//		}()
//	}
//}

// StartCollectPipeline 启动总线采集流水线：同一总线下仅一组worker顺序采集所有设备，避免串口冲突。
// 支持每设备配置独立采集间隔。
func StartCollectPipeline(devices []*ModbusDevice, out chan<- RawCollectResult, stopCh <-chan struct{}) {
	if len(devices) == 0 {
		return
	}
	go func() {
		// 每轮都遍历一遍devices，但按各自采集间隔（IntervalMs）决定是否采集
		lastCollect := make(map[string]time.Time)
		minInterval := 0
		for _, dev := range devices {
			fmt.Println("name:", dev.Cfg.Name, " intervalMs:", dev.Cfg.IntervalMs)
			lastCollect[dev.Cfg.Name] = time.Time{}
			//按照设备组中采集频率最快的能适应的采集频率进行采集
			if dev.Cfg.IntervalMs != 0 {
				if minInterval == 0 || minInterval > dev.Cfg.IntervalMs {
					minInterval = dev.Cfg.IntervalMs
				}
			}
		}
		if minInterval <= 0 {
			minInterval = 1000 // 默认1s
		}
		fmt.Println("###minInterval:", minInterval)
		interval := time.Duration(minInterval) * time.Millisecond
		ticker := time.NewTicker(interval)

		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				now := time.Now()
				for _, d := range devices {

					temp := d.Cfg.IntervalMs
					if temp <= 0 {
						temp = 1000 //默认1s
					}
					t := time.Duration(temp) * time.Millisecond

					// 距离上次采集是否到达间隔
					if now.Sub(lastCollect[d.Cfg.Name]) >= t { //防止同一组中有些设备需要慢点采集的需求
						raw, err := d.Collect()
						if err != nil {
							log.Printf("[采集流水线] 设备%s采集错误: %v", d.Cfg.Name, err)
							continue
						}
						out <- RawCollectResult{
							DeviceName: d.Cfg.Name,
							RawPoints:  raw,
							Timestamp:  now,
						}
						lastCollect[d.Cfg.Name] = now
					}
				}
			case <-stopCh:
				log.Printf("总线 worker 轮询退出, 设备: %v", deviceNames(devices))
				return
			}
		}
	}()
}

// 输出设备名列表辅助日志
func deviceNames(devs []*ModbusDevice) []string {
	names := make([]string, 0, len(devs))
	for _, d := range devs {
		names = append(names, d.Cfg.Name)
	}
	return names
}

// StartParseWorkerPool 启动多个解析worker，每个worker从in通道读出
// 并将解析后的点上传/存储/回调
func StartParseWorkerPool(
	in <-chan RawCollectResult,
	workerNum int,
	parsedHandler func(deviceName string, parsedPoints map[string]interface{}),
	stopCh <-chan struct{},
) *sync.WaitGroup { //主协程能安全等待所有采集线程处理完成后再退出 wg.Wait()安全回收
	var wg sync.WaitGroup
	for i := 0; i < workerNum; i++ {
		wg.Add(1)
		go func(workerIdx int) {
			defer wg.Done()
			for {
				select {
				case req := <-in:
					parsed := make(map[string]interface{}, len(req.RawPoints))
					for k, val := range req.RawPoints {
						// 原始数据应为RawPoint
						if rp, ok := val.(RawPoint); ok {
							parsed[k] = parseRaw(rp.Bytes, rp.PointCfg)
						} else {
							parsed[k] = val // fallback
						}
					}
					parsedHandler(req.DeviceName, parsed)
				case <-stopCh:
					log.Printf("解析worker(%d)退出", workerIdx)
					return
				}
			}
		}(i)
	}
	return &wg
}

// 打印式 parsedHandler 示例
func parsedHandler(deviceName string, parsedPoints map[string]interface{}) {
	//TODO 后面可以写入缓存、数据库/推送消息队列

	//TODO 测试http分发
	//dispathcher := data.GetDispatcherByName("http").(*data.HTTPDispatcher)
	//dispathcher.URL = "127.0.0.1"
	//err := dispathcher.Dispatch(deviceName, parsedPoints)
	//if err != nil {
	//	log.Printf("数据分发错误: %v", err)
	//}

	//TODO 测试默认日志打印
	err := data.GetDefaultDispatcher().Dispatch(deviceName, parsedPoints)
	if err != nil {
		log.Printf("数据分发错误: %v", err)
	}

}

// parseRaw 示例解析函数，请根据实际点类型扩展（如float32, int32, string等）
//func parseRaw(d []byte, pt PointConfig) interface{} {
//	switch pt.Type {
//	case "uint16":
//		if len(d) != 2 {
//			return "<error: invalid uint16 len>"
//		}
//		if pt.ByteOrder == "little" {
//			return int64(d[1])<<8 | int64(d[0])
//		}
//		return int64(d[0])<<8 | int64(d[1])
//	case "int16":
//		if len(d) != 2 {
//			return "<error: invalid int16 len>"
//		}
//		var v int16
//		if pt.ByteOrder == "little" {
//			v = int16(d[1])<<8 | int16(d[0])
//		} else {
//			v = int16(d[0])<<8 | int16(d[1])
//		}
//		return v
//	case "raw":
//		return d
//	default:
//		return d
//	}
//}

/* ==== 示例 main 测试入口 ====
func main() {
	devices := []*device.ModbusDevice{d1, d2, ...}
	rawCh := make(chan device.RawCollectResult, 100)
	stopCh := make(chan struct{})

	device.StartCollectPipeline(devices, rawCh, 100*time.Millisecond, stopCh)

	wg := device.StartParseWorkerPool(rawCh, 4, func(dev string, parsed map[string]interface{}) {
		// 你的上传、落库、转发等
		fmt.Printf("[%s] 解析结果: %+v\n", dev, parsed)
	}, stopCh)

	// 示例：某个时机可调用close(stopCh); wg.Wait()安全回收
}
*/
