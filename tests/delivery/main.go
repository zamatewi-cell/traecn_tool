//go:build ignore

package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type TargetPlatform struct {
	OS   string
	Arch string
	Ext  string
}

func calculateSHA256(filePath string) (string, int64, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return "", 0, err
	}
	defer f.Close()

	stat, err := f.Stat()
	if err != nil {
		return "", 0, err
	}

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", 0, err
	}

	return hex.EncodeToString(h.Sum(nil)), stat.Size(), nil
}

func main() {
	fmt.Println("================================================================================")
	fmt.Println("🚀 开始执行：构建分发与跨平台工程实测挑战 (challenger_1)")
	fmt.Println("================================================================================")

	allPassed := true

	// =========================================================================
	// 挑战目标 1: 桌面打包真实产物深度实测与校验
	// =========================================================================
	fmt.Println("\n【挑战 1】检验桌面打包真实产物与内嵌独立代理")

	setupPath := filepath.Join("traecn-tools", "release", "TraeCN.Tools_1.0.0_x64-setup.exe")
	portablePath := filepath.Join("traecn-tools", "release", "TraeCN.Tools_1.0.0_x64-portable.exe")
	embeddedProxyPath := filepath.Join("traecn-tools", "release", "win-unpacked", "resources", "bin", "trae-proxy.exe")

	// 1.1 Setup.exe 校验
	setupHash, setupSize, err := calculateSHA256(setupPath)
	if err != nil {
		fmt.Printf("❌ [Setup.exe 检验失败] 无法读取文件: %v\n", err)
		allPassed = false
	} else {
		setupMB := float64(setupSize) / (1024 * 1024)
		fmt.Printf("✔ [Setup.exe 存在] 路径: %s\n", setupPath)
		fmt.Printf("   体积: %d 字节 (%.2f MB)\n", setupSize, setupMB)
		fmt.Printf("   SHA256: %s\n", setupHash)
		if setupSize > 80*1024*1024 {
			fmt.Printf("   ✔ 体积达标: %.2f MB > 80 MB 门槛要求\n", setupMB)
		} else {
			fmt.Printf("   ❌ 体积不合规: %.2f MB <= 80 MB 门槛要求\n", setupMB)
			allPassed = false
		}
	}

	// 1.2 Portable.exe 校验
	portHash, portSize, err := calculateSHA256(portablePath)
	if err != nil {
		fmt.Printf("❌ [Portable.exe 检验失败] 无法读取文件: %v\n", err)
		allPassed = false
	} else {
		portMB := float64(portSize) / (1024 * 1024)
		fmt.Printf("✔ [Portable.exe 存在] 路径: %s\n", portablePath)
		fmt.Printf("   体积: %d 字节 (%.2f MB)\n", portSize, portMB)
		fmt.Printf("   SHA256: %s\n", portHash)
		if portSize > 80*1024*1024 {
			fmt.Printf("   ✔ 体积达标: %.2f MB > 80 MB 门槛要求\n", portMB)
		} else {
			fmt.Printf("   ❌ 体积不合规: %.2f MB <= 80 MB 门槛要求\n", portMB)
			allPassed = false
		}
	}

	// 1.3 独立检查 win-unpacked resources/bin/trae-proxy.exe
	embHash, embSize, err := calculateSHA256(embeddedProxyPath)
	if err != nil {
		fmt.Printf("❌ [内嵌代理检验失败] 无法找到 win-unpacked 内嵌二进制: %v\n", err)
		allPassed = false
	} else {
		embMB := float64(embSize) / (1024 * 1024)
		fmt.Printf("✔ [内嵌代理存在] 路径: %s\n", embeddedProxyPath)
		fmt.Printf("   体积: %d 字节 (%.2f MB)\n", embSize, embMB)
		fmt.Printf("   SHA256: %s\n", embHash)

		// 运行内嵌代理 -version
		cmdEmb := exec.Command(embeddedProxyPath, "-version")
		outEmb, err := cmdEmb.CombinedOutput()
		if err != nil {
			fmt.Printf("❌ [内嵌代理独立执行失败] 无法执行 %s: %v\n", embeddedProxyPath, err)
			allPassed = false
		} else {
			verOut := strings.TrimSpace(string(outEmb))
			fmt.Printf("   独立执行输出: \"%s\"\n", verOut)
			if verOut == "trae-proxy v1.0.0" {
				fmt.Printf("   ✔ 内嵌代理版本号严格匹配: \"trae-proxy v1.0.0\"\n")
			} else {
				fmt.Printf("   ❌ 内嵌代理版本号不匹配，期望 \"trae-proxy v1.0.0\"，实际: \"%s\"\n", verOut)
				allPassed = false
			}
		}
	}

	// =========================================================================
	// 挑战目标 2: 跨平台交叉编译可行性实测 (CGO_ENABLED=0 下 6 大目标架构)
	// =========================================================================
	fmt.Println("\n【挑战 2】跨平台交叉编译实测 (CGO_ENABLED=0，6 大目标架构)")

	platforms := []TargetPlatform{
		{OS: "windows", Arch: "amd64", Ext: ".exe"},
		{OS: "windows", Arch: "arm64", Ext: ".exe"},
		{OS: "darwin", Arch: "amd64", Ext: ""},
		{OS: "darwin", Arch: "arm64", Ext: ""},
		{OS: "linux", Arch: "amd64", Ext: ""},
		{OS: "linux", Arch: "arm64", Ext: ""},
	}

	tempDir, err := os.MkdirTemp("", "trae_cross_build_*")
	if err != nil {
		fmt.Printf("❌ 创建临时构建目录失败: %v\n", err)
		allPassed = false
	} else {
		defer os.RemoveAll(tempDir)
		fmt.Printf("临时编译输出目录: %s\n", tempDir)

		for i, p := range platforms {
			binName := fmt.Sprintf("trae-proxy_%s_%s%s", p.OS, p.Arch, p.Ext)
			outPath := filepath.Join(tempDir, binName)

			start := time.Now()
			cmd := exec.Command("go", "build", "-trimpath", "-ldflags=-s -w", "-o", outPath, "./cmd/trae-proxy")
			cmd.Env = append(os.Environ(),
				"CGO_ENABLED=0",
				"GOOS="+p.OS,
				"GOARCH="+p.Arch,
			)

			out, err := cmd.CombinedOutput()
			duration := time.Since(start)

			if err != nil {
				fmt.Printf("❌ [%d/6] 交叉编译失败: %s/%s\n", i+1, p.OS, p.Arch)
				fmt.Printf("   错误输出:\n%s\n", string(out))
				allPassed = false
				continue
			}

			fi, err := os.Stat(outPath)
			if err != nil {
				fmt.Printf("❌ [%d/6] 产物不存在: %s\n", i+1, outPath)
				allPassed = false
				continue
			}

			sizeMB := float64(fi.Size()) / (1024 * 1024)
			fmt.Printf("✔ [%d/6] 编译成功: %-7s / %-5s -> %-30s | 体积: %5.2f MB | 耗时: %v\n",
				i+1, p.OS, p.Arch, binName, sizeMB, duration.Round(time.Millisecond))

			// 如果是 windows/amd64，本地执行验证版本
			if p.OS == "windows" && p.Arch == "amd64" {
				testCmd := exec.Command(outPath, "-version")
				verBytes, testErr := testCmd.CombinedOutput()
				if testErr != nil {
					fmt.Printf("   ❌ 交叉编译生成的 windows/amd64 二进制执行失败: %v\n", testErr)
					allPassed = false
				} else {
					verStr := strings.TrimSpace(string(verBytes))
					if verStr == "trae-proxy v1.0.0" {
						fmt.Printf("   ✔ 产物执行测试通过: \"%s\"\n", verStr)
					} else {
						fmt.Printf("   ❌ 产物版本输出异常: \"%s\"\n", verStr)
						allPassed = false
					}
				}
			}
		}
	}

	// =========================================================================
	// 汇总裁决
	// =========================================================================
	fmt.Println("\n================================================================================")
	if allPassed {
		fmt.Println("🏆 实测裁决: 全部测试通过 (ALL CHECKS PASSED) -> APPROVE")
	} else {
		fmt.Println("⚠️ 实测裁决: 存在未达标项 (FAILURES DETECTED) -> REQUEST_CHANGES")
	}
	fmt.Println("================================================================================")

	if !allPassed {
		os.Exit(1)
	}
}
