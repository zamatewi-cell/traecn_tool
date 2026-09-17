import json

with open("scripts/captured/decrypted_agent_task_body.bin", "rb") as f:
    raw = f.read()

d = json.loads(raw.decode("utf-8"))
print("Decrypted body top-level keys:")
for k, v in d.items():
    v_type = type(v).__name__
    v_len = len(v) if isinstance(v, (list, dict, str)) else 0
    print(f"  - {k:25s} (type={v_type:6s}, len={v_len:6d})")
    if k in ["agent_id", "tunnel_id", "is_custom_model", "provider", "conversation_id", "session_id", "plugin_channel", "user_id", "device_id", "agent_type", "config_name", "model_name", "ide_version", "config_source"]:
        print(f"      value: {v}")
    elif k == "user_input":
        print(f"      user_input keys: {list(v.keys())}")
        print(f"      messages: {v.get('messages')}")

