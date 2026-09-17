import json
import time
import os
import secrets
from cryptography.hazmat.primitives.ciphers.aead import AESGCM
import base64
import requests

# 1. Load current token
with open("scripts/current_token.txt", "r", encoding="utf-8") as f:
    token = f.read().strip()

# 2. Load decrypted agent task body
with open("scripts/captured/decrypted_agent_task_body.bin", "rb") as f:
    body_json = json.loads(f.read().decode("utf-8"))

# Modify body to test
body_json["user_input"] = {
    "id": secrets.token_hex(12),
    "messages": [
        {"type": "text", "text_content": "你好，请用一句话介绍你自己"}
    ]
}
body_json["conversation_id"] = secrets.token_hex(12)
body_json["session_id"] = secrets.token_hex(12)

# Test with a new builtin model, e.g. minimax-m2.5 or Doubao-Seed-Code or deepseek-v4-flash
# Let's keep minimax-m2.5 first as captured
body_bytes = json.dumps(body_json, ensure_ascii=False).encode("utf-8")

# 3. Masticate encryption
key_hex = "6195f24ca4d430f8a4833de7db8dac37d148a084e7464a351ffa68585c16b955"
key = bytearray(bytes.fromhex(key_hex))
pin = secrets.token_bytes(8)
for i in range(8):
    key[i] ^= pin[i]

iv = secrets.token_bytes(12)
request_at = int(time.time())
aad = str(request_at).encode("utf-8")

aesgcm = AESGCM(bytes(key))
ciphertext_with_tag = aesgcm.encrypt(iv, body_bytes, aad)

# Combine: iv || ciphertext_with_tag
enc_payload = iv + ciphertext_with_tag
enc_b64 = base64.b64encode(enc_payload).decode("ascii")

# 4. Prepare headers
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

print(f"Sending live request to /api/agent/v3/create_agent_task...")
print(f"Model: {body_json.get('model_name')}, Config: {body_json.get('config_name')}")
print(f"Pin: {pin.hex()}, RequestAt: {request_at}, IV: {iv.hex()}")

url = "https://trae-api-cn.mchost.guru/api/agent/v3/create_agent_task"
resp = requests.post(url, data=enc_b64, headers=headers, stream=True, timeout=30)
print(f"Status: {resp.status_code}")
print(f"Response headers: {dict(resp.headers)}")

lines = []
for line in resp.iter_lines(decode_unicode=True):
    if line:
        print(f"  SSE: {line[:200]}")
        lines.append(line)
        if len(lines) > 20:
            break

