import base64
import struct
import zlib
import gzip

with open("scripts/captured/agent_task_req_body.bin", "rb") as f:
    raw = f.read()

data = base64.b64decode(raw)
print(f"Decoded size: {len(data)} bytes")
print("Hex 32:", data[:32].hex())

for i in range(32):
    c = chr(data[i]) if 32 <= data[i] <= 126 else "."
    print(f"[{i:02d}] 0x{data[i]:02x} ({data[i]:3d}) '{c}'")

# Check magic and fields
magic = data[:4]
print(f"Magic: {magic.hex().upper()}")

# Check uint16/uint32 at offsets
for off in range(4, 28, 2):
    val_le16 = struct.unpack("<H", data[off:off+2])[0]
    val_be16 = struct.unpack(">H", data[off:off+2])[0]
    print(f"Offset {off:02d}: LE16={val_le16:6d} (0x{val_le16:04x}), BE16={val_be16:6d} (0x{val_be16:04x})")

for off in range(4, 28, 4):
    val_le32 = struct.unpack("<I", data[off:off+4])[0]
    val_be32 = struct.unpack(">I", data[off:off+4])[0]
    print(f"Offset {off:02d}: LE32={val_le32:10d} (0x{val_le32:08x}), BE32={val_be32:10d} (0x{val_be32:08x})")

