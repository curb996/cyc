package data

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

type HTTPDispatcher struct {
	URL string
}

func (h *HTTPDispatcher) Dispatch(deviceName string, points map[string]interface{}) error {

	fmt.Println("HTTPDispatcher...")
	record := map[string]interface{}{
		"device": deviceName,
		"points": points,
	}
	b, _ := json.Marshal(record)

	fmt.Println("json:", string(b))
	resp, err := http.Post(h.URL, "application/json", bytes.NewReader(b))
	if err != nil {
		return fmt.Errorf("HTTP上传失败: %w", err)
	}
	resp.Body.Close()
	return nil
}

func init() {
	Register("http", &HTTPDispatcher{})
}
