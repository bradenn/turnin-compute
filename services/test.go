package services

import (
	"bytes"
	"context"
	"fmt"
	"github.com/bradenn/turnin-compute/schemas"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strings"
	"sync"
	"syscall"
	"time"
)

func getTestFileBuffer(path string, fileName string) ([]byte, error) {

	res, err := ioutil.ReadFile(fmt.Sprintf("%s/tests/%s", path, fileName))
	if err != nil {
		return res, err
	}

	return res, nil
}

func (res *SubmissionResultSchema) RunTests(executable string, path string, submission schemas.SubmissionSchema) error {

	fatalErrors := make(chan error)
	wgDone := make(chan bool)

	var testWg sync.WaitGroup
	testWg.Add(len(submission.SubmissionTests))

	for _, test := range submission.SubmissionTests {
		go func(res *SubmissionResultSchema, test schemas.SubmissionTest) {
			defer testWg.Done()

			testResult := new(SubmissionTestResult)
			testResult.ID = test.ID

			err := testResult.ExecuteTest(executable, path, test)
			if err != nil {
				fatalErrors <- err
			}

			err = testResult.GetDiffs(path, test)
			if err != nil {
				fatalErrors <- err
			}

			res.SubmissionTestResults = append(res.SubmissionTestResults, *testResult)

		}(res, test)
	}

	go func() {
		testWg.Wait()
		close(wgDone)
	}()

	select {
	case <-wgDone:
		break
	case err := <-fatalErrors:
		log.Fatal("Error: ", err)
		return err
	}

	return nil
}

func (res *SubmissionTestResult) ExecuteTest(executable string, path string, test schemas.SubmissionTest) error {

	if test.TestMemoryLeaks {
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
		res.MemoryLeaks = strings.Split(string(mem), "\n")

	}

	var ctx, cancel = context.WithTimeout(context.Background(), time.Millisecond*time.Duration(test.TestTimeout))
	cmd := exec.CommandContext(ctx, executable, test.TestArguments...)
	defer cancel()
	cmd.Dir = path

	buffer := bytes.Buffer{}
	writeStream, err := getTestFileBuffer(path, test.TestInput.FileName)
	if err != nil {
		return err
	}
	buffer.Write(writeStream)

	cmd.Stdin = &buffer

	if len(test.TestOutput.FileName) > 0 {
		outfile, err := os.Create(fmt.Sprintf("%s/results/%s", path, test.TestOutput.FileName))
		if err != nil {
			return err
		}

		defer outfile.Close()

		cmd.Stdout = outfile
	}

	if len(test.TestError.FileName) > 0 {
		errfile, err := os.Create(fmt.Sprintf("%s/results/%s", path, test.TestError.FileName))
		if err != nil {
			return err
		}

		defer errfile.Close()

		cmd.Stderr = errfile
	}

	err = cmd.Start()
	if err != nil {
		return err
	}

	err = cmd.Wait()
	if err != nil {
		if err.Error() == "signal: killed" {
			res.ErrorFlags = append(res.ErrorFlags, ErrorFlag{Type: "timeout"})
		}
	}

	res.TestSystemTime = cmd.ProcessState.SystemTime().String()
	res.TestUserTime = cmd.ProcessState.UserTime().String()
	res.TestElapsedTime = (cmd.ProcessState.SystemTime() + cmd.ProcessState.UserTime()).String()
	res.TestExitCode = cmd.ProcessState.ExitCode()
	res.BytesUsed = int(cmd.ProcessState.SysUsage().(*syscall.Rusage).Maxrss)

	return nil
}

func (res *SubmissionTestResult) GetDiffs(path string, test schemas.SubmissionTest) error {
	if len(test.TestOutput.FileName) > 0 {
		pathOutput := fmt.Sprintf("%s/results/%s", path, test.TestOutput.FileName)
		pathOriginal := fmt.Sprintf("%s/tests/%s", path, test.TestOutput.FileName)
		res.TestOutputDiff = GetDiff(pathOutput, pathOriginal)
		if len(res.TestOutputDiff) == 1 && res.TestExitCode == test.TestExitCode {
			res.TestPassed = true
		}
	}

	if len(test.TestError.FileName) > 0 {
		pathOutputError := fmt.Sprintf("%s/results/%s", path, test.TestError.FileName)
		pathOriginalError := fmt.Sprintf("%s/tests/%s", path, test.TestError.FileName)
		res.TestErrorDiff = GetDiff(pathOutputError, pathOriginalError)
		if len(res.TestErrorDiff) == 1 && res.TestExitCode == test.TestExitCode {
			res.TestPassed = true
		}
	}
	return nil
}

func GetDiff(pathOutput string, pathOriginal string) []string {

	cmd := exec.Command("diff", pathOutput, pathOriginal)

	stdout, _ := cmd.CombinedOutput()

	return strings.Split(string(stdout), "\n")
}

func GetFile(path string) []string {

	cmd := exec.Command("cat", path)

	stdout, _ := cmd.CombinedOutput()

	fmt.Println(path)
	return strings.Split(string(stdout), "\n")
}

func RunMemoryTest(pathOutput string, pathOriginal string) []string {

	cmd := exec.Command("diff", pathOutput, pathOriginal)

	stdout, _ := cmd.CombinedOutput()

	return strings.Split(string(stdout), "\n")
}
