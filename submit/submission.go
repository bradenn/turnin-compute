package submit

import (
	"github.com/bradenn/turnin-compute/enc"
	"sync"
)

type Submission struct {
	Files         []File        `json:"files"`
	Tests         []Test        `json:"tests"`
	Configuration Configuration `json:"configuration"`

	Results     []Result    `json:"results"`
	Compilation Compilation `json:"compilation"`

	enclave *enc.Enclave
}

func (s *Submission) Run() {
	s.enclave = allocateEnclave()
	defer s.enclave.Close()

	s.acquireResources()
	// s.enclave.Walk()
	err := s.Compile()
	if err != nil {
		return
	}

	_ = s.RunTests()
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
	wg.Add(len(s.Tests) * 3)
	for _, t := range s.Tests {
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
