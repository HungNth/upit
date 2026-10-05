# Upit Branding Assets

This directory contains the canonical branding assets and instructions for reproducing platform derivatives.

## Source

- `logo.svg`: Canonical master vector logo (squircle background `#09090b`, emerald motif `#34d399` / `#10b981`).

## Committed Derivatives

- `logo.png`: High-resolution 1024x1024 RGBA master render.
- `app.ico`: Windows multi-resolution icon containing layers:
  - 16x16, 24x24, 32x32, 48x48, 64x64, 128x128, 256x256.
- `AppIcon.icns`: Apple Icon Image container for macOS bundles containing standard and retina resolutions:
  - 16x16 (`icp4`), 32x32 (`ic11`, `icp5`), 64x64 (`ic12`, `icp6`), 128x128 (`ic07`), 256x256 (`ic13`, `ic08`), 512x512 (`ic14`, `ic09`), 1024x1024 (`ic10`).

## Reproducing Derivatives

To reproduce all platform assets from `logo.svg`, run the following commands (requires Python with `pillow` and `ImageMagick` or equivalent SVG rasterizer):

### 1. Render High-Resolution PNG (1024x1024)
```bash
magick -background none -density 1024 assets/branding/logo.svg -resize 1024x1024 assets/branding/logo.png
```

### 2. Generate Windows Multi-Layer Icon (`app.ico`)
```python
from PIL import Image

im = Image.open("assets/branding/logo.png")
sizes = [(16, 16), (24, 24), (32, 32), (48, 48), (64, 64), (128, 128), (256, 256)]
im.save("assets/branding/app.ico", format="ICO", sizes=sizes)
```

### 3. Generate Apple Icon Image (`AppIcon.icns`)
```python
import io, struct
from PIL import Image

im = Image.open("assets/branding/logo.png")
icns_map = [
    (b'icp4', 16),
    (b'ic11', 32),
    (b'icp5', 32),
    (b'ic12', 64),
    (b'icp6', 64),
    (b'ic07', 128),
    (b'ic13', 256),
    (b'ic08', 256),
    (b'ic14', 512),
    (b'ic09', 512),
    (b'ic10', 1024),
]
elements = bytearray()
for ostype, dim in icns_map:
    resized = im.resize((dim, dim), Image.Resampling.LANCZOS)
    buf = io.BytesIO()
    resized.save(buf, format="PNG")
    data = buf.getvalue()
    elements.extend(ostype + struct.pack(">I", 8 + len(data)) + data)

with open("assets/branding/AppIcon.icns", "wb") as f:
    f.write(b'icns' + struct.pack(">I", 8 + len(elements)) + elements)
```

### 4. Generate Windows PE Resource (`icon_windows_amd64.syso`)
On Windows (with MinGW `windres` or `llvm-rc`):
```bash
# Prepare icon.rc:
# 1 ICON "assets/branding/app.ico"
windres -i icon.rc -O coff -F pe-x86-64 -o cmd/upit-desktop/icon_windows_amd64.syso
```
