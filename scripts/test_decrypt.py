import base64
from cryptography.hazmat.primitives.ciphers.aead import AESGCM

# Read req_body.bin
with open("scripts/captured/agent_task_req_body.bin", "rb") as f:
    raw_b64 = f.read()

enc_data = base64.b64decode(raw_b64)
print("Decoded bytes:", len(enc_data))

key_hex = "6195f24ca4d430f8a4833de7db8dac37d148a084e7464a351ffa68585c16b955"
key = bytearray(bytes.fromhex(key_hex))
pin_hex = "9ceba7caf2dde2cb"
pin = bytes.fromhex(pin_hex)

for i in range(8):
    key[i] ^= pin[i]

requested_at = b"1773329068"

# In masticate:
# out = iv (12B) || ciphertext || tag (16B)
# In AESGCM: ciphertext_with_tag = ciphertext || tag
iv = enc_data[:12]
ct = enc_data[12:]

print("IV:", iv.hex())
print("IV first 4 bytes:", iv[:4].hex())

try:
    aesgcm = AESGCM(bytes(key))
    decrypted = aesgcm.decrypt(iv, ct, requested_at)
    print("SUCCESS! Decrypted length:", len(decrypted))
    print("Decrypted preview (first 500 chars):")
    print(decrypted[:500])
    with open("scripts/captured/decrypted_agent_task_body.bin", "wb") as f:
        f.write(decrypted)
except Exception as e:
    print("Decrypt failed with request_at:", e)
    # Try with empty AAD
    try:
        decrypted = aesgcm.decrypt(iv, ct, None)
        print("SUCCESS with None AAD! Decrypted length:", len(decrypted))
    except Exception as e2:
        print("Decrypt failed with None AAD:", e2)

