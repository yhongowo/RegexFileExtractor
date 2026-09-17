"""Regenerate the renamed UI font; requires fonttools==4.60.2 (development only).

Usage: python scripts/subset-font.py /path/to/NotoSansCJKsc-Regular.otf
Source: https://github.com/notofonts/noto-cjk/blob/main/Sans/OTF/SimplifiedChinese/NotoSansCJKsc-Regular.otf
The glyph set includes GB2312, Latin characters, and every UI source character.
Other characters use Fyne's system font fallback; do not disable that fallback.
"""
import hashlib
from pathlib import Path
import sys

from fontTools import subset
from fontTools.ttLib import TTFont

root = Path(__file__).resolve().parent.parent
source = Path(sys.argv[1])
chars = set(chr(i) for i in range(0x20, 0x250))
for lead in range(0xA1, 0xF8):
    for trail in range(0xA1, 0xFF):
        try:
            chars.update(bytes([lead, trail]).decode("gb2312"))
        except UnicodeDecodeError:
            pass
for path in (root / "internal/ui").glob("*.go"):
    chars.update(path.read_text(encoding="utf-8"))
chars.update("繁體中文測試資料夾檔案路徑選擇複製樣本")
font = TTFont(source, recalcTimestamp=False)
options = subset.Options()
options.layout_features = ["kern", "liga"]
options.name_IDs = ["*"]
options.name_legacy = True
options.name_languages = ["*"]
options.notdef_outline = True
subsetter = subset.Subsetter(options=options)
subsetter.populate(unicodes=sorted(ord(c) for c in chars))
subsetter.subset(font)
for record in font["name"].names:
    if record.nameID in (1, 3, 4, 6, 16):
        name = "RFESans-Regular" if record.nameID == 6 else "RFE Sans"
        record.string = name.encode(record.getEncoding())
font["CFF "].cff.fontNames = ["RFESans-Regular"]
top = font["CFF "].cff.topDictIndex[0]
top.FamilyName = "RFE Sans"
top.FullName = "RFE Sans Regular"
output = root / "assets/RFESans-Regular.otf"
font.save(output)
(root / "assets/font-unicodes.txt").write_text(
    "\n".join(f"{code:04X}" for code in sorted(font.getBestCmap())) + "\n",
    encoding="ascii",
)
print(f"Source SHA256: {hashlib.sha256(source.read_bytes()).hexdigest()}")
print(f"Output: {output.stat().st_size:,} bytes; {len(font.getBestCmap()):,} characters")
