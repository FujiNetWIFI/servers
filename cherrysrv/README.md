Cherry Server
=============

Cherry Server is a minimal tcp chat server that allows clients to connect to it and chat in a single room. Some commands are supported (type /help for commands)

It's filosophy is that it should be easy to implement by low powered systems (8/16bits) so some unusual decisions were taken:

* simple, line based tcp protocol
* no SSL encription
* no Unicode
* no password 


These points may change in the future, specially regarding tcp/ssl support.

Implementing a Cherry Server client
===================================

Cherry Server uses a tcp line based protocol, so clients should store all data input until client pushes ENTER/RETURN where it will send the data.

In the same way, clients are expected to read input until EOL (\n) before processing any response.

Cherry Server will discard any character over 255 in the input line, so keep it shorter. Also, it will not reply back with any line longer than 255 characters.

If the message sent by the client starts with a slash '/' (in position 0) it will be considered a command that will trigger some specific action at server side.

Responses to commands can be multi-line (see below)

If line does not start with a '/' it's considered text to be shared with the rest of the logged clients. Cherry server responses are in the following format:

 >#channelname>@sender>text

Until Cherry Server implements multiple channels, #channelname will be #main.

#channelname and @sender will be always 16 char max (17 if you count # and @) after the simbol they will always start with a letter.

If the message sent by the client starts with '/' it will be considered a command. Commands' responses can be single line or multiline. To facilitate client processing, system responses will follow the same format:

>/command>num>text

/command will be always 16 char max (17 if you count /) and after the slash they will start with a letter.
num will be always char max and in decimal format (not binary), but you can safely asume it will be much smaller.

some examples to understand the role of num:

/who
>/who>0>@username

/users
>/users>4>@user5
>/users>3>@user4
>/users>2>@user3
>/users>1>@user2
>/users>0>@user1

Additionally, the server will send events that will be unrelated to /commands or @user chats. These events may be related to users joining or leaving the room or server, the server being shut down, etc...

Event messages will be sent to the client in the followith way:

>#channelname>!event>text

Again event will be 16 max and context specific (to be documented). These event messages can happen at any time.

Cherry Server versioning
========================

Cherry Server follow semver rules:

MAJOR version when you make incompatible API changes
MINOR version when you add functionality in a backwards compatible manner
PATCH version when you make backwards compatible bug fixes

specifically for Cherry server it means:

MAJOR version (1.x.x) implies the reponse of existing commands (as well as login functionality) changes and limits existing clients functionality.
MINOR version (x.1.x) means new commands are added that require client to be updated but does not limit use of existing functions by the clients.
PATCH version (x.x.1) are bug fixes that do not impact impact in any way clients.


