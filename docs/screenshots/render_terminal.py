#!/usr/bin/env python3
"""Render captured terminal output as a terminal-window PNG.

The screenshots in this directory are real output from the demo CLI and the test
suite: the text is captured verbatim (see README.md in this directory) and this
script only adds the window frame and syntax colouring.

Usage:
    python3 render_terminal.py <transcript.txt> <title> <output.png> [--size 15]

Requires Pillow:  python3 -m pip install pillow
"""
import argparse
import re

from PIL import Image, ImageDraw, ImageFilter, ImageFont

MONO = "/usr/share/fonts/truetype/dejavu/DejaVuSansMono.ttf"
MONO_BOLD = "/usr/share/fonts/truetype/dejavu/DejaVuSansMono-Bold.ttf"

BG = (13, 17, 23, 255)
TITLEBAR = (22, 27, 34, 255)
BORDER = (48, 54, 61, 255)
SHADOW = (0, 0, 0, 110)
TEXT = (201, 209, 217, 255)
DIM = (139, 148, 158, 255)
WHITE = (230, 237, 243, 255)
GREEN = (63, 185, 80, 255)
BLUE = (88, 166, 255, 255)
KEY = (121, 192, 255, 255)
STRING = (165, 214, 255, 255)
NUMBER = (240, 136, 62, 255)
PUNCT = (139, 148, 158, 255)
YELLOW = (210, 168, 255, 255)


def load_font(size, bold=False):
    return ImageFont.truetype(MONO_BOLD if bold else MONO, size)


def json_segments(line, regular, bold):
    """Split a JSON line into coloured (text, colour, font) runs."""
    runs = []
    index = 0
    length = len(line)
    punctuation = ""

    def flush():
        nonlocal punctuation
        if punctuation:
            runs.append((punctuation, PUNCT, regular))
            punctuation = ""

    while index < length:
        char = line[index]

        if char == '"':
            end = index + 1
            while end < length:
                if line[end] == "\\":
                    end += 2
                    continue
                if line[end] == '"':
                    end += 1
                    break
                end += 1
            flush()
            token = line[index:end]
            is_key = line[end:].lstrip().startswith(":")
            runs.append((token, KEY if is_key else STRING, regular))
            index = end
            continue

        if char.isdigit() or (char == "-" and index + 1 < length and line[index + 1].isdigit()):
            flush()
            end = index + 1
            while end < length and (line[end].isdigit() or line[end] in ".eE+-"):
                end += 1
            runs.append((line[index:end], NUMBER, regular))
            index = end
            continue

        if line.startswith(("true", "false", "null"), index):
            flush()
            word = "false" if line.startswith("false", index) else line[index : index + 4]
            runs.append((word, NUMBER, regular))
            index += len(word)
            continue

        punctuation += char
        index += 1

    flush()
    return runs


def result_segments(line, regular, bold, prefix):
    """Colour a `go test` result line, dimming the timing/coverage tail."""
    runs = list(prefix)
    rest = line[len("".join(part[0] for part in prefix)) :]
    for marker in ("coverage:", "[no test files]"):
        position = rest.find(marker)
        if position >= 0:
            runs.append((rest[:position], TEXT, regular))
            runs.append((rest[position:], DIM, regular))
            return runs
    runs.append((rest, TEXT, regular))
    return runs


def line_segments(line, regular, bold):
    """Return the coloured runs for one transcript line."""
    if line == "":
        return []
    if line.startswith("$ "):
        return [("$ ", GREEN, bold), (line[2:], WHITE, bold)]
    if line.startswith("ok "):
        return result_segments(line, regular, bold, [("ok ", GREEN, bold)])
    if line.startswith("?"):
        return result_segments(line, regular, bold, [("? ", YELLOW, bold)])
    if line.startswith(("{", "}")) or (line.startswith("  ") and line.lstrip().startswith('"')):
        return json_segments(line, regular, bold)
    if "coverage:" in line and not line.startswith("  "):
        position = line.find("coverage:")
        return [(line[:position], TEXT, regular), (line[position:], DIM, regular)]
    if line.startswith("  "):
        # Keep the original padding so key/value columns stay aligned.
        match = re.match(r"^(  )(.+?)(\s{2,})(.*)$", line)
        if match:
            key, separator, value = match.group(2), match.group(3), match.group(4)
            value_colour = GREEN if key.strip() == "reply with" else WHITE
            return [(key, DIM, regular), (separator, DIM, regular), (value, value_colour, regular)]
    return [(line, BLUE, bold)]


