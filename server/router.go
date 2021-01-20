package server

import (
	"github.com/bradenn/turnin-compute/controllers"
	"github.com/gin-gonic/gin"
)

func NewRouter() *gin.Engine {
	r := gin.New()

	compile := new(controllers.CompileController)
	test := new(controllers.TestController)

	api := r.Group("api")
	{
		api.POST("/compile", compile.Compile)
		api.POST("/test", test.Test)
	}
	return r
}
