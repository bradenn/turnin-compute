package controllers

import (
	"fmt"
	"github.com/bradenn/turnin-compute/enc"
	"github.com/bradenn/turnin-compute/submission"
	"github.com/gin-gonic/gin"
	"net/http"
	"sync"
)

type TestController struct{}

func (t TestController) Test(c *gin.Context) {
	var json submission.SubmissionSchema
	if c.BindJSON(&json) == nil {
		e := enc.NewEnclave()
		var wg sync.WaitGroup
		wg.Add(len(json.SubmissionTests)*3 + len(json.SubmissionFiles))
		for _, i := range json.SubmissionFiles {
			go func(f submission.FileReference) {
				defer wg.Done()
				e.DownloadFile(f.FileName, "", fmt.Sprintf("http://10.0.0.6:8333/csuchico/%s", f.FileReference))
			}(i)
		}
		e.AddDir("tests")
		for _, test := range json.SubmissionTests {
			go func(t submission.SubmissionTest) {
				defer wg.Done()
				e.DownloadFile(t.TestInput.FileName, "tests",
					fmt.Sprintf("http://10.0.0.6:8333/csuchico/%s",
						t.TestInput.FileReference))
			}(test)
			go func(t submission.SubmissionTest) {
				defer wg.Done()
				e.DownloadFile(t.TestOutput.FileName, "tests",
					fmt.Sprintf("http://10.0.0.6:8333/csuchico/%s",
						t.TestOutput.FileReference))
			}(test)
			go func(t submission.SubmissionTest) {
				defer wg.Done()
				e.DownloadFile(t.TestError.FileName, "tests",
					fmt.Sprintf("http://10.0.0.6:8333/csuchico/%s",
						t.TestError.FileReference))
			}(test)

		}
		wg.Wait()
		e.Walk()
		e.AddDir("results")

		e.Close()
		c.JSON(200, gin.H{})

	} else {
		c.JSON(http.StatusBadRequest, http.ErrBodyNotAllowed)
	}
}
