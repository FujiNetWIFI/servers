<?php
include("fujinet.php");
include("letters.php");

/* Note: Make sure "words.txt" contains... */
/* ...words no shorter than this... */
$MIN_LETTERS = 3;
/* ...And words up to this long */
$MAX_LETTERS = intval(file_get_contents("max-wordlen.txt"));
/* (See Makefile) */
/* They must contain this many vowels */
$VOWELS = "AEIOU";
$MIN_VOWELS = 2;
$MAX_VOWELS = 4;

/**
 * Load word dictionary
 *
 * @return array of words
 */
function load_dict() {
  /* Load dictionary */
  $dict_contents = @file_get_contents("words.txt");
  if ($dict_contents === false) {
    fujinet_error("Can't open dictionary");
  }
  $dict = @explode("\n", $dict_contents);
  if (count($dict) == 0) {
    fujinet_error("Can't parse dictionary");
  }
  return $dict;
}

/**
 * Pick some random letters for a new round
 *
 * @return string of letters
 */
function random_letters() {
  global $MAX_LETTERS, $VOWELS, $MIN_VOWELS, $MAX_VOWELS, $LETTERS_FREQ;

  do {
    $letters = "";
    for ($i = 0; $i < $MAX_LETTERS; $i++) {
      $letters .= substr($LETTERS_FREQ, rand() % strlen($LETTERS_FREQ), 1);
    }
  
    /* Make sure it contains some vowels */
    $done_vow = false;
    do {
      preg_match_all("/[$VOWELS]/", $letters, $matches);
      if (count($matches) >= 1 && count($matches[0]) >= $MIN_VOWELS) {
        $done_vow = true;
      } else {
        $n = rand() % strlen($VOWELS);
        $l = rand() % $MAX_LETTERS;
        $new_letters = substr($letters, 0, $l);
        $new_letters .= substr($VOWELS, $n, 1);
        $new_letters .= substr($letters, $l + 1);
        $letters = $new_letters;
      }
    } while (!$done_vow);

    /* Make sure it doesn't contain too MANY vowels, though */
    preg_match_all("/[$VOWELS]/", $letters, $matches);
    if (count($matches) >= 1 && count($matches[0]) > $MAX_VOWELS) {
      $done_cons = false;
    } else {
      $done_cons = true;
    }
  } while (!$done_cons);

  return $letters;
}

/**
 * Start a new game, or queue up for one.
 *
 * Responses:
 *
 *  - W:{gameid}
 *    New game created with id {gameid}; waiting for opponent;
 *    check back (poll) with "M" command to see if you have one.
 *    Client assumes role of player 1.
 *
 *  - M:{gameid};{name};{letters}
 *    Starting game id {gameid} with opponent {name}, round 1's letters are {letters}
 *    Client assumes role of player 2.
 *
 * @param string $name of new player
 */
function new_game($name) {
  global $games;

  /* Sanitize username & make sure we have one */
  $name = preg_replace("/[^a-zA-Z0-9 ]/", "", $name);
  if ($name == "") {
    fujinet_error("Bad username");
  }

  $match = -1;
  for ($i = 0; $i < count($games) && $match == -1; $i++) {
    if ($games[$i]["player2"] == "") {
      $match = $i;
    }
  }

  if ($match == -1) {
    /* No one to play with yet; create a new game */

    $gameid = time() . "-" . getmypid() . "-" . rand();
    $games[] = array(
      "id" => $gameid,
      "player1" => $name,
      "player2" => "",
      "p1ip" => $_SERVER['REMOTE_ADDR'],
      "ts" => time(),
    );
    fujinet_msg("W", $gameid);
  } else {
    /* Found a match! Start the game! */

    $gameid = $games[$match]["id"];
    $games[$match]["player2"] = $name;
    $games[$match]["p2ip"] = $_SERVER['REMOTE_ADDR'];
    $games[$match]["letters"] = random_letters();
    $games[$match]["round"] = 1;
    $games[$match]["p1score"] = 0;
    $games[$match]["p2score"] = 0;
    $games[$match]["ts"] = time();

    fujinet_msg("M", $gameid, $games[$match]["player1"], $games[$match]["letters"]);
  }
}

/**
 * Get a game (its index within $games) by its game id.
 *
 * @param string $gameid
 * @return int (-1 if game is not found)
 */
