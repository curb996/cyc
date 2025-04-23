package data

import (
	"log"
)

type LogDispatcher struct{}

func (l *LogDispatcher) Dispatch(deviceName string, points map[string]interface{}) error {
	log.Printf("[分发] 设备:%s 数据:%#v", deviceName, points)
	return nil
}

func init() {
	Register("log", &LogDispatcher{})
}
