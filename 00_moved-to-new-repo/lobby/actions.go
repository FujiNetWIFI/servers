package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// send the game servers stored to the client minimised
func ShowServersMinimised(c *gin.Context) {

	form, err := parseShowServersMinimisedForm(c)

	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest,
			gin.H{
				"success": false, "message": err.Error()})

		return
	}

	ServerSliceClient, _ := txGameServerGetBy(form.Platform, form.Appkey, form.Pagesize, form.PageNumber)

	if len(ServerSliceClient) == 0 {
		c.AbortWithStatusJSON(http.StatusNotFound,
			gin.H{"success": false,
				"message": "No servers available for " + form.Platform})

		return
	}

	var ServerMinSlice []GameServerMin

	for _, server := range ServerSliceClient {
		ServerMinSlice = append(ServerMinSlice, server.Minimize())
	}

	if form.Bin == 1 {
		data := SerializeToBinaryFormat(c, ServerMinSlice, form)
		c.Data(http.StatusOK, "application/octet-stream", data)
	} else {
		c.JSON(http.StatusOK, ServerMinSlice)
	}
}

func SerializeToBinaryFormat(c *gin.Context, serverList []GameServerMin, form ShowServersMinimisedFormData) []byte {

	NextPage, _ := txGameServerGetBy(form.Platform, form.Appkey, form.Pagesize, form.PageNumber+1)

	var buf []byte
	buf = append(buf, byte(len(serverList)))
	buf = append(buf, byte(form.PageNumber))
	buf = append(buf, byte(IfElse(len(NextPage) > 0, 1, 0)))

	for _, server := range serverList {
		buf = server.appendAsBinary(buf)
	}

	return buf
}

type ShowServersMinimisedFormData struct {
	Platform   string // atari, spectrum, etc...
	Appkey     int    // -1 if none
	Pagesize   int    // number of entries to return.
	PageNumber int    // page # of the entries.
	Bin        int    // 1 if client expects binary response instead of json
}

func parseShowServersMinimisedForm(c *gin.Context) (output ShowServersMinimisedFormData, err error) {

	platform := c.Query("platform")

	if len(platform) == 0 {
		return output, fmt.Errorf("you need to submit a platform")
	}

	// optional field. If appkey is empty, it becomes a None (-1)
	appkeyForm := c.Query("appkey")
	appkey := Atoi(appkeyForm, -1)

	pagesizeForm := c.Query("pagesize")
	pagesize := 255 // big number so in case it's not in the form, the select gets all the records
	page := 0       // if it's not in the form, we defaul to start from the beginning

	// if client provides pagesize for pagination, we capture the variables
	if len(pagesizeForm) > 0 {
		pagesize = Atoi(pagesizeForm, 6)

		pageForm := c.Query("page")
		page = Atoi(pageForm, 0)
	}

	return ShowServersMinimisedFormData{
		Platform:   platform,
		Appkey:     appkey,
		Pagesize:   pagesize,
		PageNumber: page,
		Bin:        IfElse(c.Query("bin") == "1", 1, 0),
	}, nil

}

// show html view of lobby
func ShowServersHtml(c *gin.Context) {

	GameServerClient, err := txGameServerGetAll()

	if err != nil {
		message := "<tr><td colspan='10'>Unable to read servers from the database.</td></tr>"
		result := bytes.ReplaceAll(SERVERS_HTML, []byte("$$SERVERS$$"), []byte(message))
		c.Data(http.StatusOK, gin.MIMEHTML, result)

		return
	}

	ServerTemplate := `
<tr>
	<td class='server'>%s</td>
	<td class='players'>%d/%d %s </td>
</tr>
`
	GameTemplate := `
<tr>
	<td colspan='2' class='game'>%s</td>	
</tr>
`

	PlayersAvailable := "<img src='data:@file/png;base64,iVBORw0KGgoAAAANSUhEUgAAACAAAAAkCAMAAADfNcjQAAAAElBMVEUAAAD///+z9P9qfPR9hLL////Dr+VQAAAABnRSTlP//////wCzv6S/AAAACXBIWXMAAAsTAAALEwEAmpwYAAAAQElEQVQ4jWNkYsAPCMlTQQELAwMDAyMOyf/0ccOogsGjgBlXWqCjG1iYB4Eb/kMZ/9B0oPNp6AZGmApcdtPBDQA1JQVVAQAtagAAAABJRU5ErkJggg==' />"

	var servers string
	prevGame := ""

	for i, gsc := range GameServerClient {
		if gsc.Status == "online" {

			// Game type heading
			if prevGame != gsc.Game {
				servers += fmt.Sprintf(GameTemplate, html.EscapeString(gsc.Game))
				prevGame = gsc.Game
			}

			// Platform icons - commenting out for now. Not sure if really useful at this point, and it requires ongoing maint.
			// switch strings.ToLower(gsc.Client_platform) {
			// case "atari":
			// 	platformIcons += "<img src='data:@file/png;base64,iVBORw0KGgoAAAANSUhEUgAAACgAAAAgCAMAAABXc8oyAAAADFBMVEUAAAD///+z9P////83isCuAAAABHRSTlP///8AQCqp9AAAAAlwSFlzAAALEwAACxMBAJqcGAAAAGhJREFUOI3tkcEOwCAIQ8vc//9yd1CyKh7Ek0vGDSGvLVqBFmEgAANh3eTCYv2Lpy7e2hB9p7/9hTA7qRmGmnvvjhCCDQp5YnRYX10jS2TzpVVdOjNHnPFG5jrR00aeMrMe57RXKUV8AGPEFFEoV1/yAAAAAElFTkSuQmCC'/>"
			// case "apple2":
			// 	platformIcons += "<img style='transform:scale(1.1)' src='data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAABsAAAAfCAMAAAAhm0ZxAAAADFBMVEUAAAD///+z9P////83isCuAAAABHRSTlP///8AQCqp9AAAAAlwSFlzAAALEwAACxMBAJqcGAAAAHNJREFUKJHFk8EKwDAIQ1/s/v+X3WHYdltT2GmehEeCGlTjXgnoamOBqp4Mz5JhudIVWrCOOCwplgU0Wmhxn3t2sIjc7CfCyeTvIiCMDD7d8zeWQMjAnWe+85th4E03cyrwwn1+Rlj5GddgDUXfT+N9RncC1+MPVhm/JmEAAAAASUVORK5CYII='/>"
			// }

			// Add server if this is the last game client row for the server (reached the end, or next record is a different server/game)
			if i == len(GameServerClient)-1 || gsc.Server != GameServerClient[i+1].Server || gsc.Game != GameServerClient[i+1].Game {
				servers += fmt.Sprintf(ServerTemplate, html.EscapeString(gsc.Server), IfElse(gsc.Status == "online", gsc.Curplayers, 0), IfElse(gsc.Status == "online", gsc.Maxplayers, 0), IfElse(gsc.Curplayers > -1, PlayersAvailable, " "))
			}
		}
	}

	// if we have processed no servers, we put the 'no servers available message'
	if len(servers) == 0 {
		servers = "<tr><td colspan='10'>No servers available.</td></tr>"
	} else {

	}

	result := bytes.ReplaceAll(SERVERS_HTML, []byte("$$SERVERS$$"), []byte(servers))
	c.Data(http.StatusOK, gin.MIMEHTML, result)
}

