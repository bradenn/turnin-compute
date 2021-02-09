package submission

import (
	"bytes"
	"context"
	"os/exec"
	"strings"
	"time"
)

// The Compiler struct specifies the parameters for compilation.
type Compiler struct {
	Cmd     string   `json:"cmd"`
	Args    []string `json:"args"` // Unimplemented
	Exit    int      `json:"exit"`
	Timeout int      `json:"timeout"`
}

// The Compilation struct specifies the results of the compilation.
type Compilation struct {
	Time   string   `json:"time"`
	Exit   int      `json:"exit"`
	Stdout []string `json:"stdout"`
	Stderr []string `json:"stderr"`
}

// TODO: Convert path based reference to directly access Enclave.

// The Compile method of Compiler
//
// This function compiles a project based on it's specifications.
// This method takes a string "path" as an input and returns both and error and a Compilation struct.
func (c *Compiler) Compile(path string) (err error, comp Compilation) {

	// The command context (ctx) is employed to terminate the fork after the duration defined in the
	// Compiler struct has elapsed.
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(c.Timeout)*time.Millisecond)
	defer cancel() // The cancel function is deferred to when the function exits.

	// cmd is only a struct at this point, it will not do anything until with call Run or Start
	cmd := exec.CommandContext(ctx, "bash", "-c", c.Cmd)
	cmd.Dir = path // Dir is set to the path of the current submission

	// The Stdout and Stderr are piped to respective buffers.
	// The buffers are referenced so the buffers are appended while the running.
	// This reduces the amount of memory used by the fork.
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run() // Run the command

	// Stdout and Stderr buffers are reduced to strings and split into lines
	comp.Stdout = strings.Split(string(stdout.Bytes()), "\n")
	comp.Stderr = strings.Split(string(stderr.Bytes()), "\n")

	// If the process exited without a fight, we can record some statistics
	if cmd.ProcessState != nil {
		comp.Time = (cmd.ProcessState.UserTime() + cmd.ProcessState.SystemTime()).String()
		comp.Exit = cmd.ProcessState.ExitCode()
	}

	return // Exit the function
}
