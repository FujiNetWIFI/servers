#!/bin/bash

convert -size 320x192 xc:black - -resize 320x192 -gravity center -compose over -composite pnm:- \
  | convert - -resize 80x192\! -remap atari256.ppm -dither Riemersma -depth 8 ppm:- \
  | ./ppm_to_apac.php

# TODO: We can experiment with brightness/saturation tweaks (keep third number, hue, at 100); these are percentages:
#  | convert - -modulate 110,120,100 ppm:- \

