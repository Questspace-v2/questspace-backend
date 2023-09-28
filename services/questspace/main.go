package main

import (
	"fmt"
	"net/http"
	"questspace/pkg/application"

	"github.com/gin-gonic/gin"
	"golang.org/x/xerrors"
)

var config struct {
	Section struct {
		Key string `yaml:"key"`
	} `yaml:"section"`
}

var reqCount = 0

// HandleHello is test handler to check OK and err responses
func HandleHello(c *gin.Context) error {
	if reqCount == 2 {
		return xerrors.New("Too many requests")
	}
	reqCount++
	c.JSON(http.StatusOK, gin.H{"message": "hello"})
	return nil
}

func Init(app application.App) error {
	fmt.Printf("Got key: %s", config.Section.Key)

	app.Router().GET("/hello", application.AsGinHandler(HandleHello))
	return nil
}

func main() {
	application.Run(Init, &config)
}
