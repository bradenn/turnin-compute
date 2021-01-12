package server

import (
	"github.com/gin-gonic/gin"
	"../controllers"

)


func NewRouter() *gin.Engine {
	r := gin.New()

	compile := new(controllers.CompileController)

	api := r.Group("api")
	{
		api.POST("/compile", compile.Compile)
	}
	return r
}