function get_game_by_id($gameid) {
  global $games;

  /* Find the game, by it's game id */
  $match = -1;
  for ($i = 0; $i < count($games) && $match == -1; $i++) {
    if ($games[$i]["id"] == $gameid) {
      $match = $i;
    }
  }
  return $match;
}

/**
 * Start a new round
 *
 * @param int $gameid
 * @param int $ply
 * @param int $round that the player is ready for
 *
 * Responses:
 *
 *  - 0:{gameid}
 *    Game with {gameid} doesn't exist or has expired
 *    (you'll need to start a new one with "M")
 *
 *  - X:
 *
 * @param string $gameid
 */
function next_round($gameid, $ply, $round) {
  global $games;

  $match = get_game_by_id($gameid);

  if ($match == -1) {
    /* That game doesn't exist [any more]! */
    fujinet_msg("0", $gameid);
  } else {
    if ($games[$match]["round"] < $round) {
      /* Denote that we're ready */
      $games[$match]["p" . $ply . "ready"] = true;

      if ($games[$match]["p" . (2 - $ply) . "ready"] == false) {
        /* Other player is not ready yet */
        fujinet_msg("V", $gameid, $games[$match]["round"]);
      } else {
        $games[$match]["round"]++;
        $games[$match]["letters"] = random_letters();
        $games[$match]["ts"] = time();
        unset($games[$match]["words1"]);
        unset($games[$match]["words2"]);
        unset($games[$match]["p1ready"]);
        unset($games[$match]["p2ready"]);

        fujinet_msg("R", $gameid, $games[$match]["round"], $games[$match]["letters"]);
      }
    } else {
      /* The new round has been initated; send the letters */
 
      fujinet_msg("R", $gameid, $games[$match]["round"], $games[$match]["letters"]);
    }
  }
}


/**
 * Has an opponent joined my game?
 *
 * Responses:
 *
 *  - 0:{gameid}
 *    Game with {gameid} doesn't exist or has expired
 *    (you'll need to start a new one with "M")
 *
 *  - M:{gameid};{name};{letters}
 *    An opponent arrived, and game id {gameid} with opponent {name}
 *    has started; round 1's letters are {letters}.
 *    Player should enter words, and send them with "W".
 *
 *  - N:{gameid}
 *    No opponent has arrived for game id {gameid};
 *    continue to check back (poll) with "M" command to see if you have one.
 *
 * @param string $gameid
 */
function game_matched($gameid) {
  global $games;

  $match = get_game_by_id($gameid);

  if ($match == -1) {
    /* That game doesn't exist [any more]! */
    fujinet_msg("0", $gameid);
  } else {
    /* Do we have an opponent? */
    if ($games[$match]["player2"] != "") {
      fujinet_msg("M", $gameid, $games[$match]["player2"], $games[$match]["letters"]);
    } else {
      fujinet_msg("N", $gameid);
    }
    $games[$match]["ts"] = time();
  }
}

/**
 * Store a player's words, and calculate score for each while we do it.
 *
 * @param string $gameid
 * @param int $ply (1 or 2)
 * @param array $words
 */
function store_and_score_words($gameid, $ply, $words) {
  global $games, $MIN_LETTERS, $MAX_LETTERS, $LETTERS;

  $match = get_game_by_id($gameid);
  if ($match == -1) {
    fujinet_error("Game $gameid is gone!?");
  }

  $dict = load_dict();

  $tot_score = 0;

  $words_seen = array();
  $word_scores = array();
  foreach ($words as $word) {
    /* For each word, score it */
    $word = trim($word);
    if (strlen($word) > 0) {
      if (strlen($word) >= $MIN_LETTERS &&
          strlen($word) <= $MAX_LETTERS
      ) {
        if (in_array($word, $words_seen)) {
          $score = 0; /* Already used this word in this round */
          $prob = "repeat";
        } else {
          $words_seen[] = $word;
  
          /* Make sure it's made of the letters in this game */
          $available_letters = array();
          for ($i = 0; $i < $MAX_LETTERS; $i++) {
            $available_letters[] = substr($games[$match]["letters"], $i, 1);
          }
   
          $only_valid = true;
          for ($i = 0; $i < strlen($word); $i++) {
            $l = substr($word, $i, 1);
            $found = -1;
            for ($j = 0; $j < $MAX_LETTERS && $found == -1; $j++) {
              if (array_key_exists($j, $available_letters) && $available_letters[$j] == $l) {
                $found = $j;
              }
            }
            if ($found != -1) {
              unset($available_letters[$found]);
            } else {
              $only_valid = false;
            }
          }
    
          if (!$only_valid) {
            $score = 0; /* Invalid letter provided */
            $prob = "invalid";
          } else if (!in_array($word, $dict)) {
            $score = 0; /* Not in the dictionary! */
            $prob = "nonword";
          } else {
            $score = 0;
            for ($i = 0; $i < strlen($word); $i++) {
              $l = substr($word, $i, 1);
              $pt = $LETTERS[$l]['score'];
              $score = $score + $pt;
            }
            $tot_score += $score;
            $prob = "";
          }
        }
      } else {
        $score = 0;
        $prob = "length";
      }
  
      $word_scores[] = array(
        "word" => $word,
        "score" => $score,
        "prob" => $prob
      );
    }
  }

  $games[$match]["words" . $ply] = $word_scores;
  $games[$match]["p" . $ply . "score"] += $tot_score;
  $games[$match]["ts"] = time();
}


