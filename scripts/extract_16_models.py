import sys
sys.stdout.reconfigure(encoding='utf-8')
import json

with open("scripts/detail_param_v3396.json", "r", encoding="utf-8") as f:
    d = json.load(f)

configs = d.get("config_info_list", [])

# Known 5 preset models
presets = {"seed_m8", "Doubao_1_5_thinking_pro", "deepseek-R1", "deepseek-V3", "deepseek-V3-0324"}

model_matrix = []

for item in configs:
    cfg_name = item.get("config_name")
    disp_cfg = item.get("display_config") or {}
    disp_name = disp_cfg.get("display_name")
    ctx_tokens = item.get("context_window_tokens", {})
    multimodal = disp_cfg.get("multimodal", False)
    fee_level = disp_cfg.get("fee_model_level", 0)
    
    details = item.get("model_detail_list") or []
    models = [m.get("model_name") or m.get("name") for m in details]
    
    # Check if this is a primary user-facing model
    if disp_name and cfg_name not in presets:
        model_matrix.append({
            "config_name": cfg_name,
            "display_name": disp_name,
            "models": models,
            "context_tokens": ctx_tokens,
            "multimodal": multimodal,
            "fee_level": fee_level,
            "is_default": item.get("is_default", False),
            "is_invisible": item.get("is_invisible_to_user", False)
        })

print(f"Total non-preset configs with display_name: {len(model_matrix)}")
for i, m in enumerate(model_matrix):
    print(f"[{i+1:02d}] config: {m['config_name']:30s} | display: {m['display_name']:25s} | models: {m['models']} | ctx: {m['context_tokens']}")

