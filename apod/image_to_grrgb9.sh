#!/bin/bash

uuid=$(uuidgen)
fname="tmp_cv9_${uuid}"

rm img/${fname}.pnm
rm img/${fname}.cv9

convert -size 320x192 xc:black - -resize 320x192 -gravity center -compose over \
  -composite pnm:- > img/${fname}.pnm

convert img/${fname}.pnm -channel R -separate -resize 80x192\! \
  -set colorspace Gray -dither FloydSteinberg -depth 8 pgm:- \
  > img/${fname}_r.pgm
convert img/${fname}.pnm -channel G -separate -resize 80x192\! \
  -set colorspace Gray -dither FloydSteinberg -depth 8 pgm:- \
  > img/${fname}_g.pgm
convert img/${fname}.pnm -channel B -separate -resize 80x192\! \
  -set colorspace Gray -dither FloydSteinberg -depth 8 pgm:- \
  > img/${fname}_b.pgm

cat img/${fname}_r.pgm | ./pgm_to_gr9.php > img/${fname}.cv9
cat img/${fname}_g.pgm | ./pgm_to_gr9.php >> img/${fname}.cv9
cat img/${fname}_b.pgm | ./pgm_to_gr9.php >> img/${fname}.cv9

./interleave.php < img/${fname}.cv9

# FIXME: Clean up
