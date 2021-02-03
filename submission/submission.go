package submission

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

// acquireResources method of Submission
// This function downloads all test files and source code
func (s *Submission) acquireResources() {

	fg := new(sync.WaitGroup) // Define the file WaitGroup
	fg.Add(len(s.Files))      // Set the total number of files to expect

	// Download each file, concurrently.
	for _, f := range s.Files {
		go func(f File) {
			defer fg.Done()
			s.enclave.DownloadFile(f.Name, "", f.Link)
		}(f)
	}

	tg := new(sync.WaitGroup)       // Define the test WaitGroup
	tg.Add(len(s.Grader.Tests) * 3) // (Total # of tests) * (input, output, error)

	// Download all tests required for the experiment.
	// I'd like to aggregate these files into a tarball before they leave the file server, but for now,
	// one at a time is the fastest way. Spawning 3 new threads is a bit silly for this use case,
	// but it speeds up the acquisition process.
	for _, t := range s.Grader.Tests {
		go func(t Test) {
			defer tg.Done()
			s.enclave.DownloadFile(t.Stdin.Name, "tests", t.Stdin.Link)
		}(t)
		go func(t Test) {
			defer tg.Done()
			s.enclave.DownloadFile(t.Stdout.Name, "tests", t.Stdout.Link)
		}(t)
		go func(t Test) {
			defer tg.Done()
			s.enclave.DownloadFile(t.Stderr.Name, "tests", t.Stderr.Link)
		}(t)
	}

	fg.Wait() // Wait on files
	tg.Wait() // Wait on tests
}
