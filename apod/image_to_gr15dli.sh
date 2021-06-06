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

touch img/${fname}.gr15dli.img
touch img/${fname}.gr15dli.pal

for i in `seq 0 191`; do
  rm img/${fname}.gr15dli.line

  convert img/${fname}.pnm \
    -crop 320x1+0+${i} pnm:- \
  | convert - -depth 8 +dither -colors 4 pnm:- \
  | convert - +dither -remap atari128.ppm -interpolate nearest pnm:- \
  | ./ppm_to_gr15.php 1 \
  > img/${fname}.gr15dli.line

  dd iflag=count_bytes,skip_bytes bs=1 count=40 \
    < img/${fname}.gr15dli.line >> img/${fname}.gr15dli.img
  dd iflag=count_bytes,skip_bytes skip=40 bs=1 count=4 \
    < img/${fname}.gr15dli.line >> img/${fname}.gr15dli.pal

done

rm img/${fname}.pnm
rm img/${fname}.gr15dli.line

cat img/${fname}.gr15dli.img
cat img/${fname}.gr15dli.pal

rm img/${fname}.gr15dli.img
rm img/${fname}.gr15dli.pal

