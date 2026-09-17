// probe_builtin verifies which Trae CN BUILTIN models (the ones shown in the
// IDE model picker) are callable through /api/ide/v1/llm_raw_chat with the
// masticate-encrypted request shape, and whether the served model catalog is
// gated by the declared IDE version (3.3.37 vs the local install's 3.3.96).
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

type detailParamResp struct {
	ConfigInfoList []struct {
		ConfigName    string `json:"config_name"`
		ConfigSource  string `json:"config_source"`
		ConfigSwitch  bool   `json:"config_switch"`
		IsInvisible   bool   `json:"is_invisible_to_user"`
		DisplayConfig struct {
			DisplayName string `json:"display_name"`
		} `json:"display_config"`
		ModelDetailList []struct {
			ModelName string `json:"model_name"`
		} `json:"model_detail_list"`
	} `json:"config_info_list"`
}

var client = &http.Client{Timeout: 60 * time.Second}

func do(token string, dev *device.DeviceInfo, ideVersion, path string, body []byte, extra map[string]string) *http.Response {
	req, _ := http.NewRequest("POST", config.AgentDomain+path, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(config.HeaderIDEToken, token)
	req.Header.Set("User-Agent", "TraeClient/TTNet")
	req.Header.Set("app-version", ideVersion)
	req.Header.Set("x-ide-version", ideVersion)
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

func main() {
	a, err := auth.LoadTokenFromStorage(auth.DefaultStoragePath())
	if err != nil {
		fmt.Println("token:", err)
		os.Exit(1)
	}
	dev := device.NewDeviceInfo()
	detailBody := []byte(`{"function":"chat_v3","need_prompt":true,"poly_prompt":true}`)

	// ---- Phase 1: version gating on get_detail_param ----
	fmt.Println("=== Phase 1: get_detail_param under old vs new ide version ===")
	var newCatalog detailParamResp
	for _, ver := range []string{"3.3.37", "3.3.96"} {
		r := do(a.Token, dev, ver, config.EndpointGetDetailParam, detailBody, nil)
		if r == nil {
			continue
		}
		raw, _ := io.ReadAll(r.Body)
		r.Body.Close()
		var dp detailParamResp
		json.Unmarshal(raw, &dp)
		names := []string{}
		for _, c := range dp.ConfigInfoList {
			names = append(names, c.ConfigName)
		}
		fmt.Printf("  ide=%s HTTP %d configs=%d: %v\n", ver, r.StatusCode, len(names), names)
		if ver == "3.3.96" {
			newCatalog = dp
			os.WriteFile("scripts/detail_param_v3396.json", raw, 0644)
		}
	}

	// ---- Phase 2: version gating on model_list ----
	fmt.Println("=== Phase 2: model_list(type=llm_raw_chat) under old vs new ide version ===")
	type mlResp struct {
		ModelConfigs []struct {
			Name        string `json:"name"`
			DisplayName string `json:"display_name"`
			IsPreset    bool   `json:"is_preset"`
			Status      bool   `json:"status"`
		} `json:"model_configs"`
	}
	for _, ver := range []string{"3.3.37", "3.3.96"} {
		r := do(a.Token, dev, ver, config.EndpointModelList, []byte(`{"type":"llm_raw_chat"}`), map[string]string{"get-svc": "1"})
		if r == nil {
			continue
		}
		raw, _ := io.ReadAll(r.Body)
		r.Body.Close()
		var ml mlResp
		json.Unmarshal(raw, &ml)
		presets := []string{}
		for _, m := range ml.ModelConfigs {
			if m.IsPreset && m.Status {
				presets = append(presets, m.Name)
			}
		}
		fmt.Printf("  ide=%s HTTP %d entries=%d presets=%v\n", ver, r.StatusCode, len(ml.ModelConfigs), presets)
		if ver == "3.3.96" {
			os.WriteFile("scripts/model_list_v3396.json", raw, 0644)
		}
	}

	// ---- Phase 3: sweep every builtin config_name / model_name through
	// llm_raw_chat with the encrypted shape ----
	fmt.Println("=== Phase 3: llm_raw_chat sweep over builtin catalog (encrypted shape) ===")
	messagesJSON, _ := json.Marshal([]map[string]interface{}{
		{"role": "user", "content": []map[string]string{{"type": "text", "text": "hi"}}},
	})

	type candidate struct{ label, value string }
	seen := map[string]bool{}
	cands := []candidate{}
	for _, c := range newCatalog.ConfigInfoList {
		if c.ConfigName != "" && !seen[c.ConfigName] {
			seen[c.ConfigName] = true
			cands = append(cands, candidate{"config_name", c.ConfigName})
		}
		for _, md := range c.ModelDetailList {
			if md.ModelName != "" && !seen[md.ModelName] {
				seen[md.ModelName] = true
				cands = append(cands, candidate{"model_name", md.ModelName})
			}
		}
	}
	fmt.Printf("  %d unique candidates from v3.3.96 catalog\n", len(cands))

	ok, unknown, other := []string{}, []string{}, []string{}
	for _, cd := range cands {
		msg, pin, at, err := masticate(messagesJSON)
		if err != nil {
			fmt.Println("masticate:", err)
			return
		}
		body, _ := json.Marshal(map[string]interface{}{"model_name": cd.value, "message": msg})
		r := do(a.Token, dev, "3.3.96", config.EndpointLLMRawChat, body, map[string]string{
			"get-svc":        "1",
			"X-Request-Pin":  pin,
			"X-Requested-At": strconv.FormatInt(at, 10),
		})
		if r == nil {
			continue
		}
		raw, _ := io.ReadAll(io.LimitReader(r.Body, 4096))
		r.Body.Close()
		s := string(raw)
		status := "?"
		switch {
		case bytes.Contains(raw, []byte(`"code":4023`)):
			status = "4023 model unknown"
			unknown = append(unknown, cd.value)
		case bytes.Contains(raw, []byte("event: output")) || bytes.Contains(raw, []byte("event: metadata")):
			status = "OK (real stream)"
			ok = append(ok, cd.value)
		default:
			if len(s) > 160 {
				s = s[:160]
			}
			status = fmt.Sprintf("HTTP %d: %s", r.StatusCode, s)
			other = append(other, cd.value+" -> "+status)
		}
		fmt.Printf("  [%s] %-40s %s\n", cd.label, cd.value, status)
		time.Sleep(120 * time.Millisecond)
	}

	fmt.Printf("\n=== Summary ===\nOK: %v\n4023: %v\nother: %v\n", ok, unknown, other)
}
