import brotli
import json

with open("scripts/captured/detail_param_response.bin", "rb") as f:
    data = f.read()

decomp = brotli.decompress(data)
print("Decompressed length:", len(decomp))
try:
    j = json.loads(decomp.decode("utf-8"))
    print("Parsed JSON keys:", list(j.keys()))
    print("config_info_list count:", len(j.get("config_info_list", [])))
except Exception as e:
    print("JSON parse error:", e)
