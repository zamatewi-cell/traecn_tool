// probe_endpoints tests the surviving chat-ish endpoints (v1 chat, v2
// llm_raw_chat) with the masticate-encrypted request shape, which was never
// tried on them — only llm_raw_chat v1 got the encrypted treatment.
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
	"time"

	"github.com/zamatewi-cell/traecn_tool/internal/auth"
	"github.com/zamatewi-cell/traecn_tool/internal/config"
	"github.com/zamatewi-cell/traecn_tool/internal/device"
)

const masticateKeyHex = "6195f24ca4d430f8a4833de7db8dac37d148a084e7464a351ffa68585c16b955"

func masticate(plaintext []byte) (message, pin string, requestAt int64, err error) {
	key, _ := hex.DecodeString(masticateKeyHex)
	pinB := make([]byte, 8)
	rand.Read(pinB)
	for i := 0; i < 8; i++ {
		key[i] ^= pinB[i]
	}
	iv := make([]byte, 12)
	rand.Read(iv)
	requestAt = time.Now().Unix()
	block, _ := aes.NewCipher(key)
	gcm, _ := cipher.NewGCM(block)
	ct := gcm.Seal(nil, iv, plaintext, []byte(strconv.FormatInt(requestAt, 10)))
	out := append(append([]byte{}, iv...), ct...)
	return base64.StdEncoding.EncodeToString(out), hex.EncodeToString(pinB), requestAt, nil
}

func main() {
	a, err := auth.LoadTokenFromStorage(auth.DefaultStoragePath())
	if err != nil {
		fmt.Println("token:", err)
		os.Exit(1)
	}
	dev := device.NewDeviceInfo()
	client := &http.Client{Timeout: 60 * time.Second}

	messagesJSON, _ := json.Marshal([]map[string]interface{}{
		{"role": "user", "content": []map[string]string{{"type": "text", "text": "hi"}}},
	})

	try := func(label, path, modelName string) {
		msg, pin, at, err := masticate(messagesJSON)
		if err != nil {
			fmt.Println("masticate:", err)
			return
		}
		body, _ := json.Marshal(map[string]interface{}{"model_name": modelName, "message": msg})
		req, _ := http.NewRequest("POST", config.AgentDomain+path, bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set(config.HeaderIDEToken, a.Token)
		req.Header.Set("User-Agent", "TraeClient/TTNet")
		req.Header.Set("app-version", "3.3.96")
		req.Header.Set("x-ide-version", "3.3.96")
		req.Header.Set("package-type", "stable_cn")
		req.Header.Set("X-App-Id", config.AppID)
		for k, v := range dev.Headers() {
			req.Header.Set(k, v)
		}
		req.Header.Set("get-svc", "1")
		req.Header.Set("X-Request-Pin", pin)
		req.Header.Set("X-Requested-At", strconv.FormatInt(at, 10))
		resp, err := client.Do(req)
		if err != nil {
			fmt.Printf("%-52s ERR %v\n", label, err)
			return
		}
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 400))
		resp.Body.Close()
		preview := string(bytes.ReplaceAll(raw, []byte("\n"), []byte(`\n`)))
		fmt.Printf("%-52s HTTP %d  %s\n", label, resp.StatusCode, preview)
		time.Sleep(120 * time.Millisecond)
	}

	// 端点矩阵 × 代表性模型（内置 + 已知可用的预设作对照）
	endpoints := map[string]string{
		"v1 /chat":          "/api/ide/v1/chat",
		"v2 /llm_raw_chat":  "/api/ide/v2/llm_raw_chat",
		"v1 /llm_raw_chat(对照)": config.EndpointLLMRawChat,
	}
	models := []string{"Doubao-Seed-Code", "glm-5.2", "DeepSeek-V4-Flash", "seed_m8"}
	for elabel, epath := range endpoints {
		for _, m := range models {
			try(elabel+" "+m, epath, m)
		}
	}
}
