<?php
/* Astronomy Picture of the Day server app for
   Ataris with #FujiNet devices.

   Have the Atari read (e.g., "OPEN #1,4,0,..." in BASIC)
   from this script (N:HTTP://server/path/to/index.php)
   with the following GET arguments:

    * ?mode=9 -- fetch 80x192 GRAPHICS 9 16 greyscale image
    * ?mode=15 -- fetch 160x192 GRAPHICS 15 4 greyscale image
    * ?mode=8 -- fetch 320x192 GRAPHICS 8 black & white image

   Read the 7,680 bytes (40 x 192, aka 30 pages) into screen
   memory.  You can then read until an end-of-line or the
   end-of-file to grab the title and description of the image
   (e.g., "INPUT #1,A$").
*/

$today = date("Y-m-d");

$basename = "AP" . date("ymd");

/* What mode of image do they want? */
$mode = trim($_GET["mode"]);

if ($mode == "8") {
  $outfile = "img/$basename.GR8";
} else if ($mode == "15") {
  $outfile = "img/$basename.G15";
} else {
  $mode = "9";
  $outfile = "img/$basename.GR9";
}


/* Check whether it's a new day, and we'll need
   to fetch and convert an the image */
if (file_exists($outfile)) {
  $ts = date("Y-m-d", filemtime($outfile));
} else {
  $ts = "2020-01-01";
}


if ($ts < $today) {
  /* Time to fetch a new one */
  $img_src = "";
  $page = file_get_contents("https://apod.nasa.gov/apod/astropix.html");

  if (!empty($page)) {
    $dom = new DOMDocument;
    if ($dom->loadHTML($page)) {
      $imgs = $dom->getElementsByTagName('img'); 
      foreach ($imgs as $img) {
        if ($img_src == "") {
          $img_src = $img->getAttribute('src');
        }
      }
    }
  }

  system("./fetch_and_cvt.sh 'https://apod.nasa.gov/apod/$img_src' '$mode' '$outfile'");

  $rss = file_get_contents("https://apod.nasa.gov/apod.rss");
  if (!empty($rss)) {
    $dom = new DOMDocument;
    if ($dom->loadXML($rss)) {
      $items = $dom->getElementsByTagName('item');
      if ($items) {
        $latest = $items->item(0);

        if ($latest->childNodes) {
          foreach ($latest->childNodes as $child) {
            if ($child->tagName == "title") {
              $title = trim(preg_replace("/\s+/", " ", strip_tags($child->textContent)));
            }
            if ($child->tagName == "description") {
              $descr = trim(preg_replace("/\s+/", " ", strip_tags($child->textContent)));
            }
          }
        }
      }
    }

    $fo = fopen("descr.txt", "w");

    /* Store it, word-wrapping the title to avoid words
       breaking at the end of a line, but then pad each
       line to 40 characters, so we only need to INPUT
       one string (max 159 characters on the Atari end,
       to avoid scrolling any text off the 4-line text window) */
    $title = wordwrap($title, 40);
    $title_lines = explode("\n", $title);
    
    $title = "";
    foreach ($title_lines as $t) {
      $title .= $t;
      $pad = strlen($t) % 40;
      if ($pad != 0) {
        $title = $title . str_repeat("_", 40 - $pad);
      }
    }

    fprintf($fo, "%s", $title);
    fprintf($fo, "%s%c", $descr, 155); /* 155 = Atari EOL character */
    fclose($fo);
  }
}

/* Get the image */
$img = file_get_contents($outfile);
$descr = file_get_contents("descr.txt");


/* Dump the results: */
header("Content-Type: application/octet-stream");
header("Content-Length: " . 7680 + strlen($descr));
header("Content-Disposition: attachment; filename=\"" . basename($outfile) . "\"");

echo $img;
echo $descr;

