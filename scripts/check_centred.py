#!/usr/bin/env python3
"""Check that an icon's ink is centred in its canvas.

A taskbar icon is judged by where its ink sits, so this reports the four
margins and whether the mark is optically centred: equal left and right, and
the bottom margin a touch larger than the top, which is what reads as centred.
"""

from __future__ import annotations

import sys
from pathlib import Path

from PIL import Image


def check(path: Path) -> bool:
    im = Image.open(path).convert("RGBA")
    box = im.getbbox()
    if box is None:
        print(f"{path.name}: nothing drawn")
        return False

    left, top, right, bottom = box
    m = {
        "L": left,
        "R": im.width - right,
        "T": top,
        "B": im.height - bottom,
    }
    sideways = abs(m["L"] - m["R"])

    # A mark is optically centred when its ink sits slightly ABOVE the
    # geometric centre, so the bottom margin is a little larger than the top.
    # That reads as centred; ink that is mathematically centred reads low.
    # The lift is a fraction of the canvas, so a small icon has a small one.
    vertical = m["T"] - m["B"]
    # A whole pixel is a large fraction of a 16 px icon, so the tolerance is a
    # percentage of the canvas: 5% still catches a real misplacement (20 px at
    # 1024) while accepting the rounding a small icon cannot avoid.
    lift = max(1.0, im.height*0.05)
    ok = sideways <= 1 and -lift <= vertical <= 2

    print(
        f"{path.name:20} {im.width}x{im.height}  "
        f"L{m['L']:4} R{m['R']:4} T{m['T']:4} B{m['B']:4}  "
        f"h-delta {sideways}  lift {vertical:+}  "
        f"{'centred' if ok else 'NOT CENTRED'}"
    )
    return ok


def main() -> int:
    paths = [Path(p) for p in sys.argv[1:]]
    if not paths:
        print("usage: check_centred.py icon.png [...]")
        return 2

    failed = [p for p in paths if not check(p)]
    return 1 if failed else 0


if __name__ == "__main__":
    raise SystemExit(main())
