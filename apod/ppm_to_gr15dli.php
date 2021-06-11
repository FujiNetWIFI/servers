#!/usr/bin/php
<?php
/* Skip header; assuming:
P6
160 {something}
255
*/

$DEBUG = false;

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

  if (!array_key_exists($r, $atari_colors)) {
    $atari_colors[$r] = array();
  }
  if (!array_key_exists($g, $atari_colors[$r])) {
    $atari_colors[$r][$g] = array();
  }

  $atari_colors[$r][$g][$b] = ($i * 2);
}

/* Open the output files */
$img_out = fopen($argv[2], "w");
if ($img_out == NULL) {
  fprintf(STDERR, "Error opening image output '%s'\n", $argv[2]);
  exit(1);
}

$pal_out = fopen($argv[3], "w");
if ($pal_out == NULL) {
  fprintf(STDERR, "Error opening palette output '%s'\n", $argv[3]);
  exit(1);
}

for ($y = 0; $y < 192; $y++) {
  if ($DEBUG) fprintf(STDERR, "%d: Row %d...\n", hrtime(true), $y);

  if ($DEBUG) fprintf(STDERR, "%d: Cropping...\n", hrtime(true));

  /* Grab a single scanline (row) strip o fthe image */
  $im_strip = clone $im;
  $im_strip->cropImage(160, 1, 0, $y);

  if ($DEBUG) fprintf(STDERR, "%d: Quantizing...\n", hrtime(true));

  /* Reduce it to 4 colors */
  $im_strip->quantizeImage(
    4, /* 4 colors */
    Imagick::COLORSPACE_RGB,
    0, /* tree depth (fastest) */
    false, /* dither is VERY slow :( */
    false /* measure error */
  );

  /* Map to the Atari palette */
  if ($DEBUG) fprintf(STDERR, "%d: Remapping...\n", hrtime(true));
  $im_strip->remapImage($im_pal, true /* dither */);

  /* Export the pixels */
  if ($DEBUG) fprintf(STDERR, "%d: Exporting...\n", hrtime(true));
  $pixels = $im_strip->exportImagePixels(
    0, 0, 160, 1, "RGB", Imagick::PIXEL_CHAR
  );
  $px = array();

  /* Process the row */ 
  $palette = array();
  $idx = 0;

  /* First gather the colors and build a palette... */
  if ($DEBUG) fprintf(STDERR, "%d: Building palette...\n", hrtime(true));
  for ($x = 0; $x < 160; $x++) {
    /* Get the pixel, and store it (as RGB) for later export
       (as a palette index) */
    $r = $pixels[$x * 3];
    $g = $pixels[$x * 3 + 1];
    $b = $pixels[$x * 3 + 2];
    $px[$x] = array($r, $g, $b);

    /* Capture the colors (as RGB) into a palette, and write to the
       palette file */
    if (!array_key_exists($r, $palette)) {
      $palette[$r] = array();
    }
    if (!array_key_exists($g, $palette[$r])) {
      $palette[$r][$g] = array();
    }
    if (!array_key_exists($b, $palette[$r][$g])) {
      if ($idx < 4) {
        $c = chr($atari_colors[$r][$g][$b]);

        if ($DEBUG) {
          fprintf(STDERR, "%d: Adding color #%d: %d,%d,%d (atari color %d)\n",
            hrtime(true), $idx, $r, $g, $b, $atari_colors[$r][$g][$b]);
        }

        $palette[$r][$g][$b] = $idx;

        fwrite($pal_out, $c, 1);

        $idx++;
      } else {
        fprintf(STDERR, "Too many colors! (%d,%d,%d)\n", $r, $g, $b);
      }
    }
  }

  /* Pad the palette, so it's always 4 bytes */
  if ($idx < 4) {
    if ($DEBUG) fprintf(STDERR, "%d: Adding %d buffer colors\n", hrtime(true), 4 - $idx);
    for ($i = $idx; $i < 4; $i++) {
      fwrite($pal_out, chr(0), 1);
    }
  }

  if ($DEBUG) fprintf(STDERR, "\n");


  /* ...Then, map all pixels in the image from RGB triplets to a single
     byte containing 4 palette index values */
  $byt = array(0, 0, 0, 0);

  for ($x = 0; $x < 160; $x += 4) {
    for ($i = 0; $i < 4; $i++) {
      $r = $px[$x + $i][0];
      $g = $px[$x + $i][1];
      $b = $px[$x + $i][2];

      $byt[$i] = $palette[$r][$g][$b];
    }

    $c = chr(($byt[0] * 64) + ($byt[1] * 16) + ($byt[2] * 4) + $byt[3]);
    fwrite($img_out, $c, 1);
  }
}

fclose($img_out);
fclose($pal_out);

