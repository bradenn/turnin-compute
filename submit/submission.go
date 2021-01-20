package submit

import (
	"github.com/bradenn/turnin-compute/enc"
	"sync"
)

type Submission struct {
	Id          string      `json:"_id"`
	Enclave     enc.Enclave `json:"enclave"`
	Files       []File      `json:"files"`
	Tests       []Test      `json:"tests"`
	Results     []Result    `json:"result"`
	Compilation Compilation `json:"compilation"`
}

type File struct {
	Id        string `json:"_id"`
	Name      string `json:"name"`
	Reference string `json:"reference"`
	Link      string `json:"link"`
}

type Compilation struct {
	Cmd     string   `json:"cmd"`
	Args    []string `json:"args"`
	Timeout string   `json:"timeout"`
	Exit    int      `json:"exit"`
	Stdout  string   `json:"stdout"`
	Stderr  string   `json:"stderr"`
}

type Test struct {
	Id      string   `json:"_id"`
	Name    string   `json:"name"`
	Args    []string `json:"args"`
	Env     []string `json:"env"`
	Exit    int      `json:"exit"`
	Leaks   bool     `json:"leaks"`
	Timeout int      `json:"timeout"`
	Stdin   File     `json:"stdin"`
	Stdout  File     `json:"stdout"`
	Stderr  File     `json:"stderr"`
}

type Result struct {
	Id     string   `json:"_id"`
	Name   string   `json:"name"`
	Exit   string   `json:"exit"`
	Time   Time     `json:"time"`
	Leaks  []string `json:"leaks"`
	Stdout string   `json:"stdout"`
	Stderr string   `json:"stderr"`
}

type Time struct {
	Elapsed string `json:"elapsed"`
	User    string `json:"user"`
	System  string `json:"system"`
}

func (s *Submission) Run() {
	e := allocateEnclave()
	s.acquireResources(e)
	s.compile(e)

}

func allocateEnclave() *enc.Enclave {
	e := enc.NewEnclave()
	e.AddDir("tests")
	e.AddDir("results")
	return e
}

func (s *Submission) acquireResources(e *enc.Enclave) {
	wg := new(sync.WaitGroup)
	wg.Add(len(s.Files))
	for _, f := range s.Files {
		go func(f File) {
			defer wg.Done()
			e.DownloadFile(f.Name, "", f.Link)
		}(f)
	}
	// stdin / stdout / stderr
	wg.Add(len(s.Tests) * 3)
	for _, t := range s.Tests {
		go func(t Test) {
			defer wg.Done()
			e.DownloadFile(t.Stdin.Name, "tests", t.Stdin.Link)
		}(t)
		go func(t Test) {
			defer wg.Done()
			e.DownloadFile(t.Stdout.Name, "tests", t.Stdout.Link)
		}(t)
		go func(t Test) {
			defer wg.Done()
			e.DownloadFile(t.Stderr.Name, "tests", t.Stderr.Link)
		}(t)
	}
	// Ensure files download before exiting
	wg.Wait()
}

func (s *Submission) compile(e *enc.Enclave) {

}