/**
 * Retreive the words, and their scores, for the current round by a player
 *
 * @param string $gameid
 * @param int $ply (1 or 2)
 * @param int $step (0 = getting my score, 1 = getting opponent's score)
 *
 * Responses:
 *
 *  - Z:{gameid};{ply}
 *    Player {ply} hasn't submitted words for the current round of game id {gameid} yet.
 *
 *  - S:{word};{score};{problems} [repeats]
 *    Each word, and its score, followed by the next message, once.
 *
 *  - X:{step}
 *    Finishes this list of score (reminds client what step they were on)
 */
function show_word_scores($gameid, $ply, $step) {
  global $games;

  $match = get_game_by_id($gameid);
  if ($match == -1) {
    fujinet_error("Game $gameid is gone!?");
  }

  if (!array_key_exists("words" . $ply, $games[$match])) {
    /* That player hasn't submitted words for this round yet! */
    fujinet_msg("Z", $gameid);
  } else {
    $words = $games[$match]["words" . $ply];
    foreach ($words as $w) {
      fujinet_msg("S", $w["word"], $w["score"], $w["prob"]);
    }
    fujinet_msg("X", $gameid, $step);
  }
  $games[$match]["ts"] = time();
}


/* --- Main --- */

/* Open DB */
$db = fujinet_open_db("games.db");
$games = fujinet_read_db($db);

/* Debug option to dump game list */
if ($_GET["debug"] == "y") {
  echo "<pre>"; print_r($games); echo "</pre>";
  echo "Current ts = " . time();
  @flock($db, LOCK_UN);
  @fclose($db);
  exit;
}

/* Get and parse command */
list($cmd, $args) = fujinet_get_cmd(
  array(
    /* Starting a game */
    "n" => array("name"),
    "m" => array("id"),

    /* Playing a game */
    "w" => array("id", "ply", "words"),
    "g" => array("id", "ply"),
    "p" => array("id", "ply", "round"),
  )
);

if ($cmd == "n") {
  /* Start a new game (or wait for one) */
  new_game($args["name"]);
} else if ($cmd == "m") {
  /* Waiting for a match to start; do I have an opponent? */
  game_matched($args["id"]);
} else if ($cmd == "w") {
  /* Submitting my words, score them, and show me my results */
  $my_words = $args["words"];
  $my_words = preg_replace("/\s/", ",", $my_words);
  $my_words = preg_replace("/[^a-zA-Z0-9,]/", "", $my_words);

  store_and_score_words($args["id"], $args["ply"], explode(",", $my_words));
  show_word_scores($args["id"], $args["ply"] /* should be me */, 0);
} else if ($cmd == "g") {
  /* Show me my opponent's words and resulting scores */
  show_word_scores($args["id"], $args["ply"] /* should be my opponent */, 1);
} else if ($cmd == "p") {
  /* I'm ready to proceed to the next round */
  next_round($args["id"], $args["ply"], $args["round"]);
} else {
  /* Unknown command; dunno how to respond! */
  fujinet_msg("?");
}


/* Sweep away older games */

$remaining_games = array();
if (count($games)) {
  foreach ($games as $g) {
    if (time() - $g["ts"] <= 60 * 5) {
      $remaining_games[] = $g;
    }
  }
}

/* Close DB */
fujinet_write_and_close_db($db, $remaining_games);

