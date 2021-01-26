package submit

import (
	"os/exec"
	"sync"
)

type Grader struct {
	Tests []Test `json:"tests"`
}

type Grades struct {
	Results []Result `json:"tests"`
}

func (g *Grader) Grade(path string) (err error, r Grades) {
	tests := g.Tests

	// Establish waitGroup
	wg := new(sync.WaitGroup)
	wg.Add(len(tests))

	executable, err := getExecutable(path)
	if err != nil {
		return
	}
	res := make(chan Result)
	for _, t := range tests {
		go func(t Test) {
			defer wg.Done()
			r, _ := t.Run(path, executable)
			res <- r
		}(t)
		r.Results = append(r.Results, <-res)
	}
	wg.Wait()
	return
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
