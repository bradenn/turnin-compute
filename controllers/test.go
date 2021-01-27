package controllers

import (
	"fmt"
	"github.com/bradenn/turnin-compute/submit"
	"github.com/gin-gonic/gin"
	"net/http"
)

type TestController struct{}

func (t TestController) Test(c *gin.Context) {
	var json submit.Submission
	if c.BindJSON(&json) == nil {
		err := json.Run()
		fmt.Println(err)
		c.JSON(200, json.Response)
	} else {
		c.JSON(http.StatusBadRequest, http.ErrBodyNotAllowed)
	}
}
