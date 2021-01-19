// Copyright 2021 Braden Nicholson. All rights reserved.

package enc

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
)

// Enclave represents a temporary on-disk file structure and an interface to manipulate said data.
//
// An Enclave can be reused so long as operations are done through an interface.
type Enclave struct {
	Path string
}

// When we first initialize the Enclave struct, we need to assign a UUID.
func NewEnclave() *Enclave {
	enc := &Enclave{}
	enc.allocateEnclave()
	return enc
}

func (e *Enclave) allocateEnclave() {
	p := os.TempDir()
	path, err := ioutil.TempDir(p, "*-enclave")
	if err != nil {
		panic(err)
	}
	e.Path = path
}

// Add create and add an empty file to the enclave
func (e *Enclave) emplaceFile(name string) *os.File {
	path := fmt.Sprintf("%s/%s", e.Path, name)
	fmt.Println(path)
	file, err := os.Create(path)
	if err != nil {
		panic(err)
	}
	return file
}

// Add create and add an empty file to the enclave
func (e *Enclave) DownloadFile(name string, url string) {
	response, err := http.Get(url)
	if err != nil {
		panic(err)
	}
	defer response.Body.Close()

	file := e.emplaceFile(name)
	_, err = io.Copy(file, response.Body)

}

func (e *Enclave) Close() {
	os.RemoveAll(e.Path)
}
