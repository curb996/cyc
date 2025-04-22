package protocol

import (
	"fmt"
	"sync"
)

type AdapterFactory func(cfg map[string]interface{}) (ProtocolAdapter, error)

var (
	registry = sync.Map{} // map[string]AdapterFactory
)

func Register(protocol string, factory AdapterFactory) {
	registry.Store(protocol, factory)
}

func GetAdapter(protocol string, cfg map[string]interface{}) (ProtocolAdapter, error) {
	f, ok := registry.Load(protocol)
	if !ok {
		return nil, fmt.Errorf("protocol adapter not found: %v", protocol)
	}
	return f.(AdapterFactory)(cfg)
}
