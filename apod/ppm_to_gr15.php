#!/usr/bin/php
<?php
/* Skip header; assuming:
P6
160 {something}
255
*/

$DEBUG = true;

$height = 192;

if ($argc == 2) {
  $height = 1;
  $pal_out = fopen($argv[1], "a");
} else {
  $pal_out = NULL;
}

for ($i = 0; $i < 3; $i++) {
  fgets(STDIN);
}

/* Generate array to store image, so we can go over it a few times */
$px = array();
for ($y = 0; $y < $height; $y++) {
  $px[$y] = array();
  for ($x = 0; $x < 160; $x++) {
    $px[$y][$x] = array(0, 0, 0);
  }
}

/* Space to store colors: */

$colors = array();

/* Load the image */
for ($y = 0; $y < $height; $y++) {
  for ($x = 0; $x < 160; $x++) {
    $r = ord(fgetc(STDIN));
    $g = ord(fgetc(STDIN));
    $b = ord(fgetc(STDIN));
    $px[$y][$x][0] = $r;
    $px[$y][$x][1] = $g;
    $px[$y][$x][2] = $b;

    $c = sprintf("%02x%02x%02x", clamp($r), clamp($g), clamp($b));

    if (!in_array($c, $colors)) {
      if ($DEBUG) fprintf(STDERR, "Adding color $c\n");
      $colors[] = $c;
    }
  }
}

$palette = array();
$idx = 0;
foreach ($colors as $c) {
  $palette[$c] = $idx++;
}

$b = array(0, 0, 0, 0);

for ($y = 0; $y < $height; $y++) {
  for ($x = 0; $x < 160; $x += 4) {
    for ($i = 0; $i < 4; $i++) {
      $color = sprintf("%02x%02x%02x",
        clamp($px[$y][$x + $i][0]),
        clamp($px[$y][$x + $i][1]),
        clamp($px[$y][$x + $i][2])
      );

      $b[$i] = $palette[$color];
    }

    $c = chr(($b[0] * 64) + ($b[1] * 16) + ($b[2] * 4) + $b[3]);
    fwrite(STDOUT, $c, 1);
  }
}

/* Determine the Atari colors utilize, so we can send bytes for the four
   color palette entries */

$fi = fopen("atari128.ppm", "r");
/* Skip header; assuming:
P6
8 16
255
*/

for ($i = 0; $i < 3; $i++) {
  fgets($fi);
}

$atari_colors = array();
for ($i = 0; $i < 128; $i++) {
  $r = ord(fgetc($fi));
  $g = ord(fgetc($fi));
  $b = ord(fgetc($fi));

  $c = sprintf("%02x%02x%02x", clamp($r), clamp($g), clamp($b));
  $atari_colors[$c] = ($i * 2);
}
fclose($fi);

$colors = 0;
foreach ($palette as $rgb => $_) {
  if (!array_key_exists($rgb, $atari_colors)) {
    if ($DEBUG) fprintf(STDERR, "color $rgb doesn't exist\n");
    if ($pal_out) {
      fwrite($pal_out, chr(0), 1);
    } else {
      fwrite(STDOUT, chr(0), 1);
    }
  } else {
    $c = chr($atari_colors[$rgb]);
    if ($DEBUG) fprintf(STDERR, "Atari color $rgb = %s\n", $atari_colors[$rgb]);
    if ($pal_out) {
      fwrite($pal_out, $c, 1);
    } else {
      fwrite(STDOUT, $c, 1);
    }
  }
  $colors++;
}

if ($colors < 4) {
  if ($DEBUG) fprintf(STDERR, "Adding %d buffer colors\n", 4 - $colors);
  for ($i = $colors; $i < 4; $i++) {
    if ($pal_out) {
      fwrite($pal_out, chr(0), 1);
    } else {
      fwrite(STDOUT, chr(0), 1);
    }
  }
} else if ($colors > 4) {
  fprintf(STDERR, "Woah, %d colors!\n", $colors);
}

if ($pal_out) {
  fclose($pal_out);
}

/* FIXME: This function shouldn't be necessary...? */
function clamp($x) {
  return($x);
 // return (floor($x / 16) * 16);
}

