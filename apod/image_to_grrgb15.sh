#!/bin/bash

uuid=$(uuidgen)
fname="tmp_cv15_${uuid}"
dither="-dither FloydSteinberg"

rm img/${fname}.pnm
rm img/${fname}.cv15

convert - -fuzz 1% -trim +repage pnm:- \
| convert -size 320x192 xc:black - -resize 320x192 -gravity center -compose over \
  -composite pnm:- > img/${fname}.pnm

convert img/${fname}.pnm -channel R -separate -resize 160x192\! \
  -set colorspace Gray ${dither} -depth 8 pgm:- \
  > img/${fname}_r.pgm
convert img/${fname}.pnm -channel G -separate -resize 160x192\! \
  -set colorspace Gray ${dither} -depth 8 pgm:- \
  > img/${fname}_g.pgm
convert img/${fname}.pnm -channel B -separate -resize 160x192\! \
  -set colorspace Gray ${dither} -depth 8 pgm:- \
  > img/${fname}_b.pgm

cat img/${fname}_r.pgm | ./pgm_to_gr15.php 0 > img/${fname}.cv15
cat img/${fname}_g.pgm | ./pgm_to_gr15.php 1 >> img/${fname}.cv15
cat img/${fname}_b.pgm | ./pgm_to_gr15.php 2 >> img/${fname}.cv15

./interleave.php < img/${fname}.cv15

# Clean up
rm img/${fname}.pnm
rm img/${fname}_r.pgm
rm img/${fname}_g.pgm
rm img/${fname}_b.pgm
rm img/${fname}.cv15

