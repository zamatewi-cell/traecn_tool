"""
mitmproxy addon to capture Trae CN API request bodies.
Usage: mitmdump --mode "local:Trae CN" -s mitm_capture.py -w captured/flows.bin
"""
import json
import os
import re
import time
import traceback
from mitmproxy import http

CAPTURE_DIR = os.path.join(os.path.dirname(os.path.abspath(__file__)), "captured")
os.makedirs(CAPTURE_DIR, exist_ok=True)

def safe_filename(s):
    """Remove characters invalid for Windows filenames."""
    return re.sub(r'[<>:"/\\|?*]', '_', s)

class CaptureAddon:
    def request(self, flow: http.HTTPFlow):
        try:
            path = flow.request.path.split("?")[0]  # strip query string
            is_agent = "/api/agent/" in path
            is_ide = "/api/ide/" in path
            
            if not (is_agent or is_ide):
                return
            
            url = flow.request.pretty_url
            body_len = len(flow.request.content or b'')
            print(f"\n{'='*60}")
            print(f"[REQ] {flow.request.method} {url}")
            print(f"[REQ] Body size: {body_len} bytes")
            
            ts = int(time.time() * 1000)
            path_safe = safe_filename(path.replace("/", "_").strip("_"))
            prefix = os.path.join(CAPTURE_DIR, f"{ts}_{path_safe}")
            
            # Save headers
            headers_dict = dict(flow.request.headers)
            with open(f"{prefix}_req_headers.json", "w", encoding="utf-8") as f:
                json.dump(headers_dict, f, indent=2, ensure_ascii=False)
            
            # Save body
            if flow.request.content:
                with open(f"{prefix}_req_body.bin", "wb") as f:
                    f.write(flow.request.content)
                
                # Try JSON parse
                try:
                    body_json = json.loads(flow.request.content)
                    with open(f"{prefix}_req_body.json", "w", encoding="utf-8") as f:
                        json.dump(body_json, f, indent=2, ensure_ascii=False)
                    
                    # Print key fields
                    for key in ["function", "agent_type", "model_name", "session_id", "agent_id"]:
                        if key in body_json:
                            print(f"  {key}: {body_json[key]}")
                    print(f"  top_keys: {list(body_json.keys())}")
                except Exception:
                    pass
                
            print(f"[SAVED] {prefix}")
            
        except Exception as e:
            print(f"[ERROR in request] {e}")
            traceback.print_exc()
    
    def response(self, flow: http.HTTPFlow):
        try:
            path = flow.request.path.split("?")[0]
            is_agent = "/api/agent/" in path
            is_ide = "/api/ide/" in path
            
            if not (is_agent or is_ide):
                return
            
            url = flow.request.pretty_url
            resp_len = len(flow.response.content or b'')
            print(f"[RESP] {flow.response.status_code} ({resp_len} bytes) for {url}")
            
            ts = int(time.time() * 1000)
            path_safe = safe_filename(path.replace("/", "_").strip("_"))
            prefix = os.path.join(CAPTURE_DIR, f"{ts}_{path_safe}")
            
            if flow.response.content:
                with open(f"{prefix}_resp.bin", "wb") as f:
                    f.write(flow.response.content)
                try:
                    resp_json = json.loads(flow.response.content)
                    with open(f"{prefix}_resp.json", "w", encoding="utf-8") as f:
                        json.dump(resp_json, f, indent=2, ensure_ascii=False)
                except Exception:
                    pass
        except Exception as e:
            print(f"[ERROR in response] {e}")
            traceback.print_exc()

addons = [CaptureAddon()]
