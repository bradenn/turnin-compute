package services

import (
	"io"
	"net/http"
	"os"
)

func DownloadFile(path string, url string) error {
	response, err := http.Get(url)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	output, err := os.Create(path)
	if err != nil {
		return err
	}
	defer output.Close()
	_, err = io.Copy(output, response.Body)
	return err
}
