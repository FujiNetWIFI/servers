#!/bin/bash

convert -brightness-contrast 30x100 \
  -size 320x192 xc:black - -resize 320x192 -gravity center -compose over -composite pnm:- \
  | convert - -set colorspace Gray -dither FloydSteinberg -depth 1 pbm:- \
  | ./pbm_to_gr8.php

