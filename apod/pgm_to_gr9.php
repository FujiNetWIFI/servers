#!/usr/bin/php
<?php
/*
P5
80 192
255
*/

for ($i = 0; $i < 3; $i++) {
  fgets(STDIN);
}

for ($y = 0; $y < 192; $y++) {
  for ($x = 0; $x < 80; $x += 2) {
    $b1 = ord(fgetc(STDIN));
    $b2 = ord(fgetc(STDIN));

    $b1 = floor($b1 / 16);
    $b2 = floor($b2 / 16);

    $c = chr($b1 * 16 + $b2);
    fwrite(STDOUT, $c, 1);
  }
}

