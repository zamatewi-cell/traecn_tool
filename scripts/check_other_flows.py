"""Check if other API requests are also encrypted."""
from mitmproxy import io as mio
from mitmproxy import http

with open("scripts/captured/flows.bin", "rb") as f:
    reader = mio.FlowReader(f)
    for flow in reader.stream():
        if not isinstance(flow, http.HTTPFlow):
            continue
        path = flow.request.path.split("?")[0]
        body = flow.request.content or b""
        
        if "llm_raw_chat" in path:
            print(f"llm_raw_chat: {len(body)} bytes")
            print("First 300:", body[:300])
            if flow.response and flow.response.content:
                print("Response first 300:", flow.response.content[:300])
            print()
        elif "uploadFilesLimitConfig" in path:
            print(f"uploadFilesLimitConfig: {len(body)} bytes")
            print("Body:", body[:300])
            print()
        elif "get_client_config" in path:
            print(f"get_client_config: {len(body)} bytes")
            print("Body:", body[:300])
            print()
        elif "sync_history" in path:
            print(f"sync_history: {len(body)} bytes")
            print("Body:", body[:300])
            print()
