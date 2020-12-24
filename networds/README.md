# Networds - A networked word game for FujiNet

## Objective

Given a list of letters, enter as many words as you can within the
alotted time.

Point your game client at this server and it will connect you with
an opponent, once one is available. The server handles scoring (and
validation of words).

## Configuration

- The source dictionary/ies can be specified in the `Makefile`.
- The max. number of letters for each word can be specified in
  the file `max-wordlen.txt`. It is used to generate the subset
  dictionary during the `make` process, and during runtime.
- Which letters are vowels, and how frequently they must appear
  within the word list, can be set at the top of `game.php`.
- The frequency (probability of appearing) of each letter,
  and the score received for successfully using the letter,
  is set in `letters.php`.

## Credits
- Server code cobbled together by Bill Kendrick, 2020</li>
- Word list from [SCOWL (And Friends)](http://wordlist.aspell.net/)

