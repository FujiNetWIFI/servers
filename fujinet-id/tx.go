package main

func txSavePubKeyAndToken(pubkey string, token string) error {
	queryInsert := `--sql
	INSERT INTO PubKey (pubkey, token)
	VALUES ($1, $2) `

	_, err := DATABASE.Exec(queryInsert, pubkey, token)

	if err != nil {
		DB.Printf("%s error insert Pubkey: (%s)", extendedFnName(), err)

		return err
	}

	return nil
}

func txGetByPubKey(pubkey string) (output PubKeyRecord) {

	err := DATABASE.Get(&output, "SELECT * FROM PubKey WHERE pubkey=$1;", pubkey)

	if err != nil {
		DB.Printf("%s error: %s", extendedFnName(), err)
		return output
	}

	return output
}

func txGetByToken(token string) (output PubKeyRecord) {

	err := DATABASE.Get(&output, "SELECT * FROM PubKey WHERE token=$1;", token)

	if err != nil {
		DB.Printf("%s error: %s", extendedFnName(), err)
		return output
	}

	return output
}
