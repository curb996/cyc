package protocol

import (
	"fmt"
	"sync"
)

type AdapterFactory func(cfg map[string]interface{}) (ProtocolAdapter, error)

var (
	registry = sync.Map{} // map[string]AdapterFactory
)

func Register(adapterName string, factory AdapterFactory) {
	registry.Store(adapterName, factory)
}

func GetAdapter(adapterName string, cfg map[string]interface{}) (ProtocolAdapter, error) {
	f, ok := registry.Load(adapterName)
	if !ok {
		return nil, fmt.Errorf("protocol adapter not found: %v", adapterName)
	}
	return f.(AdapterFactory)(cfg)
}
