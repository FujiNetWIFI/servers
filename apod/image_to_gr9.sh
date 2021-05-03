#!/bin/bash

convert - -fuzz 1% -trim +repage pnm:- \
| convert -size 320x192 xc:black - -resize 320x192 -gravity center \
  -compose over -composite pnm:- \
| convert - -resize 80x192\! -set colorspace Gray -dither FloydSteinberg \
  -depth 8 pgm:- \
| ./pgm_to_gr9.php

