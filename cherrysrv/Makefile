all:  run


build:
	go build .

run: 
	go run .

backup:
	cd .. && tar cvzf `date "+%Y-%m-%d"`-cherrysrv.tar.gz cherrysrv && cd cherrysrv

clean:
	rm cherrysrv

test:
	go test 


help:
	@printf "make (all)		-- build and run the server\n\
	 ·   build		-- build the code\n\
	 ·   run		-- start the server\n\
	 ·   clean		-- remove the database\n\
	 ·   test		-- run code tests\n\
	 ·   backup		-- backup all directory\n"


