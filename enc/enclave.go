// Copyright 2021 Braden Nicholson. All rights reserved.

package enc

import (
	"fmt"
	"github.com/google/uuid"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
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
	p, _ := filepath.Abs(fmt.Sprintf("./temp/%s", genUUID()))
	err := os.MkdirAll(p, os.ModePerm)
	if err != nil {
		panic(err)
	}
	e.Path = p
}

func genUUID() uuid.UUID {

	id, err := uuid.NewUUID()

	if err != nil {
		log.Fatalln(err)
	}

	return id
}

// Download a file from the internet and add it to the enclave
func (e *Enclave) DownloadFile(name string, sub string, url string) {
	response, err := http.Get(url)
	if err != nil {
		return
	}
	file := e.emplaceFile(name, sub)
	_, _ = io.Copy(file, response.Body)
}

// Add create and add an empty file to the enclave
// File directories must exist!
func (e *Enclave) emplaceFile(name string, sub string) *os.File {
	path := fmt.Sprintf("%s/%s/%s", e.Path, sub, name)
	file, _ := os.Create(path)

	return file
}

// Add create and add an empty file to the enclave
// File directories must exist!
func (e *Enclave) AddDir(name string) {
	path := fmt.Sprintf("%s/%s", e.Path, name)
	err := os.MkdirAll(path, os.ModePerm)
	if err != nil {
		panic("Funny, that shouldn't happen...")
	}
}

// Add create and add an empty file to the enclave
// File directories must exist!
func (e *Enclave) Walk() {
	filepath.Walk(e.Path, func(path string, info fs.FileInfo, err error) error {
		fmt.Println(path, info.IsDir())
		return nil
	})

}

func (e *Enclave) Close() {
	os.RemoveAll(e.Path)
}
