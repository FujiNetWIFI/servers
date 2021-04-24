#!/usr/bin/php
<?php
/* Skip header; assuming:
P6
80 192
255
*/

for ($i = 0; $i < 3; $i++) {
  fgets(STDIN);
}

/* Generate array to store image, so we can go over it a few times */
$px = array();
for ($y = 0; $y < 192; $y++) {
  $px[$y] = array();
}

/* Space to store colors: */

$colors = array();

/* Load the image */
for ($y = 0; $y < 192; $y++) {
  for ($x = 0; $x < 80; $x++) {
    $r = ord(fgetc(STDIN));
    $g = ord(fgetc(STDIN));
    $b = ord(fgetc(STDIN));

    $c = sprintf("%02x%02x%02x", $r, $g, $b);

    $px[$y][$x] = $c;
  }
}

/* Determine the Atari colors utilize, so we can send bytes for the four
   color palette entries */

$fi = fopen("atari256.ppm", "r");
/* Skip header; assuming:
P6
160 192
255
*/

for ($i = 0; $i < 3; $i++) {
  fgets($fi);
}

$atari_colors = array();
for ($hue = 0; $hue < 16; $hue++) {
  for ($lum = 0; $lum < 16; $lum++) {
    $r = ord(fgetc($fi));
    $g = ord(fgetc($fi));
    $b = ord(fgetc($fi));

    $c = sprintf("%02x%02x%02x", $r, $g, $b);

    $atari_colors[$c] = array($hue, $lum);
  }
}
fclose($fi);


for ($y = 0; $y < 192; $y++) {
  for ($x = 0; $x < 80; $x++) {
    $c = $px[$y][$x];
    list($hue, $lum) = $atari_colors[$c];

    /* FIXME: Write out the bytes, and interleave scanlines correctly (saving 2 x 40 x 192 bytes out) */
    printf("(%d,%d) => %02d %02d\n", $x, $y, $hue, $lum);
  }
}

