import json
import time
import secrets
from cryptography.hazmat.primitives.ciphers.aead import AESGCM
import base64
import requests

with open("scripts/current_token.txt", "r", encoding="utf-8") as f:
    token = f.read().strip()

with open("scripts/captured/decrypted_agent_task_body.bin", "rb") as f:
    template = json.loads(f.read().decode("utf-8"))

test_models = [
    ("Doubao-Seed-Code", "Doubao-Seed-Code__dev"),
    ("DeepSeek-V4-Flash", "DeepSeek-V4-Flash__dev"),
    ("glm-5.2", "glm-5.2__dev"),
]

key_hex = "6195f24ca4d430f8a4833de7db8dac37d148a084e7464a351ffa68585c16b955"

for cfg, mname in test_models:
    body = json.loads(json.dumps(template))
    body["config_name"] = cfg
    body["model_name"] = mname
    body["history_id_list"] = [] # empty history to avoid missing_history
    body["conversation_id"] = secrets.token_hex(12)
    body["session_id"] = secrets.token_hex(12)
    body["user_input"] = {
        "id": secrets.token_hex(12),
        "messages": [
            {"type": "text", "text_content": "请回复'pong'"}
        ]
    }
    
    body_bytes = json.dumps(body, ensure_ascii=False).encode("utf-8")
    key = bytearray(bytes.fromhex(key_hex))
    pin = secrets.token_bytes(8)
    for i in range(8):
        key[i] ^= pin[i]
    iv = secrets.token_bytes(12)
    request_at = int(time.time())
    aad = str(request_at).encode("utf-8")
    aesgcm = AESGCM(bytes(key))
    ciphertext_with_tag = aesgcm.encrypt(iv, body_bytes, aad)
    enc_b64 = base64.b64encode(iv + ciphertext_with_tag).decode("ascii")

    headers = {
        "Content-Type": "application/json",
        "X-IDE-Token": token,
        "X-Request-Pin": pin.hex(),
        "X-Requested-At": str(request_at),
        "x-bridge-transport": "aha",
        "User-Agent": "TraeClient/TTNet",
        "app-version": "3.3.37",
        "x-ide-version": "3.3.37",
        "x-app-id": "6eefa01c-1036-4c7e-9ca5-d891f63bfcd8",
        "package-type": "stable_cn",
        "x-request-id": f"req_{secrets.token_hex(16)}",
        "x-trae-request-id": f"{secrets.token_hex(16)}",
    }

    url = "https://trae-api-cn.mchost.guru/api/agent/v3/create_agent_task"
    print(f"\n--- Testing config={cfg}, model={mname} ---")
    try:
        resp = requests.post(url, data=enc_b64, headers=headers, stream=True, timeout=15)
        print(f"Status: {resp.status_code}")
        ev_count = 0
        for line in resp.iter_lines(decode_unicode=True):
            if line:
                if line.startswith("event:") or line.startswith("data:"):
                    print(f"  {line[:120]}")
                    ev_count += 1
                if ev_count >= 10:
                    break
    except Exception as e:
        print(f"Request error: {e}")
    time.sleep(1)

