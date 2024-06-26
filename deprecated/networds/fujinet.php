<?php

$FUJINET_MSG = "";

/**
 * Display an error message and abort the script
 *
 * Echos a string in the form of "E:message"
 * (similar to normal messages; see fujinet_msg(), below)
 * and then ABORTS THE SCRIPT.
 *
 * @param string $msg
 */
function fujinet_error($msg) {
  global $FUJINET_MSG;

  $FUJINET_MSG = "E:" . $msg . "\n";
  fujinet_exit();
}

function fujinet_exit() {
  global $FUJINET_MSG;

  // header('Content-type: text/plain');
  header("Content-length: " . strlen($FUJINET_MSG));
  if (array_key_exists("interact", $_GET) && $_GET["interact"] == "yes") {
    echo str_replace("\n", "<br/>\n", $FUJINET_MSG);
  } else {
    echo str_replace("\n", chr(155), $FUJINET_MSG);
  }
  exit;
}

/**
 * Open and lock a "database" file
 *
 * @param string $filename
 * @return file pointer $fp
 */
function fujinet_open_db($filename) {
  $fp = @fopen($filename, "r+b");
  if ($fp === false) {
    fujinet_error("Can't open $filename");
  }
  if (!@flock($fp, LOCK_EX)) {
    fclose($fp);
    fujinet_error("Can't lock $filename");
  }
  return $fp;
}

/**
 * Read a database file
 *
 * @param file pointer
 * @return array containing the deserialized data from the file
 */
function fujinet_read_db($fp) {
  $data = "";
  if (!is_resource($fp)) {
    fujinet_error("Error reading DB - got a " . get_resource_type($fp));
  }

  while (!feof($fp)) {
    $data .= @fread($fp, 8192);
  }

  if ($data) {
    return @unserialize($data);
  } else {
    return array();
  }
}

/**
 * Write, close, and unlock a database file
 *
 * @param file pointer $fp
 * @param array $dbdata to be serialized and stored in the file
 */
function fujinet_write_and_close_db($fp, $dbdata) {
  $output = serialize($dbdata);
  @ftruncate($fp, 0);
  @fseek($fp, 0);
  @fwrite($fp, $output, strlen($output));

  /* Flush, unlock, and close "database" file */
  @fflush($fp);
  @flock($fp, LOCK_UN);
  @fclose($fp);
}

/**
 * Parse the incoming command from the client
 *
 * @param array of strings $valid_args accepted by the server
 * @return array containing command and argument
 */
function fujinet_get_cmd($valid_args) {
  if (array_key_exists("cmd", $_GET)) {
    $cmd = $_GET["cmd"];
  } else {
    $cmd = "";
  }
  $args = array();
  if (array_key_exists($cmd, $valid_args)) {
    foreach ($valid_args[$cmd] as $a) {
      if (array_key_exists($a, $_GET)) {
        $args[$a] = $_GET[$a];
      } else {
        $args[$a] = "";
      }
    }
  }

  return array($cmd, $args);
}

/**
 * Fujinet message output
 *
 * Echos a string in the form of "X:data1;data2;data3"
 *
 * @param string $response one letter response
 * @param string $data1 [optional]
 * @param string $data2 [optional]
 * @param string $data3 [optional]
 */
function fujinet_msg($response, $data1 = "", $data2 = "", $data3 = "") {
  global $FUJINET_MSG;

  $FUJINET_MSG .= $response . ":" . $data1;
  if ($data2 != "" || $data3 != "") {
    $FUJINET_MSG .= ";" . $data2;
  }
  if ($data3 != "") {
    $FUJINET_MSG .= ";" . $data3;
  }
  $FUJINET_MSG .= "\n";

  if (!array_key_exists("interact", $_GET) || $_GET["interact"] != "yes") {
    return;
  }


  /* Everything below here is for RESTful HTML form access to the game, for testing */

  /* Only show state on non-repeating responses */
  if ($response != "S") {
    if (array_key_exists("state", $_GET)) {
      $state = unserialize(stripslashes(base64_decode($_GET["state"])));
      echo "<hr><pre>"; print_r($state); echo "</pre></hr>";
    } else {
      $state = array();
    }
  }

  /* Show a form to move along to the next step, based on the response */
  if ($response == "?") {
    ?>
    <h2>New game</h2>
    <form action="game.php" method="get">
      <input type="hidden" name="interact" value="yes" />
      <input type="hidden" name="cmd" value="n" />
      Name: <input type="text" name="name" />
      <input type="submit" />
    </form>
    <?php
  } else if ($response == "W" || $response == "N") {
    $state["im_player1"] = true;
    ?>
    <h2>
      <?= $response == "N" ? "Still" : "" ?>
      Waiting for match
    </h2>
    <form action="game.php" method="get">
      <input type="hidden" name="interact" value="yes" />
      <input type="hidden" name="cmd" value="m" />
      <input type="hidden" name="state" value="<?= base64_encode(serialize($state)) ?>" />
      <input type="hidden" name="id" value="<?= $data1 ?>" />
      <input type="submit" />
    </form>
    <?php
  } else if ($response == "M" || $response == "R") {
    if ($response == "M") {
      $state["opponent"] = $data2;
      $state["round"] = 1;
      ?>
      <h2>
        Matched against <?= $data2 ?>
      </h2>
      <?php
    } else {
      $state["round"] = $data2;
    } ?>
    <h3>
      Round <?= $state["round"] ?><br/>
      Letters: <code><?= $data3 ?></code>
    </h3>
    <form action="game.php" method="get">
      <input type="hidden" name="interact" value="yes" />
      <input type="hidden" name="cmd" value="w" />
      <input type="hidden" name="ply" value="<?= $state["im_player1"] ? 1 : 2 ?>" />
      <input type="hidden" name="state" value="<?= base64_encode(serialize($state)) ?>" />
      <input type="hidden" name="id" value="<?= $data1 ?>" />
      <textarea cols="9" rows="10" name="words"></textarea><br/>
      <input type="submit" />
    </form>
    <?php
  } else if ($response == "X" && $data2 == 0 || $response == "Z") {
    ?>
    <h2>Get <?= $state["opponent"] ?>'s list <?= ($response == "Z" ? " (Still waiting)" : "") ?></h2>
    <form action="game.php" method="get">
      <input type="hidden" name="interact" value="yes" />
      <input type="hidden" name="cmd" value="g" />
      <input type="hidden" name="ply" value="<?= $state["im_player1"] ? 2 : 1 ?>" />
      <input type="hidden" name="state" value="<?= base64_encode(serialize($state)) ?>" />
      <input type="hidden" name="id" value="<?= $data1 ?>" />
      <input type="submit" />
    </form>
    <?php
  } else if (($response == "X" && $data2 == 1) || $response == "V") {
    ?>
    <h2>Start next round <?= ($response == "V" ? "(Still waiting)" : "") ?></h2>
    <form action="game.php" method="get">
      <input type="hidden" name="interact" value="yes" />
      <input type="hidden" name="cmd" value="p" />
      <input type="hidden" name="ply" value="<?= $state["im_player1"] ? 1 : 2 ?>" />
      <input type="hidden" name="round" value="<?= $state["round"] + 1 ?>" />
      <input type="hidden" name="state" value="<?= base64_encode(serialize($state)) ?>" />
      <input type="hidden" name="id" value="<?= $data1 ?>" />
      <input type="submit" />
    </form>
    <?php
  }
}

