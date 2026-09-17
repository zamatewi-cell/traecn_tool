import json

with open("scripts/captured/decrypted_agent_task_body.bin", "rb") as f:
    raw = f.read()

d = json.loads(raw.decode("utf-8"))
print("extra_config:", json.dumps(d.get("extra_config"), indent=2, ensure_ascii=False))
print("render_context:", json.dumps(d.get("render_context"), indent=2, ensure_ascii=False))
print("history_id_list:", d.get("history_id_list"))
print("user_input:", json.dumps(d.get("user_input"), indent=2, ensure_ascii=False))
