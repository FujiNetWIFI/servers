all: install-db run


build:
	go build .

install-db: db/lobby.sqlite3

db/lobby.sqlite3: lobby_schema.sql
	-@mkdir db || true 2> /dev/null
	-@rm db/lobby.sqlite3*  2> /dev/null || true
	sqlite3 -bail db/lobby.sqlite3 < lobby_schema.sql 

run:	build
#	GIN_MODE=release ./lobbyPersist
	./lobbyPersist

backup:
	cd .. && tar cvzf `date "+%Y-%m-%d"`-lobbyPersist.tar.gz lobbyPersist && cd lobbyPersist

dblist: 
	@echo "DATABASE: lobby.sqlite3"
	@echo
	@sqlite3 -column  -header db/lobby.sqlite3 $(TABLES_QUERY)
	@echo 
	@sqlite3 -column -header db/lobby.sqlite3 $(INDEX_QUERY)


dbbackup:
	@sqlite3 db/lobby.sqlite3 ".backup 'db/`date "+%Y-%m-%d"`-lobby.sqlite3'"

test:
	go test 

help:
	@printf "make (all)		-- create local database (if needed) and run the server\n\
	 ·   install-db -- set up sqlite backend\n\
	 ·   build		-- build the code\n\
	 ·   run		-- start the server\n\
	 ·   test		-- run code tests\n\
	 ·   dblist		-- generate db reports\n\
	 ·   backup		-- backup all directory\n\
	 ·   dbbackup	-- backup database in the same directory\n"



# https://www.sqlite.org/lang_analyze.html
TABLES_QUERY = "ANALYZE; \
	select DISTINCT tbl_name as TABLES, \
	CASE WHEN stat is null then 0 else cast(stat as INT) END NUMROWS \
	from sqlite_master m \
	LEFT JOIN sqlite_stat1 stat on m.tbl_name = stat.tbl \
	where m.type='table'and m.tbl_name not like 'sqlite_%' \
	order by 1; \
	drop table if exists sqlite_stat1; \
	drop table if exists sqlite_stat4;"


INDEX_QUERY = "select name as IDX, tbl_name as IN_TABLE \
	from sqlite_master where type = 'index' \
	order by name;"
