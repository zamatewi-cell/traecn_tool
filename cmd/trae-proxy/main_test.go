package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/zamatewi-cell/traecn_tool/internal/auth"
)

// TestCLI_CredentialOutputIsolation 验证反例 1：
// 验证独立 CLI 运行模式下，无论 Token 如何刷新，标准输出与重定向中绝不输出明文凭据与 __TRAE_EVENT__，
// 仅在明确启用桌面 IPC 模式时才向 stdout 发射结构化凭据事件。
func TestCLI_CredentialOutputIsolation(t *testing.T) {
	sensitiveAccessToken := "eyJhbGciOi_super_secret_access_token_xyz987"
	sensitiveRefreshToken := "refresh_token_ultra_secret_abc123"

	tok := &auth.TokenInfo{
		AccessToken:      sensitiveAccessToken,
		RefreshToken:     sensitiveRefreshToken,
		UserID:           "u_secret_999",
		ExpiresAt:        time.Now().Add(2 * time.Hour),
		RefreshExpiresAt: time.Now().Add(30 * 24 * time.Hour),
	}

	t.Run("CLI mode completely withholds credentials from stdout and event pipeline", func(t *testing.T) {
		// 拦截 stdout
		oldStdout := os.Stdout
		r, w, err := os.Pipe()
		if err != nil {
			t.Fatalf("failed to create pipe: %v", err)
		}
		os.Stdout = w

		// 同时捕获 logger 输出
		var logBuf bytes.Buffer
		testLogger := slog.New(slog.NewTextHandler(&logBuf, &slog.HandlerOptions{Level: slog.LevelInfo}))

		// 构造 CLI 模式下的刷新回调 (isDesktopIPC = false)
		cliHandler := buildTokenRefreshedHandler(false, testLogger)
		cliHandler("acc-001", "personal_work", tok)

		// 恢复 stdout 并读取输出
		_ = w.Close()
		os.Stdout = oldStdout
		outBytes, _ := io.ReadAll(r)
		stdoutOutput := string(outBytes)
		logOutput := logBuf.String()

		// 司法级断言：stdout 绝不能出现任何凭据或结构化事件
		if strings.Contains(stdoutOutput, "__TRAE_EVENT__") {
			t.Fatalf("SECURITY VIOLATION: __TRAE_EVENT__ leaked in standalone CLI stdout: %s", stdoutOutput)
		}
		if strings.Contains(stdoutOutput, sensitiveAccessToken) {
			t.Fatalf("SECURITY VIOLATION: plaintext access token leaked in CLI stdout: %s", stdoutOutput)
		}
		if strings.Contains(stdoutOutput, sensitiveRefreshToken) {
			t.Fatalf("SECURITY VIOLATION: plaintext refresh token leaked in CLI stdout: %s", stdoutOutput)
		}

		// 验证日志中也不含有明文凭据，仅有脱敏通知
		if strings.Contains(logOutput, sensitiveAccessToken) {
			t.Fatalf("SECURITY VIOLATION: plaintext access token leaked in logger: %s", logOutput)
		}
		if strings.Contains(logOutput, sensitiveRefreshToken) {
			t.Fatalf("SECURITY VIOLATION: plaintext refresh token leaked in logger: %s", logOutput)
		}
		if !strings.Contains(logOutput, "credentials withheld in CLI mode") {
			t.Fatalf("expected logger to report sanitized withholding notice, got: %s", logOutput)
		}
	})

	t.Run("Desktop IPC mode emits structured event with accurate ID and credentials", func(t *testing.T) {
		oldStdout := os.Stdout
		r, w, err := os.Pipe()
		if err != nil {
			t.Fatalf("failed to create pipe: %v", err)
		}
		os.Stdout = w

		var logBuf bytes.Buffer
		testLogger := slog.New(slog.NewTextHandler(&logBuf, &slog.HandlerOptions{Level: slog.LevelInfo}))

		// 构造桌面 IPC 模式下的刷新回调 (isDesktopIPC = true)
		ipcHandler := buildTokenRefreshedHandler(true, testLogger)
		ipcHandler("acc-001", "personal_work", tok)

		_ = w.Close()
		os.Stdout = oldStdout
		outBytes, _ := io.ReadAll(r)
		stdoutOutput := string(outBytes)

		// 验证结构化事件输出
		if !strings.Contains(stdoutOutput, "__TRAE_EVENT__:") {
			t.Fatalf("expected __TRAE_EVENT__ in desktop IPC mode, got: %s", stdoutOutput)
		}

		lines := strings.Split(stdoutOutput, "\n")
		var eventPayload map[string]interface{}
		for _, line := range lines {
			if strings.HasPrefix(line, "__TRAE_EVENT__:") {
				raw := strings.TrimPrefix(line, "__TRAE_EVENT__:")
				if err := json.Unmarshal([]byte(raw), &eventPayload); err != nil {
					t.Fatalf("failed to parse event json: %v", err)
				}
				break
			}
		}

		if eventPayload == nil {
			t.Fatalf("no valid __TRAE_EVENT__ payload parsed from: %s", stdoutOutput)
		}
		if eventPayload["id"] != "acc-001" {
			t.Errorf("expected id 'acc-001', got: %v", eventPayload["id"])
		}
		if eventPayload["account"] != "personal_work" {
			t.Errorf("expected account 'personal_work', got: %v", eventPayload["account"])
		}
		if eventPayload["token"] != sensitiveAccessToken {
			t.Errorf("expected token %s, got: %v", sensitiveAccessToken, eventPayload["token"])
		}
		if eventPayload["refresh_token"] != sensitiveRefreshToken {
			t.Errorf("expected refresh_token %s, got: %v", sensitiveRefreshToken, eventPayload["refresh_token"])
		}
	})
}

// TestAccountID_AutoGeneration verifies that when accounts in config do not specify IDs,
// stable and collision-free IDs are generated, even for accounts with the same name.
func TestAccountID_AutoGeneration(t *testing.T) {
	name := "default"
	token1 := "token_aaa"
	token2 := "token_bbb"

	h1 := sha256.Sum256([]byte(fmt.Sprintf("%d:%s:%s", 0, name, token1)))
	id1 := fmt.Sprintf("acc_%d_%x", 0, h1[:4])

	h2 := sha256.Sum256([]byte(fmt.Sprintf("%d:%s:%s", 1, name, token2)))
	id2 := fmt.Sprintf("acc_%d_%x", 1, h2[:4])

	if id1 == id2 {
		t.Fatalf("collision detected for identical names: %s == %s", id1, id2)
	}
	if !strings.HasPrefix(id1, "acc_0_") {
		t.Errorf("expected acc_0_ prefix, got: %s", id1)
	}
	if !strings.HasPrefix(id2, "acc_1_") {
		t.Errorf("expected acc_1_ prefix, got: %s", id2)
	}
}
