package controllers

import (
	"github.com/bradenn/turnin-compute/schemas"
	"github.com/bradenn/turnin-compute/services"
	"github.com/gin-gonic/gin"
	"net/http"
)

type CompileController struct{}

func (e CompileController) Compile(c *gin.Context) {
	var json schemas.SubmissionSchema
	if c.BindJSON(&json) == nil {
		obj, err := services.BuildAndCompileSubmission(json)
		if err != nil {
			c.JSON(http.StatusBadRequest, http.ErrBodyNotAllowed)
		} else {
			c.JSON(200, obj)
		}
	} else {
		c.JSON(http.StatusBadRequest, http.ErrBodyNotAllowed)
	}
}
