package submission

import (
	"bytes"
	"context"
	"os/exec"
	"strings"
	"time"
)

func (res *TestResult) GenerateLeakReport(executable string, path string, test SubmissionTest) error {
	start := time.Now()
	var ctx, cancel = context.WithTimeout(context.Background(), time.Millisecond*time.Duration(test.TestTimeout))

	cmd := exec.CommandContext(ctx, "heapusage", executable)
	defer cancel()
	cmd.Dir = path

	buffer := bytes.Buffer{}
	writeStream, err := getTestFileBuffer(path, test.TestInput.FileName)
	if err != nil {
		return err
	}
	buffer.Write(writeStream)
	mem, _ := cmd.CombinedOutput()
	leakReport := new(MemoryLeak)
	leakReport.Summary = strings.Split(string(mem), "\n")

	leakReport.ElapsedTime = time.Now().Sub(start).String()
	res.MemoryLeaksReport = *leakReport
	return nil
}
