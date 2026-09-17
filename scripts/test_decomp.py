import zlib
import gzip

with open("scripts/captured/detail_param_response.bin", "rb") as f:
    data = f.read()

print(f"Data len: {len(data)}")
print("Hex 32:", data[:32].hex())

# Try brotli
try:
    import brotli
    decomp = brotli.decompress(data)
    print("Brotli decompress SUCCESS! len=", len(decomp))
except Exception as e:
    print("Brotli failed:", e)

# Try zlib
try:
    decomp = zlib.decompress(data)
    print("zlib decompress SUCCESS! len=", len(decomp))
except Exception as e:
    print("zlib failed:", e)

# Try gzip
try:
    decomp = gzip.decompress(data)
    print("gzip decompress SUCCESS! len=", len(decomp))
except Exception as e:
    print("gzip failed:", e)

# Try zstandard
try:
    import zstandard
    dctx = zstandard.ZstdDecompressor()
    decomp = dctx.decompress(data)
    print("zstandard decompress SUCCESS! len=", len(decomp))
except Exception as e:
    print("zstandard failed:", e)

# What if there is a 4-byte or 5-byte length prefix (like gRPC)?
for skip in [1, 2, 4, 5, 8, 12, 16]:
    sub = data[skip:]
    try:
        import brotli
        d = brotli.decompress(sub)
        print(f"Brotli skip {skip} SUCCESS! len={len(d)}")
    except:
        pass
    try:
        d = zlib.decompress(sub)
        print(f"zlib skip {skip} SUCCESS! len={len(d)}")
    except:
        pass
    try:
        d = gzip.decompress(sub)
        print(f"gzip skip {skip} SUCCESS! len={len(d)}")
    except:
        pass

