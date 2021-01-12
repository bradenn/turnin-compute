package controllers

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

type CompileController struct{}

func (e CompileController) Compile(c *gin.Context) {
		c.String(http.StatusOK, "Compiler Found!")
}
