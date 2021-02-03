package controllers

import (
	"github.com/bradenn/turnin-compute/submission"
	"github.com/gin-gonic/gin"
	"net/http"
)

type SubmissionController struct{}

func (t SubmissionController) Submit(c *gin.Context) {
	var json submission.Submission
	if c.BindJSON(&json) == nil {
		err := json.Run()
		if err != nil {
			c.JSON(http.StatusBadRequest, err)
		}
		c.JSON(200, json.Response)
	} else {
		c.JSON(http.StatusBadRequest, http.ErrBodyNotAllowed)
	}
}
