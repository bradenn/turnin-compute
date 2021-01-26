package controllers

import (
	"github.com/bradenn/turnin-compute/submit"
	"github.com/gin-gonic/gin"
	"net/http"
)

type TestController struct{}

func (t TestController) Test(c *gin.Context) {
	var json submit.Submission
	if c.BindJSON(&json) == nil {
		json.Run()
		c.JSON(200, json)
	} else {
		c.JSON(http.StatusBadRequest, http.ErrBodyNotAllowed)
	}
}
