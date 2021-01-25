package submit

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os/exec"
	"strings"
	"sync"
	"syscall"
	"time"
)

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
	Memory int      `json:"memory"`
	Exit   int      `json:"exit"`
	Time   Time     `json:"time"`
	Leak   Leak     `json:"leak"`
	Heap   []string `json:"heap"`
	Stdout []string `json:"stdout"`
	Stderr []string `json:"stderr"`
}

type Time struct {
	Elapsed string `json:"elapsed"`
	User    string `json:"user"`
	System  string `json:"system"`
}

type Leak struct {
	Pid  int `json:"pid"`
	Lost struct {
		Blocks int `json:"blocks"`
		Bytes  int `json:"bytes"`
	} `json:"lost"`
	Runtime struct {
		Allocs int `json:"allocs"`
		Frees  int `json:"frees"`
		Bytes  int `json:"bytes"`
	} `json:"runtime"`
	Leaks []struct {
		Blocks int `json:"blocks"`
		Bytes  int `json:"bytes"`
		Trace  []struct {
			Address  uint64 `json:"address"`
			Location string `json:"location"`
		} `json:"trace"`
	} `json:"leaks"`
}

func (s *Submission) RunTests() error {
	wg := new(sync.WaitGroup)
	wg.Add(len(s.Tests))

	executable, err := getExecutable(s.enclave.Path)
	if err != nil {
		return err
	}
	res := make(chan Result)
	for _, t := range s.Tests {
		go func(t Test) {
			defer wg.Done()
			r, _ := s.runTest(t, executable)
			res <- r
		}(t)
		s.Results = append(s.Results, <-res)
	}
	wg.Wait()
	return nil
}

func getExecutable(path string) (string, error) {
	var executable string

	cmd := exec.Command("find", ".", "-perm", "+111", "-type", "f")
	cmd.Dir = path

	stdout, err := cmd.Output()
	executable = string(stdout)
	if err != nil {
		return "", err
	}

	executable = executable[:len(executable)-1]
	return executable, err
}

func (s *Submission) runTest(t Test, executable string) (r Result, err error) {

	r = Result{
		Id:     t.Id,
		Name:   t.Name,
		Exit:   -1,
		Time:   Time{},
		Stdout: nil,
		Stderr: nil,
	}

	mw := make(chan bool)
	if t.Leaks {
		go s.checkMemory(t, executable, mw, &r)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(s.Configuration.Timeout)*time.Millisecond)
	defer cancel()

	cmd := exec.CommandContext(ctx, executable, t.Args...)
	cmd.Dir = s.enclave.Path

	buffer := bytes.Buffer{}
	writeStream, err := getTestFileBuffer(s.enclave.Path, t.Stdin.Name)
	if err != nil {
		return
	}
	buffer.Write(writeStream)

	cmd.Stdin = &buffer

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	_ = cmd.Run()

	r.Stdout = strings.Split(string(stdout.Bytes()), "\n")
	r.Stderr = strings.Split(string(stderr.Bytes()), "\n")

	r.Memory = int(cmd.ProcessState.SysUsage().(*syscall.Rusage).Maxrss)
	r.Exit = cmd.ProcessState.ExitCode()

	r.Time = Time{
		Elapsed: (cmd.ProcessState.SystemTime() + cmd.ProcessState.UserTime()).String(),
		User:    cmd.ProcessState.UserTime().String(),
		System:  cmd.ProcessState.SystemTime().String(),
	}
	if t.Leaks {
		<-mw // Wait for any mem leaks
	}
	return r, err
}

func (s *Submission) checkMemory(t Test, executable string, c chan bool, r *Result) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(s.Configuration.Timeout)*time.Millisecond)
	defer cancel()

	cmd := exec.CommandContext(ctx, "bash", "-c", "heapusage -n "+executable+" "+fmt.Sprint(t.
		Args)+" < tests/"+t.Stdin.Name+" > /dev/null")
	cmd.Dir = s.enclave.Path

	buffer := bytes.Buffer{}
	writeStream, err := getTestFileBuffer(s.enclave.Path, t.Stdin.Name)
	if err != nil {
		return
	}
	buffer.Write(writeStream)

	mem, _ := cmd.CombinedOutput()

	leak := new(Leak)
	err = json.Unmarshal(mem, leak)

	r.Leak = *leak

	c <- true
	return
}

func getTestFileBuffer(path string, fileName string) ([]byte, error) {

	res, err := ioutil.ReadFile(fmt.Sprintf("%s/tests/%s", path, fileName))
	if err != nil {
		return res, err
	}

	return res, nil
}
