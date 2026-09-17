from mitmproxy import io as mio
from mitmproxy import http
import json

flows_path = "scripts/captured/flows.bin"

with open(flows_path, "rb") as f:
    reader = mio.FlowReader(f)
    idx = 0
    for flow in reader.stream():
        if not isinstance(flow, http.HTTPFlow):
            continue
        idx += 1
        req = flow.request
        resp = flow.response
        url = req.pretty_url
        method = req.method
        req_len = len(req.content or b"")
        resp_len = len(resp.content or b"") if resp else 0
        resp_code = resp.status_code if resp else 0
        transport = req.headers.get("x-bridge-transport", "")
        pin = req.headers.get("x-request-pin", "")
        req_at = req.headers.get("x-requested-at", "")
        ctype = req.headers.get("content-type", "")
        resp_ctype = resp.headers.get("content-type", "") if resp else ""
        resp_encoding = resp.headers.get("content-encoding", "") if resp else ""

        print(f"[{idx:2d}] {method:4s} {resp_code:3d} {url[:70]}")
        print(f"     req_len={req_len:7d} ctype={ctype} transport={transport} pin={pin} req_at={req_at}")
        print(f"     resp_len={resp_len:7d} resp_ctype={resp_ctype} encoding={resp_encoding}")
        if req_len > 0 and req_len < 200:
            print(f"     req_preview: {req.content[:100]}")
        elif req_len >= 200:
            print(f"     req_preview (hex 16): {req.content[:16].hex()}")

