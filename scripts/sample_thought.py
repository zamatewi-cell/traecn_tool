import sys
sys.stdout.reconfigure(encoding='utf-8')
import json

with open("scripts/captured/agent_task_resp.bin", "rb") as f:
    text = f.read().decode("utf-8")

for part in text.split("\n\n"):
    if "event:thought" in part:
        lines = part.split("\n")
        for l in lines:
            if l.startswith("data:"):
                d = json.loads(l[5:])
                print("Thought event data sample:")
                print(json.dumps(d, indent=2, ensure_ascii=False))
                sys.exit(0)

