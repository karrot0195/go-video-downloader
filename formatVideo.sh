#!/usr/bin/env bash

set -e

INPUT="$1"

if [[ -z "$INPUT" ]]; then
    echo "Usage: $0 input.mp4 [output.mp4]"
    exit 1
fi

if [[ ! -f "$INPUT" ]]; then
    echo "❌ File not found: $INPUT"
    exit 1
fi

OUTPUT="${2:-${INPUT%.*}_covered.mp4}"

if ! command -v ffmpeg >/dev/null 2>&1; then
    echo "❌ ffmpeg not installed"
    echo "sudo dnf install ffmpeg"
    exit 1
fi

if ! command -v ffprobe >/dev/null 2>&1; then
    echo "❌ ffprobe not found"
    echo "sudo dnf install ffmpeg"
    exit 1
fi

# Get video dimensions
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

read -p "X (horizontal): " X
read -p "Y (vertical):   " Y
read -p "Width:          " W
read -p "Height:         " H

echo
read -p "Start time (seconds, default 0): " START
read -p "End time (Enter for full video): " END

START="${START:-0}"

if [[ -z "$END" ]]; then
    ENABLE="gte(t,$START)"
else
    ENABLE="between(t,$START,$END)"
fi

echo
echo "Processing..."
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
echo "✅ Done!"
echo "📁 $OUTPUT"