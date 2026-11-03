#!/usr/bin/env python3
"""Build Wand's app icons from the symbol SVG.

The symbol's ink is 800x608 inside a 1024 square, so it sits 24px high. A
taskbar icon is judged by where its ink sits, not by where its canvas sits, so
this centres the ink's bounding box optically: equal margins, and a 1.5% lift
because a shape that is exactly centred reads as low.

It writes:
  resources/icon.png        1024  the .exe, taskbar and window icon
  resources/icon-dev.png    1024  the dev build, with a dev band
  design/logo/renders/*.png       the size ladder, for looking at
"""

from __future__ import annotations

import subprocess
import sys
from pathlib import Path

from PIL import Image, ImageDraw

ROOT = Path(__file__).resolve().parents[1]
SYMBOL = ROOT / "design" / "logo" / "symbol.svg"
SYMBOL_SMALL = ROOT / "design" / "logo" / "symbol-small.svg"
RENDERS = ROOT / "design" / "logo" / "renders"
RESOURCES = ROOT / "resources"
RENDER = (
    Path.home()
    / ".config/opencode/skills/logo-design/scripts/render_png.py"
)

INK = "#05070f"
ACCENT = "#f2e9d8"
DEV = "#c8452f"

# opticalLift raises the mark above the geometric centre, as type does.
OPTICAL_LIFT = 0.015


def render_symbol(size: int, source: Path = SYMBOL) -> Image.Image:
    """Render the symbol at size, on transparent."""
    RENDERS.mkdir(parents=True, exist_ok=True)
    subprocess.run(
        [sys.executable, str(RENDER), str(source), "--out-dir", str(RENDERS),
         "--size", str(size)],
        check=True,
        capture_output=True,
    )
    return Image.open(RENDERS / (Path(source).stem + ".png")).convert("RGBA")


def centred(mark: Image.Image, size: int, margin: float) -> Image.Image:
    """Place the mark's ink centred in a square, with equal margins."""
    box = mark.getbbox()
    if box is None:
        raise SystemExit("the symbol drew nothing")

    ink = mark.crop(box)
    inner = int(round(size * (1 - margin * 2)))

    # Scale the ink so its longer side fills the inner square.
    scale = min(inner / ink.width, inner / ink.height)
    w = max(1, int(round(ink.width * scale)))
    h = max(1, int(round(ink.height * scale)))
    ink = ink.resize((w, h), Image.LANCZOS)

    canvas = Image.new("RGBA", (size, size), (0, 0, 0, 0))
    x = (size - w) // 2
    # A mark whose ink is exactly centred reads as low, so it is lifted by a
    # small fraction of the canvas. Subtracting moves it up.
    y = (size - h) // 2 - int(round(size * OPTICAL_LIFT))
    canvas.alpha_composite(ink, (x, y))
    return canvas


def tint(alpha: Image.Image, colour: str) -> Image.Image:
    """Recolour a mark, keeping its alpha."""
    solid = Image.new("RGBA", alpha.size, colour)
    solid.putalpha(alpha.getchannel("A"))
    return solid


def build_icon(size: int) -> Image.Image:
    """The mark as one colour, centred, for the taskbar and the .exe.

    Below 32 px the small-size cut is used: the full drawing's gaps close up at
    that size, so the letter is redrawn taller, with wider gaps and a bigger
    square, rather than scaled down.
    """
    source = SYMBOL_SMALL if size <= 32 else SYMBOL
    mark = render_symbol(1024, source)
    shape = centred(mark, size, margin=0.08 if size <= 32 else 0.10)
    return tint(shape, INK)


def build_dev_icon(size: int) -> Image.Image:
    """The mark with a corner band, so a dev build is never mistaken for the
    release. The mark is nudged off the band so the icon as a whole still
    reads as one centred object."""
    icon = build_icon(size)
    draw = ImageDraw.Draw(icon)

    band = int(round(size * 0.18))
    draw.rectangle([size - band, size - band, size, size], fill=DEV)
    return icon


def build_plate(size: int) -> Image.Image:
    """The mark on its dark plate, for contexts that need an opaque square."""
    icon = Image.new("RGBA", (size, size), (0, 0, 0, 0))
    draw = ImageDraw.Draw(icon)

    radius = int(round(size * 0.22))
    draw.rounded_rectangle([0, 0, size - 1, size - 1], radius=radius, fill=INK)

    mark = render_symbol(1024)
    shape = centred(mark, int(size * 0.92), margin=0.08)
    light = tint(shape, ACCENT)

    offset = (size - shape.width) // 2
    icon.alpha_composite(light, (offset, offset))
    return icon


def save(image: Image.Image, path: Path) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    image.save(path, "PNG")
    print(f"wrote {path.relative_to(ROOT)}  {image.width}x{image.height}")


def main() -> None:
    RESOURCES.mkdir(exist_ok=True)
    RENDERS.mkdir(parents=True, exist_ok=True)

    save(build_icon(1024), RESOURCES / "icon.png")
    save(build_dev_icon(1024), RESOURCES / "icon-dev.png")

    # The window and taskbar icon is the same mark, so the app carries one.
    save(build_icon(1024), Path(__file__).resolve().parents[1] / "internal" / "app" / "icon.png")

    for size in (512, 256, 128, 64, 48, 32, 24, 16):
        save(build_icon(size), RENDERS / f"icon-{size}.png")
        zoom = max(4, 224 // size)
        build_icon(size).resize((size * zoom, size * zoom), Image.NEAREST).save(
            RENDERS / f"zoom-{size}.png"
        )

    save(build_plate(1024), RENDERS / "plate-1024.png")


if __name__ == "__main__":
    main()
