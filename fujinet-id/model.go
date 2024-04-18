package main

import (
	"errors"
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"
)

type JSONInput interface {
	AdditionalInputCheck() error
	ShouldBindJSON(c *gin.Context) error
}

// used for form checking when asking for a pubkey+token
type PrivKey struct {
	PrivKey string `json:"privkey" binding:"required,printascii"`
}

func (pk *PrivKey) AdditionalInputCheck() (err error) {

	return errors.Join(
		ErrorIf(len(pk.PrivKey) < 16, fmt.Errorf("error: Field validation for 'privkey' length cannot be less than 16 chars")),
		ErrorIf(!strings.Contains(pk.PrivKey, "#"), fmt.Errorf("error: Field validation for 'privkey' must include '#' character")),
	)

}

func (pk *PrivKey) ShouldBindJSON(c *gin.Context) (err error) {

	err1 := c.ShouldBindJSON(&pk)
	if err1 != nil && err1.Error() == "EOF" {
		return fmt.Errorf("submitted Json cannot be parsed")
	}

	err2 := pk.AdditionalInputCheck()

	return errors.Join(err1, err2)
}

// used for form checking and sending the data back to the client
type Token struct {
	Token string `json:"token" binding:"required,printascii"`
}

func (tk *Token) AdditionalInputCheck() (err error) {

	return errors.Join(
		ErrorIf(len(tk.Token) < 16, fmt.Errorf("error: Field validation for 'token' length cannot be less than 16 chars")),
	)
}

func (tk *Token) ShouldBindJSON(c *gin.Context) (err error) {
	err1 := c.ShouldBindJSON(&tk)
	if err1 != nil && err1.Error() == "EOF" {
		return fmt.Errorf("submitted Json cannot be parsed")
	}

	err2 := tk.AdditionalInputCheck()

	return errors.Join(err1, err2)

}

// used to retrieve the data from the database
type PubKeyRecord struct {
	Pubkey     string
	Token      string
	Created_on string
}

// Do additional checking

// rogersm#secret --> rogersm, secret
func splitPrivKey(privkey string) (username string, password string) {
	return split2(privkey, "#")
}

// rogersm!asdfgh --> rogersm, asdfgh
func splitPubKey(pubkey string) (username string, tripcode string) {
	return split2(pubkey, "!")
}
