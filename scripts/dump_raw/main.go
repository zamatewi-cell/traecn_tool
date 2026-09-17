package main

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/zamatewi-cell/traecn_tool/internal/auth"
	"github.com/zamatewi-cell/traecn_tool/internal/config"
	"github.com/zamatewi-cell/traecn_tool/internal/device"
)

// 拉取原始 JSON 落盘, 供人工分析字段结构
func main() {
	a, err := auth.LoadTokenFromStorage(auth.DefaultStoragePath())
	if err != nil {
		fmt.Println("token:", err)
		os.Exit(1)
	}
	dev := device.NewDeviceInfo()
	client := &http.Client{}

	do := func(path string, body []byte, extra map[string]string) []byte {
		req, _ := http.NewRequest("POST", config.AgentDomain+path, bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set(config.HeaderIDEToken, a.Token)
		req.Header.Set("User-Agent", "TraeClient/TTNet")
		req.Header.Set("app-version", config.IDEVersion)
		req.Header.Set("package-type", "stable_cn")
		req.Header.Set("X-App-Id", config.AppID)
		for k, v := range dev.Headers() {
			req.Header.Set(k, v)
		}
		for k, v := range extra {
			req.Header.Set(k, v)
		}
		resp, err := client.Do(req)
		if err != nil {
			fmt.Println("ERR", path, err)
			return nil
		}
		defer resp.Body.Close()
		raw, _ := io.ReadAll(resp.Body)
		fmt.Printf("%s -> HTTP %d, %d bytes\n", path, resp.StatusCode, len(raw))
		return raw
	}

	if raw := do(config.EndpointModelList, []byte(`{"type":"llm_raw_chat"}`), map[string]string{"get-svc": "1"}); raw != nil {
		os.WriteFile(`scripts\probe4023\model_list_raw.json`, raw, 0o644)
	}
	if raw := do(config.EndpointGetDetailParam, []byte(`{"function":"chat_v3","need_prompt":false,"poly_prompt":false}`), nil); raw != nil {
		os.WriteFile(`scripts\probe4023\detail_param_chat_v3.json`, raw, 0o644)
	}
}
