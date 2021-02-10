package enclave

import (
	"fmt"
	"github.com/google/uuid"
	"io/fs"
	"log"
	"os"
	"path/filepath"
)

type Enclave struct {
	Cwd        string
	Repository string
	Commit     string
}

// Instantiate a new Enclave
func NewEnclave() (e *Enclave, err error) {
	e = &Enclave{}
	err = e.generateDirectory()
	return e, nil
}

func (e *Enclave) Walk() {
	filepath.Walk(e.Cwd, func(path string, info fs.FileInfo, err error) error {
		fmt.Println(path, info.IsDir())
		return nil
	})
}

func genUUID() uuid.UUID {
	id, err := uuid.NewUUID()
	if err != nil {
		log.Fatalln(err) // If this is triggered, hell might as well have frozen over
	}
	return id
}

func (e *Enclave) generateDirectory() error {
	p, _ := filepath.Abs(fmt.Sprintf("./temp/%s", genUUID()))
	err := os.MkdirAll(p, os.ModePerm)
	if err != nil {
		return err
	}
	e.Cwd = p
	return nil
}
