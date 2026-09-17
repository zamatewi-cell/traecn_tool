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

body = json.loads(json.dumps(template))
body["config_name"] = "Doubao-Seed-Code"
body["model_name"] = "Doubao-Seed-Code__dev"
body["history_id_list"] = []
body["conversation_id"] = secrets.token_hex(12)
body["session_id"] = secrets.token_hex(12)
body["user_input"] = {
    "id": secrets.token_hex(12),
    "messages": [
        {"type": "text", "text_content": "请用Python写一个Hello World函数"}
    ]
}

body_bytes = json.dumps(body, ensure_ascii=False).encode("utf-8")
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
print("Sending prompt to Doubao-Seed-Code...")
resp = requests.post(url, data=enc_b64, headers=headers, stream=True, timeout=60)
print("HTTP Status:", resp.status_code)

for line in resp.iter_lines(decode_unicode=True):
    if not line:
        continue
    if line.startswith("event:"):
        ev = line[6:].strip()
        print(f"\n[EVENT: {ev}]")
    elif line.startswith("data:"):
        try:
            d = json.loads(line[5:].strip())
            if "thought" in d and d["thought"]:
                print(d["thought"], end="", flush=True)
            elif "reasoning_content" in d and d["reasoning_content"]:
                print(f"[THINK] {d['reasoning_content']}", end="", flush=True)
            elif "task_id" in d:
                print(f"task_id={d['task_id']}")
            elif "token_usage" in str(line):
                print(f"tokens: prompt={d.get('prompt_tokens')}, comp={d.get('completion_tokens')}")
        except:
            print(f"data: {line[5:80]}")

print("\n\nStream finished.")
