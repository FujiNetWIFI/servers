all: run


build:
	go build .

run: build
	go run .

backup:
	cd .. && tar cvzf `date "+%Y-%m-%d"`-scorched.tar.gz scorched && cd scorched

test:
	go test 

nettest:
	
	@cd wsclient && make && cd ..
	@ ./test_do_command.bats

nettests:
	@cd wsclient && make && cd ..
	@while true ; do clear; ./test_do_command.bats ;  done


help:
	@printf "make (all)		-- create local database (if needed) and run the server\n\
	 ·   build		-- build the code\n\
	 ·   run		-- start the server\n\
	 ·   clean		-- remove the database\n\
	 ·   test		-- run code tests\n\
	 ·   nettest		-- run remote user tests\n\
	 ·   nettests		-- run remote user tests (loop)\n\
	 ·   backup		-- backup all directory\n"