def render(transcript_path, title, out_path, size=15):
    scale = 2
    pad_x, pad_top, pad_bottom = 26 * scale, 18 * scale, 22 * scale
    title_height = 36 * scale

    regular = load_font(size * scale)
    bold = load_font(size * scale, bold=True)
    title_font = load_font(int(size * 0.86 * scale))

    lines = [line.expandtabs(8) for line in open(transcript_path).read().splitlines()]
    while lines and lines[-1] == "":
        lines.pop()

    ascent, descent = regular.getmetrics()
    line_height = int((ascent + descent) * 1.42)

    widths = [
        sum(font.getlength(text) for text, _, font in (line_segments(line, regular, bold) or [("", TEXT, regular)]))
        for line in lines
    ]
    content_width = max(max(widths) if widths else 0, 420 * scale)

    window_width = int(content_width + pad_x * 2)
    window_height = int(title_height + pad_top + line_height * len(lines) + pad_bottom)
    margin = 18 * scale
    radius = 11 * scale

    image = Image.new("RGBA", (window_width + margin * 2, window_height + margin * 2), (0, 0, 0, 0))

    shadow = Image.new("RGBA", image.size, (0, 0, 0, 0))
    ImageDraw.Draw(shadow).rounded_rectangle(
        (margin, margin + 6 * scale, margin + window_width, margin + window_height + 6 * scale),
        radius=radius,
        fill=SHADOW,
    )
    image.alpha_composite(shadow.filter(ImageFilter.GaussianBlur(9 * scale)))

    draw = ImageDraw.Draw(image)
    draw.rounded_rectangle(
        (margin, margin, margin + window_width, margin + window_height),
        radius=radius,
        fill=BG,
        outline=BORDER,
        width=scale,
    )

    # Title bar: rounded at the top, square where it meets the terminal body.
    draw.rounded_rectangle(
        (margin, margin, margin + window_width, margin + title_height + radius),
        radius=radius,
        fill=TITLEBAR,
    )
    draw.rectangle(
        (margin + scale, margin + title_height - radius, margin + window_width - scale, margin + title_height),
        fill=TITLEBAR,
    )
    draw.line(
        (margin + scale, margin + title_height, margin + window_width - scale, margin + title_height),
        fill=(33, 38, 45, 255),
        width=scale,
    )

    dot_y = margin + title_height // 2
    dot_radius = 6 * scale
    for index, colour in enumerate([(255, 95, 87), (254, 188, 46), (40, 200, 64)]):
        centre_x = margin + pad_x - 4 * scale + index * 20 * scale
        draw.ellipse(
            (centre_x - dot_radius, dot_y - dot_radius, centre_x + dot_radius, dot_y + dot_radius),
            fill=colour + (255,),
        )

    title_width = title_font.getlength(title)
    draw.text(
        (margin + (window_width - title_width) / 2, dot_y - int(size * 0.6 * scale)),
        title,
        font=title_font,
        fill=DIM,
    )

    y = margin + title_height + pad_top
    for line in lines:
        x = margin + pad_x
        for text, colour, font in line_segments(line, regular, bold):
            if text:
                draw.text((x, y), text, font=font, fill=colour)
            x += font.getlength(text)
        y += line_height

    image = image.resize((image.width // scale, image.height // scale), Image.LANCZOS)
    # Quantise so the committed PNGs stay small; RGBA alpha is preserved.
    image = image.quantize(colors=256, method=Image.Quantize.FASTOCTREE)
    image.save(out_path, optimize=True)
    print(f"wrote {out_path} ({image.width}x{image.height})")


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("transcript", help="file containing the captured terminal output")
    parser.add_argument("title", help="text shown in the window title bar")
    parser.add_argument("output", help="PNG file to write")
    parser.add_argument("--size", type=int, default=15, help="font size in pixels (default 15)")
    args = parser.parse_args()

    render(args.transcript, args.title, args.output, args.size)


if __name__ == "__main__":
    main()
