package controllers

import (
	"github.com/bradenn/turnin-compute/submission"
	"github.com/gin-gonic/gin"
	"net/http"
)

type CompileController struct{}

func (e CompileController) Compile(c *gin.Context) {
	var json submission.SubmissionSchema
	if c.BindJSON(&json) == nil {
		results := new(submission.ResultSchema)
		err := results.BuildAndCompileSubmission(json)
		if err != nil {
			c.JSON(http.StatusBadRequest, http.ErrBodyNotAllowed)
		} else {
			c.JSON(200, results)
		}
	} else {
		c.JSON(http.StatusBadRequest, http.ErrBodyNotAllowed)
	}
}
