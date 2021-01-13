package controllers

import (
	"github.com/bradenn/turnin-compute/schemas"
	"github.com/bradenn/turnin-compute/services"
	"github.com/gin-gonic/gin"
)

type CompileController struct{}

func (e CompileController) Compile(c *gin.Context) {
	var json schemas.SubmissionSchema
	if c.BindJSON(&json) == nil {

		c.JSON(200, services.BuildWorkspace(json))

	}
}
