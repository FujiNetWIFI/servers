#!/bin/bash

convert -size 320x192 xc:black - \
  -resize 320x192 -gravity center -compose over -composite pnm:- \
  | convert - -resize 160x192\! -depth 8 -remap atari256.ppm -colors 4 ppm:- \
  | convert - -remap atari256.ppm ppm:- \
  | ./ppm_to_gr15.php

