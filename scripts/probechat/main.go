// Probe: function-matched emp + base64 user_input for agent task.
package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/zamatewi-cell/traecn_tool/internal/auth"
	"github.com/zamatewi-cell/traecn_tool/internal/config"
)

var captured map[string]string
var reqPin, reqAt string

func newReq(token, path string, body []byte) *http.Request {
	req, _ := http.NewRequest("POST", config.AgentDomain+path, bytes.NewReader(body))
	skip := map[string]bool{
		"content-length": true, "referer": true, "x-custom-repo-urls": true,
		"x-ide-token": true, "x-request-id": true, "x-trae-request-id": true,
		"x-custom-trace-id": true, "x-tt-trace-id": true, "x-requested-at": true,
		"accept-encoding": true, "x-request-pin": true,
	}
	for k, v := range captured {
		if !skip[k] {
			req.Header.Set(k, v)
		}
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-IDE-Token", token)
	if reqPin != "" {
		req.Header.Set("x-request-pin", reqPin)
	}
	if reqAt != "" {
		req.Header.Set("x-requested-at", reqAt)
	}
	return req
}

type detailResp struct {
	ConfigInfoList []struct {
		ConfigName      string `json:"config_name"`
		ModelDetailList []struct {
			EncryptedModelParams string `json:"encrypted_model_params"`
		} `json:"model_detail_list"`
	} `json:"config_info_list"`
}

func fetchEmp(client *http.Client, token, fn, model string) string {
	req := newReq(token, config.EndpointGetDetailParam, []byte(`{"function":"`+fn+`","need_prompt":true,"poly_prompt":true}`))
	resp, err := client.Do(req)
	if err != nil {
		return ""
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	var d detailResp
	json.Unmarshal(body, &d)
	for _, c := range d.ConfigInfoList {
		if c.ConfigName == model && len(c.ModelDetailList) > 0 {
			return c.ModelDetailList[0].EncryptedModelParams
		}
	}
	return ""
}

func main() {
	a, err := auth.LoadTokenFromStorage(auth.DefaultStoragePath())
	if err != nil {
		fmt.Println("load token failed:", err)
		os.Exit(1)
	}
	raw, _ := os.ReadFile("scripts/captured/agent_task_req_headers.json")
	json.Unmarshal(raw, &captured)
	client := &http.Client{Timeout: 60 * time.Second}
	model := "DeepSeek-V4-Flash"

	try := func(label, path string, payload map[string]interface{}) {
		body, _ := json.Marshal(payload)
		resp, err := client.Do(newReq(a.Token, path, body))
		if err != nil {
			fmt.Printf("%-44s ERR %v\n", label, err)
			return
		}
		data, _ := io.ReadAll(io.LimitReader(resp.Body, 2000))
		resp.Body.Close()
		preview := strings.ReplaceAll(string(data), "\n", "\\n")
		if len(preview) > 500 {
			preview = preview[:500]
		}
		fmt.Printf("%-44s -> %d  %s\n", label, resp.StatusCode, preview)
	}

	prompt := "用三个字回答:1+1=?"
	parts := []map[string]string{{"type": "text", "text": prompt}}
	partsJSON, _ := json.Marshal(parts)

	// A. emp fetched with function=chat
	encChat := fetchEmp(client, a.Token, "chat", model)
	var eo struct{ RequestPin, RequestAt string }
	json.Unmarshal([]byte(encChat), &eo)
	reqPin, reqAt = eo.RequestPin, eo.RequestAt
	fmt.Printf("emp(chat) pin=%q\n", reqPin)
	try("A. raw_chat + emp(chat)", "/api/ide/v1/llm_raw_chat", map[string]interface{}{
		"config_name": model, "encrypted_model_params": encChat, "stream": true,
		"function": "chat",
		"messages": []map[string]interface{}{{"role": "user", "content": parts}},
	})

	// B/C/D: agent task with emp from chat_v3
	encV3 := fetchEmp(client, a.Token, "chat_v3", model)
	var eo3 struct{ RequestPin, RequestAt string }
	json.Unmarshal([]byte(encV3), &eo3)
	reqPin, reqAt = eo3.RequestPin, eo3.RequestAt
	fmt.Printf("emp(chat_v3) pin=%q\n", reqPin)

	agentBase := map[string]interface{}{
		"conversation_id": "c1", "session_id": "s1",
		"task_id": "t1", "message_id": "m1",
		"user_id": a.UserID, "device_id": captured["x-device-id"],
		"model_name": model, "config_name": model, "encrypted_model_params": encV3,
		"stream": true, "function": "chat_v3", "agent_type": "chat",
		"ide_version": "3.3.37",
	}
	clone := func() map[string]interface{} {
		b, _ := json.Marshal(agentBase)
		var m map[string]interface{}
		json.Unmarshal(b, &m)
		return m
	}

	m := clone()
	m["user_input"] = map[string]string{"id": "u1", "content": base64.StdEncoding.EncodeToString(partsJSON)}
	try("B. agent user_input=b64(partsJSON)", "/api/agent/v3/create_agent_task", m)

	m = clone()
	m["user_input"] = map[string]string{"id": "u1", "content": base64.StdEncoding.EncodeToString([]byte(prompt))}
	try("C. agent user_input=b64(text)", "/api/agent/v3/create_agent_task", m)

	m = clone()
	m["user_input"] = map[string]interface{}{"id": "u1", "content": parts}
	try("D. agent user_input content=parts", "/api/agent/v3/create_agent_task", m)
}
