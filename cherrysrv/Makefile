all:  run


build:
	go build .

run: build
	./cherrysrv

backup:
	cd .. && tar cvzf `date "+%Y-%m-%d"`-cherrysrv.tar.gz cherrysrv && cd cherrysrv

clean:
	rm cherrysrv

test:
	go test 

deploy:
	git pull
	go build .
	mv cherrysrv ~/bin
	echo "cherrysrv -srvaddr rogersm-freebsd.subnet.vcn.oraclevcn.com:1512 > cherrysrv.log 2>&1"

help:
	@printf "make (all)		-- build and run the server\n\
	 ·   build		-- build the code\n\
	 ·   run		-- start the server\n\
	 ·   clean		-- remove the database\n\
	 ·   test		-- run code tests\n\
	 ·   backup		-- backup all directory\n"


