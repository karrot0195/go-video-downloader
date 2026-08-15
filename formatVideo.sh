#!/usr/bin/env bash

set -e

INPUT="$1"

if [[ -z "$INPUT" ]]; then
    echo "Usage: $0 input.mp4 [output.mp4]"
    exit 1
fi

if [[ ! -f "$INPUT" ]]; then
    echo "❌ Không tìm thấy file: $INPUT"
    exit 1
fi

OUTPUT="${2:-${INPUT%.*}_covered.mp4}"

if ! command -v ffmpeg >/dev/null 2>&1; then
    echo "❌ Chưa cài ffmpeg"
    echo "sudo dnf install ffmpeg"
    exit 1
fi

if ! command -v ffprobe >/dev/null 2>&1; then
    echo "❌ Chưa có ffprobe"
    echo "sudo dnf install ffmpeg"
    exit 1
fi

# Lấy kích thước video
IFS=x read -r WIDTH HEIGHT < <(
    ffprobe -v error \
        -select_streams v:0 \
        -show_entries stream=width,height \
        -of csv=p=0:s=x \
        "$INPUT"
)

echo
echo "================================"
echo "Input  : $INPUT"
echo "Size   : ${WIDTH}x${HEIGHT}"
echo "Output : $OUTPUT"
echo "================================"
echo

read -p "X (ngang): " X
read -p "Y (dọc):   " Y
read -p "Width:     " W
read -p "Height:    " H

echo
read -p "Bắt đầu (giây, mặc định 0): " START
read -p "Kết thúc (Enter = hết video): " END

START="${START:-0}"

if [[ -z "$END" ]]; then
    ENABLE="gte(t,$START)"
else
    ENABLE="between(t,$START,$END)"
fi

echo
echo "Đang xử lý..."
echo "Box: x=$X y=$Y w=$W h=$H"
echo "Time: $START -> ${END:-END}"
echo

ffmpeg -hide_banner -i "$INPUT" \
    -vf "drawbox=x=$X:y=$Y:w=$W:h=$H:color=black:t=fill:enable='$ENABLE'" \
    -c:v libx264 \
    -preset veryfast \
    -crf 18 \
    -c:a copy \
    -y "$OUTPUT"

echo
echo "✅ Xong!"
echo "📁 $OUTPUT"