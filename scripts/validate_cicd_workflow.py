import sys
import os
import yaml

if hasattr(sys.stdout, 'reconfigure'):
    sys.stdout.reconfigure(encoding='utf-8')

def validate_workflow():
    workflow_path = os.path.join(".github", "workflows", "build.yml")
    print(f"================================================================================")
    print(f"🔍 检查 CI/CD 工作流配置文件: {workflow_path}")
    print(f"================================================================================")

    if not os.path.exists(workflow_path):
        print(f"❌ 错误: 找不到文件 {workflow_path}")
        return False

    with open(workflow_path, "r", encoding="utf-8") as f:
        content = f.read()

    # 1. YAML 语法与解析性
    try:
        data = yaml.safe_load(content)
        print("✔ 1. YAML 语法合法性: 解析成功，语法完全合规")
    except yaml.YAMLError as e:
        print(f"❌ 1. YAML 语法解析失败: {e}")
        return False

    all_passed = True

    # 2. 触发条件检查
    triggers = data.get("on") or data.get(True) or {}
    print("\n✔ 2. 触发条件校验:")
    print(f"   on: {triggers}")
    push_cfg = triggers.get("push", {})
    pr_cfg = triggers.get("pull_request", {})

    if "main" in push_cfg.get("branches", []):
        print("   ✔ push 包含 main 分支")
    else:
        print("   ❌ push 缺少 main 分支")
        all_passed = False

    tags = push_cfg.get("tags", [])
    if any(t in ["v*", "v*.*.*"] for t in tags):
        print(f"   ✔ push 包含标签触发: {tags}")
    else:
        print(f"   ❌ push 缺少版本标签触发: {tags}")
        all_passed = False

    if "main" in pr_cfg.get("branches", []):
        print("   ✔ pull_request 包含 main 分支")
    else:
        print("   ❌ pull_request 缺少 main 分支")
        all_passed = False

    # 3. 校验权限
    perms = data.get("permissions", {})
    if perms.get("contents") == "write":
        print(f"   ✔ permissions.contents 为 write (满足 GitHub Release 自动发布权限)")
    else:
        print(f"   ❌ permissions 缺少 contents: write，实际: {perms}")
        all_passed = False

    # 4. 校验各个 Job
    jobs = data.get("jobs", {})
    expected_jobs = ["test", "build-proxy", "build-desktop", "release"]
    print(f"\n✔ 3. Job 架构与依赖链路校验 (定义了 {len(jobs)} 个 Job: {list(jobs.keys())}):")
    for ej in expected_jobs:
        if ej in jobs:
            print(f"   ✔ 包含关键 Job: {ej}")
        else:
            print(f"   ❌ 缺少关键 Job: {ej}")
            all_passed = False

    # 4.1 test job
    test_job = jobs.get("test", {})
    test_steps = test_job.get("steps", [])
    print("   [Job: test]")
    print(f"     runs-on: {test_job.get('runs-on')}")
    test_runs = [s.get("run", "") for s in test_steps if "run" in s]
    has_go_test = any("go test" in r for r in test_runs)
    has_npm_build = any("run build" in r for r in test_runs)
    print(f"     包含 Go 单元测试门禁: {has_go_test}")
    print(f"     包含前端编译门禁: {has_npm_build}")
    if not (has_go_test and has_npm_build):
        all_passed = False

    # 4.2 build-proxy job
    bp_job = jobs.get("build-proxy", {})
    print("   [Job: build-proxy]")
    bp_needs = bp_job.get("needs", [])
    print(f"     依赖项 needs: {bp_needs}")
    if "test" not in bp_needs:
        print("     ❌ build-proxy 缺少对 test 的依赖！")
        all_passed = False
    else:
        print("     ✔ build-proxy 正确依赖 test 门禁")

    strategy = bp_job.get("strategy", {})
    matrix_include = strategy.get("matrix", {}).get("include", [])
    arch_set = set()
    for item in matrix_include:
        os_name = item.get("os")
        arch_name = item.get("arch")
        arch_set.add(f"{os_name}/{arch_name}")
    print(f"     矩阵架构包含: {sorted(list(arch_set))}")
    required_archs = {
        "windows/amd64", "windows/arm64",
        "linux/amd64", "linux/arm64",
        "darwin/amd64", "darwin/arm64"
    }
    missing_archs = required_archs - arch_set
    if missing_archs:
        print(f"     ❌ 缺少目标架构: {missing_archs}")
        all_passed = False
    else:
        print(f"     ✔ 6 大目标跨平台架构矩阵完全覆盖 (Win/Linux/macOS x amd64/arm64)")

    # 4.3 build-desktop job
    bd_job = jobs.get("build-desktop", {})
    print("   [Job: build-desktop]")
    bd_needs = bd_job.get("needs", [])
    print(f"     依赖项 needs: {bd_needs}")
    if "test" not in bd_needs:
        print("     ❌ build-desktop 缺少对 test 的依赖！")
        all_passed = False
    else:
        print("     ✔ build-desktop 正确依赖 test 门禁")
    bd_steps = bd_job.get("steps", [])
    bd_runs = [s.get("run", "") for s in bd_steps if "run" in s]
    has_prebuild = any("trae-proxy" in r for r in bd_runs)
    has_dist_win = any("dist:win" in r for r in bd_runs)
    print(f"     包含预编译内嵌 trae-proxy: {has_prebuild}")
    print(f"     包含 Electron dist:win 打包: {has_dist_win}")
    if not (has_prebuild and has_dist_win):
        all_passed = False

    # 4.4 release job
    rel_job = jobs.get("release", {})
    print("   [Job: release]")
    rel_needs = rel_job.get("needs", [])
    print(f"     依赖项 needs: {rel_needs}")
    if "build-proxy" not in rel_needs or "build-desktop" not in rel_needs:
        print("     ❌ release 依赖不完整，必须等待 build-proxy 和 build-desktop 完成！")
        all_passed = False
    else:
        print("     ✔ release 正确依赖 [build-proxy, build-desktop]")

    rel_cond = rel_job.get("if", "")
    print(f"     触发条件 if: {rel_cond}")
    if "tags/v" in rel_cond:
        print("     ✔ release 条件限制严格匹配版本标签 (v*)")
    else:
        print(f"     ❌ release 触发条件存在缺陷: {rel_cond}")
        all_passed = False

    rel_steps = rel_job.get("steps", [])
    rel_runs = [s.get("run", "") for s in rel_steps if "run" in s]
    has_checksums = any("sha256sum" in r for r in rel_runs)
    print(f"     包含 SHA256 校验和自动生成 (sha256sum): {has_checksums}")
    if not has_checksums:
        all_passed = False

    # 5. 校验 Action 引用安全性与现代化版本
    print("\n✔ 4. GitHub Actions 引用版本检查:")
    all_uses = []
    for j_name, j_data in jobs.items():
        for step in j_data.get("steps", []):
            if "uses" in step:
                all_uses.append((j_name, step.get("name", "unnamed"), step["uses"]))

    for j_name, s_name, u in all_uses:
        print(f"   - [{j_name}] {s_name}: {u}")
        # 验证不是废弃的旧版本 (v1, v2 for checkout/setup)
        if "actions/checkout@" in u and not u.endswith("@v4"):
            print(f"     ⚠️ 警告: actions/checkout 建议使用 @v4，当前: {u}")
        if "actions/setup-go@" in u and not u.endswith("@v5"):
            print(f"     ⚠️ 警告: actions/setup-go 建议使用 @v5，当前: {u}")
        if "actions/setup-node@" in u and not u.endswith("@v4"):
            print(f"     ⚠️ 警告: actions/setup-node 建议使用 @v4，当前: {u}")
        if "actions/upload-artifact@" in u and not u.endswith("@v4"):
            print(f"     ⚠️ 警告: actions/upload-artifact 建议使用 @v4，当前: {u}")
        if "actions/download-artifact@" in u and not u.endswith("@v4"):
            print(f"     ⚠️ 警告: actions/download-artifact 建议使用 @v4，当前: {u}")
        if "softprops/action-gh-release@" in u and not u.endswith("@v2"):
            print(f"     ⚠️ 警告: softprops/action-gh-release 建议使用 @v2，当前: {u}")

    print("\n================================================================================")
    if all_passed:
        print("🏆 CI/CD 自动化流水线配置校验 100% 合规通过！")
    else:
        print("⚠️ CI/CD 自动化流水线配置发现不合规项！")
    print("================================================================================")
    return all_passed

if __name__ == "__main__":
    success = validate_workflow()
    sys.exit(0 if success else 1)
