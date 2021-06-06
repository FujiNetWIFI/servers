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

./ppm_to_gr15dli.php img/${fname}.pnm img/${fname}.gr15dli.img img/${fname}.gr15dli.pal

rm img/${fname}.pnm

cat img/${fname}.gr15dli.img
cat img/${fname}.gr15dli.pal

rm img/${fname}.gr15dli.img
rm img/${fname}.gr15dli.pal