// send the game servers stored to the client in full
// TODO: sort the names, too confusing
func ShowServers(c *gin.Context) {

	GameServerClient, err := txGameServerGetAll()

	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError,
			gin.H{"success": false,
				"message": "Database transaction issue",
				"errors":  []string{err.Error()}})

		return
	}

	if len(GameServerClient) == 0 {
		c.AbortWithStatusJSON(http.StatusNotFound,
			gin.H{"success": false, "message": "No servers available"})

		return

	}

	GameServerSlice := GameServerClient.toGameServerSlice()

	c.IndentedJSON(http.StatusOK, GameServerSlice)
}

// insert/update uploaded server to the database. It also covers delete
func UpsertServer(c *gin.Context) {

	server := GameServer{}

	err1 := c.ShouldBindJSON(&server)
	if err1 != nil && err1.Error() == "EOF" {
		c.AbortWithStatusJSON(http.StatusBadRequest,
			gin.H{"success": false,
				"message": "VALIDATEERR - Invalid Json",
				"errors":  []string{"Submitted Json cannot be parsed"}})
		return
	}

	err2 := server.CheckInput()

	err := errors.Join(err1, err2)

	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest,
			gin.H{"success": false,
				"message": "VALIDATEERR - Invalid Json",
				"errors":  strings.Split(err.Error(), "\n")})
		return
	}

	err = txGameServerUpsert(server)

	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError,
			gin.H{"success": false,
				"message": "Database transaction issue",
				"errors":  []string{err.Error()}})

		return
	}

	if len(EVTSERVER_WEBHOOK) > 0 {
		go CallEventWebHook("POST", server, 2*time.Second)
	}

	c.JSON(http.StatusCreated, gin.H{"success": true,
		"message": "Server correctly updated"})
}

// sends back the current server version + uptime
func ShowStatus(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"success": true,
		"version": STRINGVER,
		"uptime":  uptime(STARTEDON)})
}

// show documentation in html
func ShowDocs(c *gin.Context) {
	c.Data(http.StatusOK, gin.MIMEHTML, DOCHTML)
}

// delete server from database. It doesn't check if it exists.
func DeleteServer(c *gin.Context) {

	server := GameServerDelete{}

	err1 := c.ShouldBindJSON(&server)
	if err1 != nil && err1.Error() == "EOF" {
		c.AbortWithStatusJSON(http.StatusBadRequest,
			gin.H{
				"success": false,
				"message": "VALIDATEERR - Invalid Json",
				"errors":  []string{"Submitted Json cannot be parsed"}})
		return
	}

	err2 := server.CheckInput()

	err := errors.Join(err1, err2)

	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest,
			gin.H{
				"success": false,
				"message": "VALIDATEERR - Invalid Json",
				"errors":  strings.Split(err.Error(), "\n")})
		return
	}

	err = txGameServerDelete(server.Serverurl)

	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
			"success": false, "message": "Database transaction issue",
			"errors": []string{err.Error()}})

		return
	}

	if len(EVTSERVER_WEBHOOK) > 0 {
		go CallEventWebHook("DELETE", server, 2*time.Second)
	}

	c.JSON(http.StatusNoContent, gin.H{"success": true,
		"message": "Server correctly deleted"})
}

// update the status of the server to the eventserver webhook
// supports updates (POST) and deletion (DELETE)
func CallEventWebHook(method string, ServerData any, time time.Duration) error {

	json, err := json.MarshalIndent(ServerData, "", "\t")
	if err != nil {
		ERROR.Printf("Unable to json.Marshal %v", ServerData)
		return err
	}

	req, err := http.NewRequest(method, EVTSERVER_WEBHOOK, bytes.NewBuffer(json))

	if err != nil {
		ERROR.Printf("Unable to create http.NewRequest for event webhook")
		return err
	}
	req.Header.Set("X-Lobby-Client", VERSION)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: time}

	resp, err := client.Do(req)
	if err != nil {
		ERROR.Printf("Unable to post event to webhook: %s", EVTSERVER_WEBHOOK)
		return err
	}
	defer resp.Body.Close()

	return nil
}
