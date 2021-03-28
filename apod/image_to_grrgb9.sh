#!/bin/bash

rm img/tmp.pnm
rm img/tmp.cv9

convert -size 320x192 xc:black - -resize 320x192 -gravity center -compose over -composite pnm:- > img/tmp.pnm

convert img/tmp.pnm -channel R -separate -resize 80x192\! -set colorspace Gray -dither FloydSteinberg -depth 8 pgm:- > img/tmp_r.pgm
convert img/tmp.pnm -channel G -separate -resize 80x192\! -set colorspace Gray -dither FloydSteinberg -depth 8 pgm:- > img/tmp_g.pgm
convert img/tmp.pnm -channel B -separate -resize 80x192\! -set colorspace Gray -dither FloydSteinberg -depth 8 pgm:- > img/tmp_b.pgm

cat img/tmp_r.pgm | ./pgm_to_gr9.php > img/tmp.cv9
cat img/tmp_g.pgm | ./pgm_to_gr9.php >> img/tmp.cv9
cat img/tmp_b.pgm | ./pgm_to_gr9.php >> img/tmp.cv9

cat img/tmp.cv9

