# Additional improvements to the lobby server

* Create test scenarios (done)
* Add an html page with documentation and use of endpoints.
* In test scenarions set client url to TNFS:// (on hold)
* Deploy CI/CD.
* Develop a reaper process to remove extremely old and offline servers from the database.
* Provide persistance to the servers stored dumping GAMESRV to a json file every X seconds.
* Provide a http://host/version to dump current version and time of the server alive. (done)
* Add https support to the server.
* Simplify server even further using base go and removing gin framework.
* Set up fail2ban (infra)