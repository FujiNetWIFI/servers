package main

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

// serve favicon.ico
func handleFavicon(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "./assets/img/favicon.ico")
}

func Root(w http.ResponseWriter, req *http.Request) {

}

func ShowStatus(w http.ResponseWriter, req *http.Request) {
	HTTPJsonResponse(w, http.StatusOK, Map{
		"version": STRINGVER,
		"uptime":  uptime(STARTEDON)})
}

func ShowGames(w http.ResponseWriter, req *http.Request) {

	games := GAMES.AllAsMap()

	HTTPTemplateResponse(w, "{{.}}", games.NamedM("list-of-games"))
}

func GetGameState(w http.ResponseWriter, req *http.Request) {
	id_txt := chi.URLParam(req, "id")
	id := Atoi(id_txt, -1)

	game, ok := GAMES.GetAtPos(id)

	if !ok {
		HTTPJsonResponse(w, http.StatusNotFound, Map{"message": "game " + id_txt + " does not exist."})
		return
	}

	HTTPJsonResponse(w, http.StatusOK, Map{"state": game.M()})
}

func PostShoot(w http.ResponseWriter, req *http.Request) {}

func PostNewPlayer(w http.ResponseWriter, req *http.Request) {

	req.Body = http.MaxBytesReader(w, req.Body, 1024)

	var player PlayerCred

	id_txt := chi.URLParam(req, "id")
	id := Atoi(id_txt, -1)

	if err := HTTPDecodeStructFromPost(req.Body, &player); err != nil {
		HTTPJsonResponse(w, http.StatusBadRequest, Map{"message": "VALIDATEERR - Invalid Json",
			"errors": []string{err.Error()}})
		return
	}

	game, ok := GAMES.GetAtPos(id)

	if !ok {
		HTTPJsonResponse(w, http.StatusBadRequest, Map{"message": "GAMEERR - Game does not exist",
			"errors": []string{"Game " + id_txt + " does not exist."}})
		return
	}

	game.Add(makePlayer(player.Name, req.RemoteAddr, 1)) // TODO: change 1 to something that makes sense

	HTTPJsonResponse(w, http.StatusOK, Map{"message": "Player added to game " + id_txt})

}

func PostLeavePlayer(w http.ResponseWriter, req *http.Request) {

	var player PlayerCred

	id_txt := chi.URLParam(req, "id")
	id := Atoi(id_txt, -1)

	if err := HTTPDecodeStructFromPost(req.Body, &player); err != nil {
		HTTPJsonResponse(w, http.StatusBadRequest, Map{"message": "VALIDATEERR - Invalid Json",
			"errors": []string{err.Error()}})
		return
	}

	game, ok := GAMES.GetAtPos(id)

	if !ok {
		HTTPJsonResponse(w, http.StatusBadRequest, Map{"message": "GAMEERR - Game does not exist",
			"errors": []string{"Game " + id_txt + " does not exist."}})
	}

	game.Remove(player.Name)

	HTTPJsonResponse(w, http.StatusOK, Map{"message": "Player removed from game " + id_txt})
}
