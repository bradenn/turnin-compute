package submit

import (
	"bytes"
	"context"
	"os/exec"
	"strings"
	"time"
)

type Configuration struct {
	Cmd     string   `json:"cmd"`
	Args    []string `json:"args"`
	Exit    int      `json:"exit"`
	Timeout int      `json:"timeout"`
	Stdout  []string `json:"stdout"`
	Stderr  []string `json:"stderr"`
}

type Compilation struct {
	Time   string   `json:"time"`
	Exit   int      `json:"exit"`
	Stdout []string `json:"stdout"`
	Stderr []string `json:"stderr"`
}

func (s *Submission) Compile() error {

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(s.Configuration.Timeout)*time.Millisecond)
	defer cancel()

	cmd := exec.CommandContext(ctx, s.Configuration.Cmd)
	cmd.Dir = s.enclave.Path

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	s.Compilation = Compilation{
		Time:   (cmd.ProcessState.UserTime() + cmd.ProcessState.SystemTime()).String(),
		Exit:   cmd.ProcessState.ExitCode(),
		Stdout: strings.Split(string(stdout.Bytes()), "\n"),
		Stderr: strings.Split(string(stderr.Bytes()), "\n"),
	}

	return err
}
