"""Extract ALL flows from flows.bin, including llm_raw_chat"""
import sys
sys.path.insert(0, '')

from mitmproxy.io import FlowReader
import json
import os

flows_path = os.path.join(os.path.dirname(__file__), 'captured', 'flows.bin')
out_dir = os.path.join(os.path.dirname(__file__), 'captured')

with open(flows_path, 'rb') as f:
    reader = FlowReader(f)
    for i, flow in enumerate(reader.stream()):
        url = flow.request.pretty_url
        method = flow.request.method
        status = flow.response.status_code if flow.response else 'N/A'
        req_len = len(flow.request.content) if flow.request.content else 0
        resp_len = len(flow.response.content) if flow.response and flow.response.content else 0
        
        # Only show API calls (skip monitoring etc)
        path = flow.request.path
        if '/api/' in path or 'llm' in path.lower():
            print(f"[{i}] {method} {path}")
            print(f"    Status: {status}, Req: {req_len}B, Resp: {resp_len}B")
            
            # Save headers
            headers = dict(flow.request.headers)
            
            # Check if body is encrypted (has x-bridge-transport: aha)
            transport = headers.get('x-bridge-transport', 'none')
            print(f"    Transport: {transport}")
            
            # For llm_raw_chat, save everything
            if 'llm_raw_chat' in path:
                prefix = f'llm_raw_chat_{i}'
                with open(os.path.join(out_dir, f'{prefix}_headers.json'), 'w') as hf:
                    json.dump(headers, hf, indent=2)
                with open(os.path.join(out_dir, f'{prefix}_req.bin'), 'wb') as rf:
                    rf.write(flow.request.content or b'')
                with open(os.path.join(out_dir, f'{prefix}_resp.bin'), 'wb') as rf:
                    rf.write(flow.response.content or b'')
                print(f"    -> Saved {prefix}_*")
                
                # Try to decode body
                body = flow.request.content
                if body and transport != 'aha':
                    try:
                        j = json.loads(body)
                        print(f"    Body (JSON): {json.dumps(j, indent=2)[:500]}")
                    except:
                        print(f"    Body (raw): {body[:200]}")
                elif body and transport == 'aha':
                    print(f"    Body: ENCRYPTED ({len(body)}B)")
                    
                # Response
                resp = flow.response.content
                if resp:
                    try:
                        text = resp.decode('utf-8')
                        print(f"    Response: {text[:500]}")
                    except:
                        print(f"    Response: binary {len(resp)}B")
            
            # For create_agent_task, check transport
            if 'create_agent_task' in path:
                print(f"    Body encrypted: {transport == 'aha'}")
            
            print()
