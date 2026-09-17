import sys
sys.stdout.reconfigure(encoding='utf-8')
import json

with open("scripts/captured/agent_task_resp.bin", "rb") as f:
    raw = f.read()

text = raw.decode("utf-8", errors="replace")
lines = text.split("\n")

events = []
current_event = {}

for line in lines:
    line = line.strip()
    if not line:
        if current_event:
            events.append(current_event)
            current_event = {}
        continue
    if line.startswith("id:"):
        current_event["id"] = line[3:]
    elif line.startswith("event:"):
        current_event["event"] = line[6:]
    elif line.startswith("data:"):
        data_str = line[5:]
        try:
            current_event["data"] = json.loads(data_str)
        except:
            current_event["data"] = data_str

if current_event:
    events.append(current_event)

print(f"Total SSE events: {len(events)}")
for i, ev in enumerate(events):
    ev_name = ev.get("event")
    ev_id = ev.get("id")
    ev_data = ev.get("data")
    data_keys = list(ev_data.keys()) if isinstance(ev_data, dict) else "raw"
    print(f"[{i+1:2d}] id={ev_id} event={ev_name:18s} data_keys={data_keys}")
    if isinstance(ev_data, dict):
        if "thought" in ev_data:
            print(f"     thought: {ev_data.get('thought')[:60]}...")
        if "content" in ev_data:
            print(f"     content: {ev_data.get('content')[:60]}...")
        if "delta" in ev_data:
            print(f"     delta: {ev_data.get('delta')[:60]}...")
        if "message" in ev_data:
            print(f"     message: {ev_data.get('message')[:60]}...")
        if "tokens" in ev_data:
            print(f"     tokens: {ev_data.get('tokens')}")

