// test/modbus_pipeline_test.go
package test

import (
	"cycV2/internal/device" // 这里import你实际的包路径
	"log"
	"testing"
	"time"
)

func TestModbusPipelineAndReload(t *testing.T) {
	// 1. 构造Manager与配置
	mgr := device.NewManager("./test/devices.json") // 修改为你实际json
	err := mgr.ReloadFromFile()
	if err != nil {
		t.Fatal(err)
	}

	// 2. 启动解析worker
	stopParse := make(chan struct{})
	wgParse := device.StartParseWorkerPool(
		mgr.RawCh,
		2,
		func(dev string, points map[string]interface{}) {
			log.Printf("[解析:%s] %+v", dev, points)
		},
		stopParse,
	)

	// 3. 运行一段时间观测结果
	time.Sleep(3 * time.Second)

	// 4. 模拟reload热加载
	err = mgr.ReloadFromFile()
	if err != nil {
		t.Error("Reload after 3s failed: ", err)
	}
	time.Sleep(2 * time.Second)

	// 5. 优雅关闭
	for _, ch := range mgr.BusStop { // BusStops方法应返回所有bus stopCh
		close(ch)
	}
	close(stopParse)
	wgParse.Wait()
	log.Println("Test finished")
}
