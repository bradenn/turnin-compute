package enclave

import (
	"fmt"
	"github.com/google/uuid"
	"io/fs"
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

// This is more of sanity check method. Use sparingly.
func (e *Enclave) Walk() {
	filepath.Walk(e.Cwd, func(path string, info fs.FileInfo, err error) error {
		fmt.Println(path, info.IsDir())
		return nil
	})
}

func (e *Enclave) generateDirectory() error {
	// There is a 1 in 2^121 chance of an intersection of UUID names.
	// I really don't think it is anything to worry about,
	// especially considering it is highly unlikely for more than one submission to be concurrently running on this
	// module at a time. Should there be an intersection,
	// the submission grading would most definitely fail and the nexus would retest.
	p, _ := filepath.Abs(fmt.Sprintf("./temp/%s", uuid.New()))
	err := os.MkdirAll(p, os.ModePerm)
	if err != nil {
		return err
	}
	e.Cwd = p
	return nil
}
