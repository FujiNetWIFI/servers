<?php include("letters.php"); ?>
<html><head><title>Networds - A networked word game for FujiNet</title></head>
<body>

<h1 align=center>
  Networds<br/>
  A networked word game for FujiNet
</h1>

<p>
  Given a list of letters, enter as many words as you can
  within the alotted time.
</p>

<p>
  Point your game client at this server and it will connect you
  with an opponent, once one is available.  The server handles
  scoring (and validation of words).
</p>

<p>
  The game uses a subset of an English dictionary of words,
  containing all 3- to <?php echo trim(file_get_contents("max-wordlen.txt")); ?>-letter
  words that contain no punctuation or accented characters (only A-Z).
  That's <?php echo number_format(file_get_contents("words.cnt")); ?> words!
</p>

<h2>Scoring:</h2>
<?php
$scores = array();
foreach ($LETTERS as $l=>$v) {
  if (!array_key_exists($v['score'], $scores)) {
    $scores[$v['score']] = "";
  }
  $scores[$v['score']] .= $l . ", ";
}
ksort($scores);
?>

<ul>
  <?php
  foreach ($scores as $pt=>$ltr) {
    $ltr = rtrim($ltr, " ,");
    ?>
    <li>
      <code><?= $ltr ?></code>
      &mdash;
      <?= $pt ?> point<?= ($pt != 1) ? "s" : "" ?>
    </li>
    <?php
  } ?>
</ul>

<h2>Credits</h2>
<ul>
  <li>Server code cobbled together by Bill Kendrick, 2020</li>
  <li>Word list from <a href="http://wordlist.aspell.net/">SCOWL (And Friends)</a></li>
</ul>

</body></html>

