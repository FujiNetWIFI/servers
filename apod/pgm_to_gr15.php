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

    $b1 = dither($b1, $x + 0, $y + intval($argc[1]));
    $b2 = dither($b2, $x + 1, $y + intval($argc[1]));
    $b3 = dither($b3, $x + 2, $y + intval($argc[1]));
    $b4 = dither($b4, $x + 3, $y + intval($argc[1]));

    $c = chr($b1 * 64 + $b2 * 16 + $b3 * 4 + $b4);
    fwrite(STDOUT, $c, 1);
  }
}

function dither($n, $x, $y) {
  $pattern = array(
    0 => array(// 0
      array(0, 0, 0, 0),
      array(0, 0, 0, 0),
      array(0, 0, 0, 0),
      array(0, 0, 0, 0),
    ),
    1 => array(// 4
      array(1, 0, 1, 0),
      array(0, 0, 0, 0),
      array(1, 0, 1, 0),
      array(0, 0, 0, 0),
    ),
    2 => array(//7
      array(1, 0, 1, 0),
      array(0, 1, 0, 1),
      array(1, 0, 1, 0),
      array(0, 1, 0, 0),
    ),
    3 => array(//10
      array(1, 1, 0, 1),
      array(1, 0, 1, 0),
      array(0, 1, 0, 1),
      array(1, 0, 1, 1),
    ),
    4 => array(//13
      array(1, 0, 1, 1),
      array(1, 1, 1, 0),
      array(1, 0, 1, 1),
      array(1, 1, 1, 1),
    ),

    5 => array(// 0
      array(1, 1, 1, 1),
      array(1, 1, 1, 1),
      array(1, 1, 1, 1),
      array(1, 1, 1, 1),
    ),
    6 => array(// 4
      array(2, 1, 2, 1),
      array(1, 1, 1, 1),
      array(2, 1, 2, 1),
      array(1, 1, 1, 1),
    ),
    7 => array(//7
      array(2, 1, 2, 1),
      array(1, 2, 1, 2),
      array(2, 1, 2, 1),
      array(1, 2, 1, 1),
    ),
    8 => array(//10
      array(2, 2, 1, 2),
      array(2, 1, 2, 1),
      array(1, 2, 1, 2),
      array(2, 1, 2, 2),
    ),
    9 => array(//13
      array(2, 1, 2, 2),
      array(2, 2, 2, 1),
      array(2, 1, 2, 2),
      array(2, 2, 2, 2),
    ),

    10 => array(// 0
      array(2, 2, 2, 2),
      array(2, 2, 2, 2),
      array(2, 2, 2, 2),
      array(2, 2, 2, 2),
    ),
    11 => array(// 4
      array(3, 2, 3, 2),
      array(2, 2, 2, 2),
      array(3, 2, 3, 2),
      array(2, 2, 2, 2),
    ),
    12 => array(//7
      array(3, 2, 3, 2),
      array(2, 3, 2, 3),
      array(3, 2, 3, 2),
      array(2, 3, 2, 2),
    ),
    13 => array(//10
      array(3, 3, 2, 3),
      array(3, 2, 3, 2),
      array(2, 3, 2, 3),
      array(3, 2, 3, 3),
    ),
    14 => array(//13
      array(3, 2, 3, 3),
      array(3, 3, 3, 2),
      array(3, 2, 3, 3),
      array(3, 3, 3, 3),
    ),

    15 => array(// 0
      array(3, 3, 3, 3),
      array(3, 3, 3, 3),
      array(3, 3, 3, 3),
      array(3, 3, 3, 3),
    ),
  );

  $n = floor($n / 16);

  $n = $pattern[$n][$y % 4][$x % 4];

  return $n;
}

