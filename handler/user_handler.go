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
	// If user has logged in, force him to log out firstly
	for iter := component.GlobalCache.Iterator(); iter.SetNext(); {
		info, err := iter.Value()
		if err != nil {
			continue
		}
		if string(info.Value()) == username {
			component.GlobalCache.Delete(info.Key())
			log.Printf("forced %s to log out\n", username)
			break
		}
	}
	// Apparently we don't do this in real world :)
	if username == "alice" && password == "111" {
		log.Println(fmt.Sprintf("%s has logged in.", username))
	} else if username == "bob" && password == "123" {
		log.Println(fmt.Sprintf("%s has logged in.", username))
	} else {
		c.JSON(200, component.RestResponse{Message: "no such account"})
		return
	}

	// Generate random session id
	u, err := uuid.NewRandom()
	if err != nil {
		log.Println(fmt.Errorf("failed to generate UUID: %w", err))
	}
	sessionId := fmt.Sprintf("%s-%s", u.String(), username)
	// Store current subject in cache
	err = component.GlobalCache.Set(sessionId, []byte(username))
	if err != nil {
		log.Fatalln(fmt.Errorf("failed to store current subject in cache: %w", err))
		return
	}
	// Send session id back to client in cookie
	c.SetCookie("current_subject", sessionId, 30*60, "/api", "", false, true)
	c.JSON(200, component.RestResponse{Code: 1, Message: username + " logged in successfully"})
}
