package submit

import (
	"github.com/bradenn/turnin-compute/enc"
	"sync"
)

type Submission struct {
	Files    []File   `json:"files"`
	Grader   Grader   `json:"grader"`
	Compiler Compiler `json:"compiler"`
	Response Response `json:"response"`
	enclave  *enc.Enclave
}

type Response struct {
	Grades      Grades      `json:"grades"`
	Compilation Compilation `json:"compilation"`
}

func (s *Submission) Run() (err error) {
	s.enclave = allocateEnclave()
	path := s.enclave.Path
	defer s.enclave.Close()
	s.acquireResources()

	// Compile the submission
	err, s.Response.Compilation = s.Compiler.Compile(path)
	if err != nil {
		return
	}

	// Run all of the tests
	err, s.Response.Grades = s.Grader.Grade(path)
	if err != nil {
		return
	}

	return
}

func allocateEnclave() *enc.Enclave {
	e := enc.NewEnclave()
	e.AddDir("tests")
	e.AddDir("results")
	return e
}

func (s *Submission) acquireResources() {
	wg := new(sync.WaitGroup)
	wg.Add(len(s.Files))
	for _, f := range s.Files {
		go func(f File) {
			defer wg.Done()
			s.enclave.DownloadFile(f.Name, "", f.Link)
		}(f)
	}
	wg.Wait()
	wg = new(sync.WaitGroup)
	wg.Add(len(s.Grader.Tests) * 3)
	for _, t := range s.Grader.Tests {
		go func(t Test) {
			defer wg.Done()
			s.enclave.DownloadFile(t.Stdin.Name, "tests", t.Stdin.Link)
		}(t)
		go func(t Test) {
			defer wg.Done()
			s.enclave.DownloadFile(t.Stdout.Name, "tests", t.Stdout.Link)
		}(t)
		go func(t Test) {
			defer wg.Done()
			s.enclave.DownloadFile(t.Stderr.Name, "tests", t.Stderr.Link)
		}(t)
	}
	// Ensure files download before exiting
	wg.Wait()
}
