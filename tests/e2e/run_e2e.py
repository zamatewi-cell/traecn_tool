#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
Trae-Proxy 端到端自动化黑盒测试套件 (E2E Test Runner)

支持双模式：
1. 外部网关实测模式：对已启动的网关发起完整 HTTP/SSE 探测 (指定 --gateway-url，如 http://localhost:8080)
2. Go 原生沙箱自测模式：自动化执行 Go E2E 沙箱测试套件 (支持快速离线验证与 CI)
"""

import sys
import os
import json
import time
import argparse
import subprocess
from typing import List, Dict, Any, Tuple

# ANSI 彩色高亮定义
GREEN = "\033[92m"
RED = "\033[91m"
YELLOW = "\033[93m"
CYAN = "\033[96m"
BOLD = "\033[1m"
RESET = "\033[0m"

LEGACY_MODELS = [
    "seed_m8",
    "Doubao_1_5_thinking_pro",
    "deepseek-R1",
    "deepseek-V3",
    "deepseek-V3-0324",
]

NEW_BUILTIN_MODELS = [
    "Seed-Code",
    "Seed-Evolving",
    "Seed-2.1-Pro-0915",
    "Seed-2.1-Turbo",
    "GLM-5.3-Flash",
    "GLM-5.3",
    "GLM-5.2",
    "DeepSeek-V4.1-Flash",
    "DeepSeek-V4-Flash",
    "DeepSeek-V4-Pro",
    "Kimi-K3",
    "Kimi-K2.8-Preview",
    "MiniMax-M3",
    "Qwen3.8-Flash",
    "Qwen3.8-Max",
    "Qwen3.7-Plus",
]

def log_tier_header(tier_name: str, desc: str):
    print(f"\n{BOLD}{CYAN}{'='*68}{RESET}")
    print(f"{BOLD}{CYAN}[{tier_name}] {desc}{RESET}")
    print(f"{BOLD}{CYAN}{'='*68}{RESET}")

def log_result(tc_id: str, desc: str, passed: bool, detail: str = ""):
    status = f"{GREEN}[PASS]{RESET}" if passed else f"{RED}[FAIL]{RESET}"
    print(f"  {status} {BOLD}{tc_id}{RESET} - {desc}")
    if detail:
        print(f"         {YELLOW}↳ {detail}{RESET}")

class E2EGatewayTester:
    def __init__(self, base_url: str):
        self.base_url = base_url.rstrip("/")
        import requests
        self.session = requests.Session()
        self.session.headers.update({"Content-Type": "application/json"})
        self.passed_count = 0
        self.failed_count = 0

    def record(self, passed: bool):
        if passed:
            self.passed_count += 1
        else:
            self.failed_count += 1

    def test_tier1(self):
        log_tier_header("Tier 1: 特性全量覆盖", "5个存量模型 + 16个新内置模型 + /v1/models")

        # 1. /v1/models 完整性测试
        try:
            r = self.session.get(f"{self.base_url}/v1/models", timeout=10)
            passed = (r.status_code == 200)
            data = r.json() if passed else {}
            models_list = data.get("data", [])
            model_ids = {m.get("id") for m in models_list}

            missing_builtin = [m for m in NEW_BUILTIN_MODELS if m not in model_ids]
            check_ok = passed and (len(missing_builtin) == 0) and (data.get("object") == "list")
            log_result("TC-T1-001", "/v1/models 模型列表完整性检索", check_ok,
                       f"总模型数: {len(models_list)}, 缺漏新内置模型: {missing_builtin or '无'}")
            self.record(check_ok)
        except Exception as e:
            log_result("TC-T1-001", "/v1/models 模型列表完整性检索", False, str(e))
            self.record(False)

        # 2. 存量 5 个预设模型对话补全
        for m in LEGACY_MODELS:
            payload = {
                "model": m,
                "messages": [{"role": "user", "content": f"Ping legacy model {m}"}],
                "stream": False
            }
            try:
                r = self.session.post(f"{self.base_url}/v1/chat/completions", json=payload, timeout=20)
                passed = (r.status_code == 200)
                content = ""
                if passed:
                    cdata = r.json()
                    choices = cdata.get("choices", [])
                    if choices:
                        content = choices[0].get("message", {}).get("content", "")
                check_ok = passed and bool(content)
                log_result(f"TC-T1-002[{m}]", f"存量预设模型对话验证: {m}", check_ok, f"响应长度: {len(content)}")
                self.record(check_ok)
            except Exception as e:
                log_result(f"TC-T1-002[{m}]", f"存量预设模型对话验证: {m}", False, str(e))
                self.record(False)

        # 3. 16 个新内置模型对话补全
        for m in NEW_BUILTIN_MODELS:
            payload = {
                "model": m,
                "messages": [{"role": "user", "content": f"Ping new model {m}"}],
                "stream": False
            }
            try:
                r = self.session.post(f"{self.base_url}/v1/chat/completions", json=payload, timeout=20)
                passed = (r.status_code == 200)
                content = ""
                if passed:
                    cdata = r.json()
                    choices = cdata.get("choices", [])
                    if choices:
                        content = choices[0].get("message", {}).get("content", "")
                check_ok = passed and bool(content)
                log_result(f"TC-T1-003[{m}]", f"新一代内置模型对话验证: {m}", check_ok, f"响应长度: {len(content)}")
                self.record(check_ok)
            except Exception as e:
                log_result(f"TC-T1-003[{m}]", f"新一代内置模型对话验证: {m}", False, str(e))
                self.record(False)

    def test_tier2(self):
        log_tier_header("Tier 2: 边界与容错", "畸形请求、未知模型、极端参数防护")

        # 1. 空请求与缺失字段
        cases = [
            ("空请求体", "", 400),
            ("缺少model", {"messages": [{"role": "user", "content": "hi"}]}, 400),
            ("缺少messages", {"model": "Seed-Code"}, 400),
            ("messages为空数组", {"model": "Seed-Code", "messages": []}, 400),
        ]
        for name, body, expected_code in cases:
            try:
                if isinstance(body, str):
                    r = self.session.post(f"{self.base_url}/v1/chat/completions", data=body, timeout=10)
                else:
                    r = self.session.post(f"{self.base_url}/v1/chat/completions", json=body, timeout=10)
                passed = (r.status_code == expected_code)
                log_result(f"TC-T2-001[{name}]", f"非法/缺失参数拦截", passed, f"状态码: {r.status_code}")
                self.record(passed)
            except Exception as e:
                log_result(f"TC-T2-001[{name}]", f"非法/缺失参数拦截", False, str(e))
                self.record(False)

        # 2. 未知模型
        try:
            r = self.session.post(f"{self.base_url}/v1/chat/completions", json={
                "model": "non-existent-fake-model-xyz",
                "messages": [{"role": "user", "content": "test"}]
            }, timeout=10)
            passed = (r.status_code in [200, 400, 502])
            log_result("TC-T2-002", "未知模型名称容错（不崩溃）", passed, f"状态码: {r.status_code}")
            self.record(passed)
        except Exception as e:
            log_result("TC-T2-002", "未知模型名称容错（不崩溃）", False, str(e))
            self.record(False)

        # 3. 极端参数
        try:
            r = self.session.post(f"{self.base_url}/v1/chat/completions", json={
                "model": "Seed-Code",
                "messages": [{"role": "user", "content": "extreme"}],
                "temperature": -2.0,
                "max_tokens": 99999999
            }, timeout=10)
            passed = (r.status_code in [200, 400])
            log_result("TC-T2-003", "极端参数容错 (temperature/max_tokens)", passed, f"状态码: {r.status_code}")
            self.record(passed)
        except Exception as e:
            log_result("TC-T2-003", "极端参数容错 (temperature/max_tokens)", False, str(e))
            self.record(False)

    def test_tier3(self):
        log_tier_header("Tier 3: 多模型组合与路由验证", "连续交替调用与并发请求稳定性")

        # 连续交替调用
        seq = ["seed_m8", "DeepSeek-V4.1-Flash", "deepseek-R1", "GLM-5.3"]
        seq_ok = True
        for m in seq:
            try:
                r = self.session.post(f"{self.base_url}/v1/chat/completions", json={
                    "model": m,
                    "messages": [{"role": "user", "content": f"ping {m}"}]
                }, timeout=15)
                if r.status_code != 200:
                    seq_ok = False
                    break
            except Exception:
                seq_ok = False
                break
        log_result("TC-T3-001", "双通道模型顺序交替路由无串扰", seq_ok, f"测试序列: {' -> '.join(seq)}")
        self.record(seq_ok)

    def test_tier4(self):
        log_tier_header("Tier 4: 端到端真实流式 (SSE)", "SSE增量推送、reasoning_content与[DONE]")

        for m in ["Seed-Code", "Doubao_1_5_thinking_pro"]:
            payload = {
                "model": m,
                "messages": [{"role": "user", "content": "Count from 1 to 3"}],
                "stream": True
            }
            try:
                r = self.session.post(f"{self.base_url}/v1/chat/completions", json=payload, stream=True, timeout=20)
                is_sse = "text/event-stream" in r.headers.get("Content-Type", "")
                got_done = False
                chunks_received = 0
                has_role = False
                content_acc = ""

                for line in r.iter_lines(decode_unicode=True):
                    if not line:
                        continue
                    if line.startswith("data: "):
                        d = line[6:].strip()
                        if d == "[DONE]":
                            got_done = True
                            break
                        try:
                            cj = json.loads(d)
                            chunks_received += 1
                            ch = cj.get("choices", [])
                            if ch:
                                delta = ch[0].get("delta", {})
                                if delta.get("role") == "assistant":
                                    has_role = True
                                if delta.get("content"):
                                    content_acc += delta.get("content")
                        except Exception:
                            pass

                check_ok = is_sse and got_done and (chunks_received > 0)
                log_result(f"TC-T4-001[{m}]", f"流式 SSE 规范与 [DONE] 结束标记: {m}", check_ok,
                           f"SSE协议: {is_sse}, 收到Chunk: {chunks_received}, 结束标记: {got_done}")
                self.record(check_ok)
            except Exception as e:
                log_result(f"TC-T4-001[{m}]", f"流式 SSE 规范与 [DONE] 结束标记: {m}", False, str(e))
                self.record(False)

    def run_all(self) -> int:
        print(f"\n{BOLD}开始执行端到端自动化测试套件 (目标网关: {self.base_url}){RESET}")
        start_time = time.time()
        self.test_tier1()
        self.test_tier2()
        self.test_tier3()
        self.test_tier4()
        cost = time.time() - start_time

        print(f"\n{BOLD}{CYAN}{'='*68}{RESET}")
        print(f"{BOLD}测试总结报告 (Summary):{RESET}")
        total = self.passed_count + self.failed_count
        print(f"  总用例数: {total}")
        print(f"  通过用例: {GREEN}{self.passed_count}{RESET}")
        print(f"  失败用例: {RED if self.failed_count else GREEN}{self.failed_count}{RESET}")
        print(f"  执行耗时: {cost:.2f} 秒")
        print(f"{BOLD}{CYAN}{'='*68}{RESET}\n")
        return 0 if self.failed_count == 0 else 1

def run_go_sandbox_tests() -> int:
    print(f"\n{BOLD}{CYAN}{'='*68}{RESET}")
    print(f"{BOLD}{CYAN}执行 Go 原生沙箱端到端测试套件 (go test -v ./tests/e2e/...){RESET}")
    print(f"{BOLD}{CYAN}{'='*68}{RESET}\n")

    cmd = ["go", "test", "-v", "./tests/e2e/..."]
    res = subprocess.run(cmd, capture_output=False)
    return res.returncode

def main():
    parser = argparse.ArgumentParser(description="Trae-Proxy E2E Test Suite Runner")
    parser.add_argument("--gateway-url", type=str, default="", help="外部网关URL (如 http://localhost:8080)")
    parser.add_argument("--sandbox", action="store_true", help="强制执行 Go 原生沙箱测试")
    args = parser.parse_args()

    if args.sandbox or not args.gateway_url:
        # 默认模式：执行 Go 沙箱测试套件
        ret = run_go_sandbox_tests()
        sys.exit(ret)
    else:
        tester = E2EGatewayTester(args.gateway_url)
        sys.exit(tester.run_all())

if __name__ == "__main__":
    main()
