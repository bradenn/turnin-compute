package server

import (
	"fmt"
	"github.com/bradenn/turnin-compute/controllers"
	"github.com/bradenn/turnin-compute/enc"
	"github.com/gin-gonic/gin"
)

func NewRouter() *gin.Engine {
	r := gin.New()

	compile := new(controllers.CompileController)

	api := r.Group("api")
	{
		api.POST("/compile", compile.Compile)
		api.POST("/test", func(c *gin.Context) {
			e := enc.NewEnclave()
			e.DownloadFile("main.cpp", "http://10.0.0.6:8333/csuchico/1e8c74eb-4a05-450b-9f8b-45c8bfdcb89e")
			fmt.Println(e)
			e.Close()
			c.JSON(200, gin.H{})
		})
	}
	return r
}
