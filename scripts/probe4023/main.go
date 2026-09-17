package main

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/zamatewi-cell/traecn_tool/internal/auth"
	"github.com/zamatewi-cell/traecn_tool/internal/config"
	"github.com/zamatewi-cell/traecn_tool/internal/device"
)

// masticateKey is the hardcoded AES-256 key found in Trae's server.js
// (module wMr, function exported as "masticateVegetablesToPulp").
const masticateKeyHex = "6195f24ca4d430f8a4833de7db8dac37d148a084e7464a351ffa68585c16b955"

// masticate encrypts plaintext exactly like the client does:
// key[:8] ^= pin; AES-256-GCM with random 12B IV; AAD = decimal requestAt;
// message = base64(iv || ciphertext || tag).
func masticate(plaintext []byte) (message, pin string, requestAt int64, err error) {
	key, err := hex.DecodeString(masticateKeyHex)
	if err != nil {
		return "", "", 0, err
	}
	pinB := make([]byte, 8)
	if _, err = rand.Read(pinB); err != nil {
		return "", "", 0, err
	}
	for i := 0; i < 8; i++ {
		key[i] ^= pinB[i]
	}
	iv := make([]byte, 12)
	if _, err = rand.Read(iv); err != nil {
		return "", "", 0, err
	}
	requestAt = time.Now().Unix()
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", "", 0, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", "", 0, err
	}
	ct := gcm.Seal(nil, iv, plaintext, []byte(strconv.FormatInt(requestAt, 10)))
	out := append(append([]byte{}, iv...), ct...)
	return base64.StdEncoding.EncodeToString(out), hex.EncodeToString(pinB), requestAt, nil
}

type modelListResp struct {
	ModelConfigs []struct {
		Name         string `json:"name"`
		ConfigName   string `json:"config_name"`
		DisplayName  string `json:"display_name"`
		PromptMaxTok int    `json:"prompt_max_tokens"`
	} `json:"model_configs"`
}

func dump(label string, resp *http.Response) {
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if label == "v1 preset deepseek-V3" {
		os.WriteFile("scripts/tmp_full_stream.txt", body, 0644)
	}
	preview := strings.ReplaceAll(string(body), "\n", "\\n")
	if len(preview) > 600 {
		preview = preview[:600]
	}
	fmt.Printf("%-46s HTTP %d  %s\n", label, resp.StatusCode, preview)
}

func main() {
	a, err := auth.LoadTokenFromStorage(auth.DefaultStoragePath())
	if err != nil {
		fmt.Println("token:", err)
		os.Exit(1)
	}
	dev := device.NewDeviceInfo()
	client := &http.Client{Timeout: 60 * time.Second}

	do := func(path string, body []byte, extra map[string]string) *http.Response {
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
		return resp
	}

	// 1. model_list type=llm_raw_chat —— 真实客户端拉取 agent 模型列表的方式
	resp := do(config.EndpointModelList, []byte(`{"type":"llm_raw_chat"}`), map[string]string{"get-svc": "1"})
	if resp == nil {
		return
	}
	raw, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	fmt.Printf("model_list HTTP %d bytes=%d\n", resp.StatusCode, len(raw))
	os.WriteFile("scripts/tmp_model_list.json", raw, 0644)
	var ml modelListResp
	json.Unmarshal(raw, &ml)
	names := []string{}
	for i, m := range ml.ModelConfigs {
		if i < 12 {
			fmt.Printf("  [%d] name=%q config=%q display=%q\n", i, m.Name, m.ConfigName, m.DisplayName)
		}
		if m.Name != "" {
			names = append(names, m.Name)
		}
	}
	if len(names) == 0 {
		fmt.Println("no models; raw preview:")
		fmt.Println(string(raw[:min(len(raw), 800)]))
		return
	}

	// 2. 真实 llm_raw_chat 形状: {"model_name":..., "message": <masticate(messages)>}
	messagesJSON, _ := json.Marshal([]map[string]interface{}{
		{"role": "user", "content": []map[string]string{{"type": "text", "text": "say hi in one word"}}},
	})

	tryReal := func(label, path, modelName string) {
		msg, pin, at, err := masticate(messagesJSON)
		if err != nil {
			fmt.Println("masticate:", err)
			return
		}
		body, _ := json.Marshal(map[string]interface{}{
			"model_name": modelName,
			"message":    msg,
		})
		r := do(path, body, map[string]string{
			"get-svc":        "1",
			"X-Request-Pin":  pin,
			"X-Requested-At": strconv.FormatInt(at, 10),
		})
		if r != nil {
			dump(label, r)
		}
	}

	// builder=true(预设) vs client_connect=true(自定义) 对照
	tryReal("v1 preset deepseek-V3", config.EndpointLLMRawChat, "deepseek-V3")
	tryReal("v1 preset seed_m8", config.EndpointLLMRawChat, "seed_m8")

	// v4-flash 4023 排查：尝试补充字段
	tryVariant := func(label string, extra map[string]interface{}) {
		msg, pin, at, err := masticate(messagesJSON)
		if err != nil {
			fmt.Println("masticate:", err)
			return
		}
		bodyMap := map[string]interface{}{"model_name": "deepseek-v4-flash", "message": msg}
		for k, v := range extra {
			bodyMap[k] = v
		}
		body, _ := json.Marshal(bodyMap)
		r := do(config.EndpointLLMRawChat, body, map[string]string{
			"get-svc":        "1",
			"X-Request-Pin":  pin,
			"X-Requested-At": strconv.FormatInt(at, 10),
		})
		if r != nil {
			dump(label, r)
		}
	}
	tryVariant("v4-flash base", nil)
	tryVariant("v4-flash +custom_model_id", map[string]interface{}{"custom_model_id": 2467803906})

	tryVariant2 := func(label, modelName string, extra map[string]interface{}) {
		msg, pin, at, err := masticate(messagesJSON)
		if err != nil {
			fmt.Println("masticate:", err)
			return
		}
		bodyMap := map[string]interface{}{"model_name": modelName, "message": msg}
		for k, v := range extra {
			bodyMap[k] = v
		}
		body, _ := json.Marshal(bodyMap)
		r := do(config.EndpointLLMRawChat, body, map[string]string{
			"get-svc":        "1",
			"X-Request-Pin":  pin,
			"X-Requested-At": strconv.FormatInt(at, 10),
		})
		if r != nil {
			dump(label, r)
		}
	}
	tryVariant2("v4-pro base", "deepseek-v4-pro", nil)
	tryVariant2("v4-pro +custom_model_id", "deepseek-v4-pro", map[string]interface{}{"custom_model_id": 2608953602})

	// 对照: model_list type=chat 是否返回不同模型集合
	if r := do(config.EndpointModelList, []byte(`{"type":"chat"}`), map[string]string{"get-svc": "1"}); r != nil {
		raw, _ := io.ReadAll(r.Body)
		r.Body.Close()
		var ml2 modelListResp
		json.Unmarshal(raw, &ml2)
		fmt.Printf("model_list type=chat: %d entries\n", len(ml2.ModelConfigs))
		for i, m := range ml2.ModelConfigs {
			if i < 15 {
				fmt.Printf("  chat[%d] name=%q display=%q\n", i, m.Name, m.DisplayName)
			}
		}
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
