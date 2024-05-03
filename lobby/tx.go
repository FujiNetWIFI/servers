package main

// Retrieve all GameServers with its clients from the database ordered according to 'liveness'
func txGameServerGetAll() (output GameServerClientSlice, err error) {

	// output should be: online first, offline last. Inside each category, newer last ping goes first

	err = DATABASE.Select(&output, "SELECT * FROM GameServerClients ORDER BY Game, Status DESC, Curplayers DESC, Server ")

	if err != nil {
		DB.Printf("%s error: %s", extendedFnName(), err)
		return output, err
	}

	return output, nil
}

// Retrieve GameServers with its clients filtered by platform and appkey (optional) from the database ordered according to 'liveness'
// TODO: we added a simple (and cpu consuming) pagination. It would be better to use a index based, as per:
// https://www2.sqlite.org/cvstrac/wiki?p=ScrollingCursor
// but that would require to change the protocol with the client: they will have to send the latest client received
func txGameServerGetBy(platform string, appkey int, pagesize int, pagenumber int) (output GameServerClientSlice, err error) {

	// Note that SQLite LIKE operator is case-insensitive. It means "A" LIKE "a" is true.
	if appkey == -1 {
		// Sort by Game (so client can efficiently group by game name), Status (so OFFLINE stay at the bottom), Curplayers (so populated servers are at the top), and finally Server name
		err = DATABASE.Select(&output, "SELECT * FROM GameServerClients WHERE client_platform LIKE $1 ORDER BY Game, Status DESC, Curplayers DESC, Server LIMIT $2 OFFSET $3", "%"+platform+"%", pagesize, pagenumber*pagesize)
	} else {
		err = DATABASE.Select(&output, "SELECT * FROM GameServerClients WHERE client_platform LIKE $1 AND appkey=$2 ORDER BY Status DESC, Lastping DESC LIMIT $3 OFFSET $4", "%"+platform+"%", appkey, pagesize, pagenumber*pagesize)
	}

	if err != nil {
		DB.Printf("%s error: (%s)", extendedFnName(), err)
		return output, err
	}

	return output, nil
}

// Upsert new GameServer with client input
func txGameServerUpsert(gs GameServer) (err error) {

	tx, err := DATABASE.Begin()

	if err != nil {
		DB.Printf("%s error beginTx: (%s)", extendedFnName(), err)
		tx.Rollback()

		return err
	}

	queryDelete := `--sql
		DELETE FROM GameServer WHERE Serverurl = $1 -- will delete Clients with DELETE ON CASCADE
	`

	_, err = tx.Exec(queryDelete, gs.Serverurl)

	if err != nil {
		DB.Printf("%s error delete: (%s)", extendedFnName(), err)
		tx.Rollback()

		return err
	}

	queryInsert := `--sql
		INSERT INTO GameServer (Serverurl, Game, Appkey, Server, Region, Status, Maxplayers, Curplayers)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8) -- insert main server
	`

	_, err = tx.Exec(queryInsert, gs.Serverurl, gs.Game, gs.Appkey, gs.Server, gs.Region, gs.Status, gs.Maxplayers, gs.Curplayers)

	if err != nil {
		DB.Printf("%s error insert GameServer: (%s)", extendedFnName(), err)
		tx.Rollback()

		return err
	}

	queryClient := `--sql
		INSERT INTO Clients (serverurl, client_platform, client_url) VALUES ($1, $2, $3) -- insert each of the clients for the previous server
	`
	for _, client := range gs.Clients {
		_, err = tx.Exec(queryClient, gs.Serverurl, client.Platform, client.Url)

		if err != nil {
			DB.Printf("%s error insert Client: (%s)", extendedFnName(), err)
			tx.Rollback()

			return err
		}

	}

	err = tx.Commit()

	if err != nil {
		DB.Printf("%s error: (%s)", extendedFnName(), err)
		tx.Rollback()

		return err
	}

	return nil
}

// Delete a GameServers with its associated clients
func txGameServerDelete(serverurl string) (err error) {

	query := `--sql
		DELETE FROM GameServer WHERE Serverurl = $1 
	`

	_, err = DATABASE.Exec(query, serverurl)

	if err != nil {
		DB.Printf("%s error: (%s)", extendedFnName(), err)
		return err
	}

	return nil
}
