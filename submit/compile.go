package submit

import (
	"bytes"
	"context"
	"os/exec"
	"strings"
	"time"
)

type Compiler struct {
	Cmd     string   `json:"cmd"`
	Args    []string `json:"args"`
	Exit    int      `json:"exit"`
	Timeout int      `json:"timeout"`
}

type Compilation struct {
	Time   string   `json:"time"`
	Exit   int      `json:"exit"`
	Stdout []string `json:"stdout"`
	Stderr []string `json:"stderr"`
}

func (c *Compiler) Compile(path string) (err error, comp Compilation) {

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(c.Timeout)*time.Millisecond)
	defer cancel()

	cmd := exec.CommandContext(ctx, c.Cmd)
	cmd.Dir = path

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()

	comp = Compilation{
		Time:   (cmd.ProcessState.UserTime() + cmd.ProcessState.SystemTime()).String(),
		Exit:   cmd.ProcessState.ExitCode(),
		Stdout: strings.Split(string(stdout.Bytes()), "\n"),
		Stderr: strings.Split(string(stderr.Bytes()), "\n"),
	}

	return
}
