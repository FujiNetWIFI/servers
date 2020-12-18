#!/usr/bin/php
<?php
/*
P4
320 192
*/

for ($i = 0; $i < 2; $i++) {
  fgets(STDIN);
}

for ($y = 0; $y < 192; $y++) {
  for ($x = 0; $x < 320; $x += 8) {
    $c = ord(fgetc(STDIN));
    fwrite(STDOUT, chr(255 - $c), 1);
  }
}

