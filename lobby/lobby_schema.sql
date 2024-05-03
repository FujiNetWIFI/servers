
-- SOME BASIC RULES TO CREATE TABLES:
--
-- a) try not to over use AUTOINCREMENT, especially for MASTER TABLES
-- b) Any column that ends with id is an autoincrement and primary key. But tables ending in _id, like <tablename>_id reference id of the other table.
-- c) All fields should have NOT NULL and DEFAULTS. Fields without default will have to be approved.
-- d) Fields can be only: INTEGER AUTOINCREMENT (called id), TEXT, INT/INTEGER, TINYINT (bool) and DATETIME
-- e) Fields can only have ascii characters + _
-- f) If a field is an external reference format will be: <foreigntable>_<foreignfield>_REF
-- g) Schema will not use cascades, but it will use foreign keys.
-- h) Primary keys are defined in the CREATE TABLE.
-- i) Indices have the format: IDX_<tablename>_<fieldname>
-- k) for bool (TINYINT) => '1' = true, '0' = false
-- m) datetime is of format (\d\d\d)?\d\d\d\d-\d\d\ Only exception is table login logoff that has ISO formatting. For infitite future use: 9999999-999, for infinite past use: 0000-000
-- n) field impsysloc is special: \c\c\c(\c)+-\d\d\d\d. If it applies to no locations we use NONE/0000. If it applies to many locations ANY/0000.

-- LambdaMOO ideas:
-- https://www.hayseed.net/MOO/manuals/LambdaCoreProgMan.html
-- https://tecfa.unige.ch/guides/MOO/ProgMan/ProgrammersManual_toc.html
-- https://writing.upenn.edu/~afilreis/88/moo-glossary.html

PRAGMA foreign_keys=ON;
BEGIN TRANSACTION;

CREATE TABLE GameServer (
	serverurl  TEXT NOT NULL UNIQUE PRIMARY KEY,
	game       TEXT NOT NULL,
	appkey     INT NOT NULL,
	server     TEXT NOT NULL,
	region     TEXT NOT NULL,
	status     TEXT NOT NULL, 
	maxplayers INT NOT NULL,
	curplayers INT NOT NULL, 
    lastping datetime NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE UNIQUE INDEX idx_GameServer_Serverurl ON GameServer (Serverurl ASC);


CREATE TABLE Clients (
    serverurl TEXT NOT NULL,
    client_platform TEXT NOT NULL, 
    client_url      TEXT NOT NULL, 
    FOREIGN KEY(serverurl) REFERENCES GameServer(serverurl) ON DELETE CASCADE
);
CREATE INDEX idx_Clients_Serverurl ON Clients (Serverurl ASC);


CREATE VIEW GameServerClients AS SELECT GameServer.*, Clients.client_platform as client_platform, Clients.client_url as client_url FROM GameServer JOIN Clients ON GameServer.Serverurl = Clients.Serverurl;

COMMIT;