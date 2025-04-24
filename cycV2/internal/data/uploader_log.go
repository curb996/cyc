package data

import (
	"log"
)

type LogDispatcher struct{}

func (l *LogDispatcher) Dispatch(deviceName string, points map[string]interface{}) error {
	for k, v := range points {
		log.Printf("[分发] 设备:%s 数据对象:%s 数据值:%v\n", deviceName, k, v)
	}

	return nil
}

func init() {
	Register("log", &LogDispatcher{})
}
