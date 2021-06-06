#!/usr/bin/php
<?php
/* Skip header; assuming:
P6
160 {something}
255
*/

$DEBUG = true;

if ($argc != 4) {
  fprintf(STDERR, "Usage: %s input_ppm_image output_atari_image output_atari_palette\n", $argv[0]);
  exit(1);
}


/* Load the image */
$im = new Imagick();

if (!$im->readImage($argv[1])) {
  fprintf(STDERR, "Error opening input '%s'\n", $argv[1]);
  exit(1);
}

/* Load our palette */
$im_pal = new Imagick();
if (!$im_pal->readImage("atari128.ppm")) {
  fprintf(STDERR, "Error opening color palette map file '%s'\n", "atari128.ppm");
  exit(1);
}

$pal_px = $im_pal->exportImagePixels(0, 0, 8, 16, "RGB", Imagick::PIXEL_CHAR);

$atari_colors = array();
for ($i = 0; $i < 128; $i++) {
  $r = $pal_px[$i * 3];
  $g = $pal_px[$i * 3 + 1];
  $b = $pal_px[$i * 3 + 2];

  $c = sprintf("%02x%02x%02x", $r, $g, $b);
  $atari_colors[$c] = ($i * 2);
}

/* Open the output files */
$img_out = fopen($argv[2], "w");
if ($img_out == NULL) {
  fprintf(STDERR, "Error opening palette output '%s'\n", $argv[2]);
  exit(1);
}

$pal_out = fopen($argv[3], "w");
if ($pal_out == NULL) {
  fprintf(STDERR, "Error opening palette output '%s'\n", $argv[3]);
  exit(1);
}

for ($y = 0; $y < 192; $y++) {

  if ($DEBUG) fprintf(STDERR, "Row %d...\n", $y);

#  | convert - -depth 8 +dither -colors 4 pnm:- \
#  | convert - +dither -remap atari128.ppm -interpolate nearest pnm:- \

  /* Grab a single scanline (row) strip o fthe image */
  $im_strip = clone $im;
  $im_strip->cropImage(160, 1, 0, $y);

  /* Reduce it to 4 colors */
  $im_strip->quantizeImage(
    4, /* 4 colors */
    Imagick::COLORSPACE_RGB,
    0, /* tree depth (fastest) */
    true, /* dither */
    false /* measure error */
  );

  /* Map to the Atari palette */
  $im_strip->remapImage($im_pal, true /* dither */);

  $pixels = $im_strip->exportImagePixels(0, 0, 160, 1, "RGB", Imagick::PIXEL_CHAR);

  $px = array();
  $colors = array();

  for ($x = 0; $x < 160; $x++) {
    $r = $pixels[$x * 3];
    $g = $pixels[$x * 3 + 1];
    $b = $pixels[$x * 3 + 2];
    $px[$x] = array($r, $g, $b);

    $c = sprintf("%02x%02x%02x", $r, $g, $b);

    if (!in_array($c, $colors)) {
      if ($DEBUG) fprintf(STDERR, "Adding color $c\n");
      $colors[] = $c;
    }
  }

  if ($DEBUG) fprintf(STDERR, "\n");

  $palette = array();
  $idx = 0;
  foreach ($colors as $c) {
    $palette[$c] = $idx++;
  }

  $b = array(0, 0, 0, 0);

  for ($x = 0; $x < 160; $x += 4) {
    for ($i = 0; $i < 4; $i++) {
      $color = sprintf("%02x%02x%02x",
        $px[$x + $i][0],
        $px[$x + $i][1],
        $px[$x + $i][2]
      );

      $b[$i] = $palette[$color];
    }

    $c = chr(($b[0] * 64) + ($b[1] * 16) + ($b[2] * 4) + $b[3]);
    fwrite($img_out, $c, 1);
  }

  /* Determine the Atari colors utilize, so we can send bytes for the four
     color palette entries */

  $colors = 0;
  foreach ($palette as $rgb => $_) {
    if (!array_key_exists($rgb, $atari_colors)) {
      if ($DEBUG) fprintf(STDERR, "color $rgb doesn't exist\n");
      fwrite($pal_out, chr(0), 1);
    } else {
      $c = chr($atari_colors[$rgb]);
      if ($DEBUG) fprintf(STDERR, "Atari color $rgb = %s\n", $atari_colors[$rgb]);
      fwrite($pal_out, $c, 1);
    }
    $colors++;
  }

  if ($colors < 4) {
    if ($DEBUG) fprintf(STDERR, "Adding %d buffer colors\n", 4 - $colors);
    for ($i = $colors; $i < 4; $i++) {
      fwrite($pal_out, chr(0), 1);
    }
  } else if ($colors > 4) {
    fprintf(STDERR, "Woah, %d colors!\n", $colors);
    exit(1);
  }
}

fclose($img_out);
fclose($pal_out);

