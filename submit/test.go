package submit

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/pmezard/go-difflib/difflib"
	"io/ioutil"
	"os/exec"
	"strings"
	"syscall"
	"time"
)

type Test struct {
	Id      string   `json:"_id"`
	Name    string   `json:"name"`
	Args    []string `json:"args"`
	Env     []string `json:"env"`
	Exit    int      `json:"exit"`  // The exit code
	Leaks   bool     `json:"leaks"` // Check for memory leaks
	Timeout int      `json:"timeout"`
	Stdin   File     `json:"stdin"`
	Stdout  File     `json:"stdout"`
	Stderr  File     `json:"stderr"`
}

type Result struct {
	Passed bool     `json:"passed"`
	Id     string   `json:"_id"`
	Name   string   `json:"name"`
	Memory int      `json:"memory"`
	Exit   int      `json:"exit"`
	Time   Time     `json:"time"`
	Leak   Leak     `json:"leak"`
	Diff   Diff     `json:"diff"`
	Stdout []string `json:"stdout"`
	Stderr []string `json:"stderr"`
}

type Time struct {
	Elapsed string `json:"elapsed"`
	User    string `json:"user"`
	System  string `json:"system"`
}

type Diff struct {
	Passed  bool     `json:"passed"`
	Elapsed string   `json:"elapsed"`
	Stdout  []string `json:"stdout"`
	Stderr  []string `json:"stderr"`
}

type Leak struct {
	Passed  bool   `json:"passed"`
	Elapsed string `json:"elapsed"`
	Pid     int    `json:"pid"`
	Lost    struct {
		Blocks int `json:"blocks"`
		Bytes  int `json:"bytes"`
	} `json:"lost"`
	Runtime struct {
		Allocs int `json:"allocs"`
		Frees  int `json:"frees"`
		Bytes  int `json:"bytes"`
	} `json:"runtime"`
	Leaks []struct {
		Blocks int      `json:"blocks"`
		Bytes  int      `json:"bytes"`
		Trace  []string `json:"trace"`
	} `json:"leaks"`
}

func (t *Test) Run(path string, executable string) (r Result, err error) {

	r = Result{
		Id:     t.Id,
		Name:   t.Name,
		Time:   Time{},
		Stdout: nil,
		Stderr: nil,
	}

	mw := make(chan bool)
	if t.Leaks {
		go checkMemory(*t, path, executable, mw, &r)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(t.Timeout)*time.Millisecond)
	defer cancel()

	cmd := exec.CommandContext(ctx, executable, t.Args...)
	cmd.Dir = path

	buffer := bytes.Buffer{}
	writeStream, err := getTestFileBuffer(path, t.Stdin.Name)
	if err != nil {
		fmt.Println(err)
		return
	}
	buffer.Write(writeStream)

	cmd.Stdin = &buffer

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	_ = cmd.Start()

	_ = cmd.Wait()

	r.Stdout = strings.Split(string(stdout.Bytes()), "\n")
	r.Stderr = strings.Split(string(stderr.Bytes()), "\n")

	r.Memory = int(cmd.ProcessState.SysUsage().(*syscall.Rusage).Maxrss)
	r.Exit = cmd.ProcessState.ExitCode()

	r.Time = Time{
		Elapsed: (cmd.ProcessState.SystemTime() + cmd.ProcessState.UserTime()).String(),
		User:    cmd.ProcessState.UserTime().String(),
		System:  cmd.ProcessState.SystemTime().String(),
	}

	err, r.Diff = generateDiff(*t, path, &r)

	if t.Leaks {
		<-mw // Wait for any mem leaks
	}

	return
}

func checkMemory(t Test, path string, executable string, c chan bool, r *Result) {
	start := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(t.Timeout)*time.Millisecond)
	defer cancel()

	commandString := fmt.Sprintf("heapusage -o results/%s.mem -n %s %s < tests/%s > /dev/null 2> /dev/null && cat results/%s.mem",
		t.Name, executable, fmt.Sprint(t.Args), t.Stdin.Name, t.Name)

	cmd := exec.CommandContext(ctx, "bash", "-c", commandString)
	cmd.Dir = path

	buffer := bytes.Buffer{}
	writeStream, err := getTestFileBuffer(path, t.Stdin.Name)
	if err != nil {
		return
	}
	buffer.Write(writeStream)

	mem, err := cmd.CombinedOutput()
	leak := &Leak{}
	err = json.Unmarshal(mem, leak)

	r.Leak = *leak

	c <- true

	r.Leak.Elapsed = time.Now().Sub(start).String()
	return
}

func generateDiff(t Test, path string, r *Result) (err error, diff Diff) {
	start := time.Now()

	stdoutOriginal, err := getTestFileBuffer(path, t.Stdout.Name)

	if len(stdoutOriginal) > 0 {
		udOut := difflib.UnifiedDiff{
			A:        strings.Split(string(stdoutOriginal), "\n"),
			FromFile: fmt.Sprintf("tests/%s.out", t.Name),
			FromDate: "",
			B:        r.Stdout,
			ToFile:   fmt.Sprintf("results/%s.out", t.Name),
			ToDate:   "",
			Eol:      "",
			Context:  0,
		}
		diffOut, _ := difflib.GetUnifiedDiffString(udOut)
		diff.Stdout = strings.Split(diffOut, "\n")
	}

	stderrOriginal, err := getTestFileBuffer(path, t.Stderr.Name)

	if len(stderrOriginal) > 0 {
		udErr := difflib.UnifiedDiff{
			A:        strings.Split(string(stderrOriginal), "\n"),
			FromFile: fmt.Sprintf("tests/%s.err", t.Name),
			FromDate: "",
			B:        r.Stdout,
			ToFile:   fmt.Sprintf("results/%s.err", t.Name),
			ToDate:   "",
			Eol:      "",
			Context:  0,
		}

		diffErr, _ := difflib.GetUnifiedDiffString(udErr)

		diff.Stderr = strings.Split(diffErr, "\n")
	}

	diff.Elapsed = time.Now().Sub(start).String()
	return
}

func getTestFileBuffer(path string, fileName string) ([]byte, error) {

	res, err := ioutil.ReadFile(fmt.Sprintf("%s/tests/%s", path, fileName))
	if err != nil {
		return res, err
	}

	return res, nil
}
