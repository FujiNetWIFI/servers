#!/bin/bash

convert -size 320x192 xc:black - \
  -resize 320x192 -gravity center -compose over -composite pnm:- \
  | convert - -monochrome pbm:- \
  | ./pbm_to_gr8.php

