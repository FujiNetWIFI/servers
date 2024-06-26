<?php
  /* Based on analysis of all letters found in "words.txt" via
     `cat words.txt | tr -d "\n" | sed -e "s/\(.\)/\1\n/g" | sort | uniq -c` */
  $LETTERS = array(
    "A" => array("freq" =>  23, "score" =>  2),
    "B" => array("freq" =>   6, "score" =>  7),
    "C" => array("freq" =>  12, "score" =>  5),
    "D" => array("freq" =>  14, "score" =>  5),
    "E" => array("freq" =>  38, "score" =>  1),
    "F" => array("freq" =>   5, "score" => 10),
    "G" => array("freq" =>  11, "score" =>  5),
    "H" => array("freq" =>   7, "score" =>  7),
    "I" => array("freq" =>  24, "score" =>  2),
    "J" => array("freq" =>   1, "score" => 15),
    "K" => array("freq" =>   4, "score" => 10),
    "L" => array("freq" =>  17, "score" =>  5),
    "M" => array("freq" =>   8, "score" =>  7),
    "N" => array("freq" =>  20, "score" =>  5),
    "O" => array("freq" =>  18, "score" =>  5),
    "P" => array("freq" =>   9, "score" =>  7),
    "Q" => array("freq" =>   1, "score" => 15),
    "R" => array("freq" =>  22, "score" =>  2),
    "S" => array("freq" =>  30, "score" =>  2),
    "T" => array("freq" =>  20, "score" =>  5),
    "U" => array("freq" =>  11, "score" =>  5),
    "V" => array("freq" =>   3, "score" => 10),
    "W" => array("freq" =>   4, "score" => 10),
    "X" => array("freq" =>   1, "score" => 15),
    "Y" => array("freq" =>   5, "score" => 10),
    "Z" => array("freq" =>   1, "score" => 15),
  );

$LETTERS_FREQ = "";
foreach ($LETTERS as $k=>$v) {
  $LETTERS_FREQ .= str_repeat($k, $v["freq"]);
}

