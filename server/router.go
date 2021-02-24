package server

import (
	"github.com/bradenn/turnin-compute/controllers"
	"github.com/gin-gonic/gin"
)

func NewRouter() (r *gin.Engine) {
	r = gin.New()

	submission := new(controllers.SubmissionController)
	git := new(controllers.GitController)

	api := r.Group("/api/v1")
	{
		api.POST("/submit", submission.Submit)
		api.POST("/git", git.Git)
	}

	return
}
