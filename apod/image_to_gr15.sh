#!/bin/bash

convert -brightness-contrast 30x100 \
  -size 320x192 xc:black - -resize 320x192 -gravity center -compose over -composite pnm:- \
  | convert - -resize 160x192\! -set colorspace Gray -dither FloydSteinberg -depth 4 pgm:- \
  | ./pgm_to_gr15.php

