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


for ($scrn = 0; $scrn < 2; $scrn++) {
  for ($y = 0; $y < 192; $y++) {
    for ($x = 0; $x < 80; $x+=2) {
      /* Fetch two pixels from the image */
      $c1 = $px[$y][$x];
      $c2 = $px[$y][$x + 1];

      /* Get their Atari hue & luminence values */
      list($hue1, $lum1) = $atari_colors[$c1];
      list($hue2, $lum2) = $atari_colors[$c2];

      /* Save out the appropriate, interleaved image */
      if (($y + $scrn) % 2 == 0) {
        $byt = ($lum1 << 4) + $lum2;
      } else {
        $byt = ($hue1 << 4) + $hue2;
      }

      $chr = chr($byt);
      fwrite(STDOUT, $chr, 1);
    }
  }
}

