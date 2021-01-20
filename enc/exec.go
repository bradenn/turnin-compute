package enc

import "bytes"

type Compile struct {
	Path    string
	Args    []string
	Env     []string
	Process Process
	Stdin   bytes.Buffer
	Stdout  bytes.Buffer
	Stderr  bytes.Buffer
}

type Process struct {
	Stdin  bytes.Buffer
	Stdout bytes.Buffer
	Stderr bytes.Buffer
}

func (c *Compile) NewCompiler() *Compile {

	return &Compile{}

}
