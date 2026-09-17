"""Extract create_agent_task request/response from mitmproxy flow file."""
import json
import sys
from mitmproxy import io as mio
from mitmproxy import http

flows_path = "scripts/captured/flows.bin"
out_dir = "scripts/captured"

with open(flows_path, "rb") as f:
    reader = mio.FlowReader(f)
    idx = 0
    for flow in reader.stream():
        if not isinstance(flow, http.HTTPFlow):
            continue
        idx += 1
        path = flow.request.path.split("?")[0]
        
        # Print all API calls
        if "/api/" in path:
            body_len = len(flow.request.content or b'')
            resp_len = len(flow.response.content or b'') if flow.response and flow.response.content else 0
            print(f"[{idx}] {flow.request.method} {flow.request.pretty_url[:100]} req={body_len} resp={resp_len}")
        
        # Extract create_agent_task
        if "create_agent_task" in path:
            # Save request headers
            headers_dict = dict(flow.request.headers)
            with open(f"{out_dir}/agent_task_req_headers.json", "w", encoding="utf-8") as fout:
                json.dump(headers_dict, fout, indent=2, ensure_ascii=False)
            print(f"\n  => Saved request headers ({len(headers_dict)} headers)")
            
            # Save request body
            if flow.request.content:
                with open(f"{out_dir}/agent_task_req_body.bin", "wb") as fout:
                    fout.write(flow.request.content)
                print(f"  => Saved request body ({len(flow.request.content)} bytes)")
                
                try:
                    body_json = json.loads(flow.request.content)
                    with open(f"{out_dir}/agent_task_req_body.json", "w", encoding="utf-8") as fout:
                        json.dump(body_json, fout, indent=2, ensure_ascii=False)
                    print(f"  => Saved JSON request body")
                    print(f"  => Top-level keys: {list(body_json.keys())}")
                    for key in ["function", "agent_type", "model_name", "session_id", "agent_id"]:
                        if key in body_json:
                            val = body_json[key]
                            if isinstance(val, str) and len(val) > 100:
                                val = val[:100] + "..."
                            print(f"     {key}: {val}")
                except Exception as e:
                    print(f"  => Not JSON: {e}")
            
            # Save response
            if flow.response and flow.response.content:
                with open(f"{out_dir}/agent_task_resp.bin", "wb") as fout:
                    fout.write(flow.response.content)
                print(f"  => Saved response ({len(flow.response.content)} bytes)")
                
                # Try to parse SSE events
                try:
                    text = flow.response.content.decode("utf-8", errors="replace")
                    lines = text.split("\n")
                    event_count = sum(1 for l in lines if l.startswith("event:"))
                    print(f"  => Response has {event_count} SSE events")
                    # Print first few events
                    for line in lines[:20]:
                        if line.strip():
                            print(f"     {line[:200]}")
                except Exception:
                    pass
        
        # Also extract llm_raw_chat
        if "llm_raw_chat" in path:
            if flow.request.content:
                try:
                    body_json = json.loads(flow.request.content)
                    with open(f"{out_dir}/llm_raw_chat_req_body.json", "w", encoding="utf-8") as fout:
                        json.dump(body_json, fout, indent=2, ensure_ascii=False)
                    print(f"\n  => Saved llm_raw_chat body ({len(flow.request.content)} bytes)")
                    print(f"  => Keys: {list(body_json.keys())}")
                except Exception:
                    pass
