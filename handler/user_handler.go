package handler

import (
	"casbin_example/component"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"log"
)

func Login(c *gin.Context) {
	username, password := c.PostForm("username"), c.PostForm("password")
	if username == "alice" && password == "111" {
		log.Println(fmt.Sprintf("%s has logged in.", username))
	} else if username == "bob" && password == "123" {
		log.Println(fmt.Sprintf("%s has logged in.", username))
	} else {
		c.JSON(200, component.RestResponse{Message: "no such account"})
	}
	// Generate random session id
	u, err := uuid.NewRandom()
	if err != nil {
		log.Fatal(err)
	}
	sessionId := fmt.Sprintf("%s-%s", u.String(), username)
	// Store current subject in cache
	component.GlobalCache.Set(sessionId, []byte(username))
	// Send session id back to client in cookie
	c.SetCookie("current_subject", sessionId, 30*60, "/resource", "", false, true)
	c.JSON(200, component.RestResponse{Code: 1, Message:username + " logged in successfully"})
}
