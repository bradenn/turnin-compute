package services

import (
	"fmt"
	"io"
	"net/http"
	"os"
)

func FetchFile(fileKey string, fileName string) <-chan bool {

	res := make(chan bool)

	go func() {
		err := downloadFile(fmt.Sprintf("temp/%s", fileName), fmt.Sprintf("http://10.0.0.6:8333/csuchico/%s", fileKey))
		if err != nil {
			res <- false
		}
		res <- true
	}()
	return res

}

func downloadFile(path string, url string) error {
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
