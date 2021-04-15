#!/usr/bin/php
<?php
for ($rgb = 0; $rgb < 3; $rgb++) {
  for ($y = 0; $y < 192; $y++) {
    $row[$rgb][$y] = fread(STDIN, 40);
  }
}

for ($rgb = 0; $rgb < 3; $rgb++) {
  for ($y = 0; $y < 192; $y += 3) {
    $offset = $rgb;
    fwrite(STDOUT, $row[$offset][$y], 40);

    $offset = ($rgb + 1) % 3;
    fwrite(STDOUT, $row[$offset][$y + 1], 40);

    $offset = ($rgb + 2) % 3;
    fwrite(STDOUT, $row[$offset][$y + 2], 40);
  }
}

