# Makefile for networds server
# Bill Kendrick <bill@newbreedsoftware.com>
# 2020-12-08 - 2020-12-26

all:	words.txt words.cnt games.db NETWORDS.LST

# Delete everything, including the current game state database!
veryclean:	clean
	-rm games.db

# Delete the constructed dictionary files
clean:
	-rm words.txt.new
	-rm words.txt
	-rm words.cnt.new
	-rm words.cnt
	-rm full-dict.txt.new
	-rm full-dict.txt
	-rm NETWORDS.LST

# A dictionary of only words possible in the game
words.txt:	full-dict.txt max-wordlen.txt
	grep -E "^[abcdefghijklmnopqrstuvwxyz]{3,`cat max-wordlen.txt`}\$$" full-dict.txt | tr a-z A-Z > words.txt.new
	mv words.txt.new words.txt
	chmod 644 words.txt

# Source of the full dictionary
#full-dict.txt:	/usr/share/dict/words
# These via Ubuntu packages; see http://wordlist.aspell.net/
full-dict.txt:	/usr/share/dict/american-english-huge \
		/usr/share/dict/british-english-huge \
		/usr/share/dict/canadian-english-huge
	cat $^ | sort -u > full-dict.txt.new
	mv full-dict.txt.new full-dict.txt
	chmod 644 full-dict.txt

# Count of how many words the game has
words.cnt:	words.txt
	cat words.txt | wc -l > words.cnt.new
	mv words.cnt.new words.cnt
	chmod 644 words.cnt
	@echo -n "Word count is "
	@cat words.cnt
	@echo

# A database file (start as blank) for the state of the games
games.db:
	touch games.db
	chmod 666 games.db

NETWORDS.LST:	networds-list.txt
	cat networds-list.txt | tr "\n" "\233" > NETWORDS.LST

