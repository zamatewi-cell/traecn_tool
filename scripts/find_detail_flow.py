from mitmproxy import io as mio
from mitmproxy import http

with open("scripts/captured/flows.bin", "rb") as f:
    reader = mio.FlowReader(f)
    idx = 0
    for flow in reader.stream():
        if not isinstance(flow, http.HTTPFlow):
            continue
        idx += 1
        resp = flow.response
        if resp and resp.content:
            if len(resp.content) == 462006:
                print(f"Matched detail_param_response in flow {idx}: {flow.request.pretty_url}")
            elif b"config_info_list" in resp.content:
                print(f"Matched config_info_list in flow {idx}: {flow.request.pretty_url}")

with open("scripts/captured/detail_param_response.bin", "rb") as f:
    det = f.read()

print(f"detail_param_response.bin len={len(det)}")
print(f"First 16 bytes hex: {det[:16].hex()}")
