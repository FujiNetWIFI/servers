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

   Other more complicated modes:

    * ?mode=rgb9 -- fetch 80x192 GRAPHICS 9 4096 color (R, G, B split)

   Sample options:

    * ?sample=N -- fetch a sample image, rather than APOD (where N is 1 or higher)
*/

$sample_files = array(
  "alt_reality.jpg",
  "ngc2818.jpg",
  "Parrot.jpg",
  "SPACE.JPG",
);

$today = date("Y-m-d");

if (array_key_exists("sample", $_GET) &&
    intval($_GET["sample"]) &&
    intval($_GET["sample"]) <= count($sample_files)
) {
  $sample = intval($_GET["sample"]);
  $basename = "SAMPLE" . $sample;
} else {
  $sample = false;
  $basename = "AP" . date("ymd");
}

/* What mode of image do they want? */
if (array_key_exists("mode", $_GET)) {
  $mode = trim($_GET["mode"]);
} else {
  $mode = "";
}

if ($mode == "8") {
  $img_size = 7680;
  $outfile = "img/$basename.GR8";
} else if ($mode == "15") {
  $img_size = 7680;
  $outfile = "img/$basename.G15";
} else if ($mode == "rgb9") {
  $img_size = 7680 * 3;
  $outfile = "img/$basename.CV9";
} else {
  $img_size = 7680;
  $mode = "9";
  $outfile = "img/$basename.GR9";
}


if (!$sample) {
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

      if ($img_src != "") {
        /* Found an image! Convert it! */
        system("./fetch_and_cvt.sh 'https://apod.nasa.gov/apod/$img_src' '$mode' '$outfile'");
      } else {
        /* Let's see if there's a YouTube video */
        $vid_src = "";

        $iframes = $dom->getElementsByTagName('iframe');
        foreach ($iframes as $iframe) {
          if ($vid_src == "") {
            if (stripos($iframe->getAttribute('src'), "https://www.youtube.com/embed/") !== false) {
              $vid_src = $iframe->getAttribute('src');
            }
          }
        }

        if ($vid_src != "") {
          /* Found a video! Fetch and convert its thumbnail! */
          $vid_url_path = parse_url($vid_src, PHP_URL_PATH);
          $vid_parts = explode("/", $vid_url_path);
          $vid_id = $vid_parts[2];

          if ($vid_id) {
            system("./fetch_and_cvt.sh 'https://img.youtube.com/vi/$vid_id/maxresdefault.jpg' '$mode' '$outfile'");
          }
        }
      }
    }


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
} else {
  /* This is kinda dumb, but `wget` can't fetch via `file` scheme */
  $sample_img = "http://billsgames.com/fujinet/apod/samples/" . $sample_files[$sample - 1];
  system("./fetch_and_cvt.sh '$sample_img' '$mode' '$outfile'");
}

/* Get the image */
$img = file_get_contents($outfile);
$descr = file_get_contents("descr.txt");


/* Dump the results: */
header("Content-Type: application/octet-stream");
header("Content-Length: " . ($img_size + strlen($descr)));
header("Content-Disposition: attachment; filename=\"" . basename($outfile) . "\"");

echo $img;
echo $descr;
