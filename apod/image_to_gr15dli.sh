#!/bin/bash

uuid=$(uuidgen)
fname="tmp_gr15dli_${uuid}"

rm img/${fname}.pnm

convert - -fuzz 1% -trim +repage pnm:- \
| convert -size 320x192 \
  xc:black - \
  -resize 320x192 \
  -gravity center \
  -compose over \
  -composite pnm:- \
| convert - -resize 160x192\! -depth 8 +dither -remap atari256.ppm pnm:- \
> img/${fname}.pnm

touch img/${fname}.gr15dli

for i in `seq 0 191`; do
  convert img/${fname}.pnm \
    -crop 320x1+0+${i} pnm:- \
  | convert - -depth 8 +dither -colors 4 pnm:- \
  | convert - +dither -remap atari128.ppm -interpolate nearest pnm:- \
  | ./ppm_to_gr15.php 1 \
  >> img/${fname}.gr15dli
done

rm img/${fname}.pnm
cat img/${fname}.gr15dli
rm img/${fname}.gr15dli

