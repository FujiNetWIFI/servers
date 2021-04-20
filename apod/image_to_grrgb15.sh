#!/bin/bash

uuid=$(uuidgen)
fname="tmp_cv15_${uuid}"

rm img/${fname}.pnm
rm img/${fname}.cv15

convert -size 160x192 xc:black - -resize 160x192 -gravity center -compose over \
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

cat img/${fname}_r.pgm | ./pgm_to_gr15.php > img/${fname}.cv15
cat img/${fname}_g.pgm | ./pgm_to_gr15.php >> img/${fname}.cv15
cat img/${fname}_b.pgm | ./pgm_to_gr15.php >> img/${fname}.cv15

./interleave.php < img/${fname}.cv15

# FIXME: Clean up
