package controllers

import (
	"fmt"
	"github.com/bradenn/turnin-compute/enclave"
	"github.com/gin-gonic/gin"
	"time"
)

type GitController struct{}

func (t GitController) Git(c *gin.Context) {
	enc, err := enclave.NewEnclave()
	if err != nil {
		c.String(200, err.Error())
	}

	repo := &enclave.Repository{
		URL:    "https://github.com/bradenn/website",
		Commit: "3d3c32a273073feca60d2501f6511d110a920226",
	}

	start := time.Now()

	err = repo.CloneRepository(enc)

	fmt.Println(time.Since(start))

	if err != nil {
		c.String(200, err.Error())
	}

	// enc.Walk()

	c.String(200, "")
}
