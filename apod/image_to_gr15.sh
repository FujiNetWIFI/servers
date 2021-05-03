#!/bin/bash

convert - -fuzz 1% -trim +repage pnm:- \
| convert -size 320x192 xc:black - \
  -resize 320x192 -gravity center -compose over -composite pnm:- \
| convert - -resize 160x192\! -depth 8 +dither -remap atari256.ppm pnm:- \
| convert - -depth 8 +dither -colors 4 pnm:- \
| convert - +dither -remap atari128.ppm -interpolate nearest pnm:- \
| ./ppm_to_gr15.php

