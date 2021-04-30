#!/usr/bin/php
<?php
/*
P5
160 192
15
*/

for ($i = 0; $i < 3; $i++) {
  fgets(STDIN);
}

for ($y = 0; $y < 192; $y++) {
  for ($x = 0; $x < 160; $x += 4) {
    $b1 = ord(fgetc(STDIN));
    $b2 = ord(fgetc(STDIN));
    $b3 = ord(fgetc(STDIN));
    $b4 = ord(fgetc(STDIN));

    $b1 = floor($b1 / 64);
    $b2 = floor($b2 / 64);
    $b3 = floor($b3 / 64);
    $b4 = floor($b4 / 64);

    $c = chr($b1 * 64 + $b2 * 16 + $b3 * 4 + $b4);
    fwrite(STDOUT, $c, 1);
  }
}

