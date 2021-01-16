package services

import (
	"bytes"
	"context"
	"fmt"
	"github.com/bradenn/turnin-compute/schemas"
	"github.com/google/uuid"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strings"
	"sync"
	"syscall"
	"time"
)

type SubmissionResultSchema struct {
	SubmissionTestResults []SubmissionTestResult `json:"submissionTestResults"`
	CompilationResults    CompilationResults     `json:"compilationResults"`
	GradingOptions        GradingOptions         `json:"gradingOptions"`
}

type SubmissionTestResult struct {
	ID              string      `json:"_id"`
	TestPassed      bool        `json:"testPassed"`
	TestExitCode    int         `json:"testExitCode"`
	TestUserTime    string      `json:"testUserTime"`
	TestSystemTime  string      `json:"testSystemTime"`
	TestElapsedTime string      `json:"testElapsedTime"`
	TestStatus      string      `json:"testStatus"`
	BytesUsed       int         `json:"bytesUsed"`
	TestOutputDiff  []string    `json:"testOutputDiff"`
	TestErrorDiff   []string    `json:"testErrorDiff"`
	MemoryLeaks     []string    `json:"memoryLeaks"`
	ErrorFlags      []ErrorFlag `json:"errorFlags"`
}

type CompilationResults struct {
	CompilationOutput []string    `json:"compilationOutput"`
	CompilationTime   string      `json:"compilationTime"`
	ErrorFlags        []ErrorFlag `json:"errorFlags"`
}

type GradingOptions struct {
	LintFiles      bool `json:"lintFiles"`
	SubmissionLang bool `json:"submissionLang"`
}

type ErrorFlag struct {
	Type string `json:"errorType"`
}

func (res *SubmissionResultSchema) BuildAndCompileSubmission(submission schemas.SubmissionSchema) error {

	id := generateUUID()

	path, testPath, _, err := AllocateWorkspace(id)
	if err != nil {
		log.Println("Could not allocate workspace.")
		return err
	}

	err = BuildWorkspace(path, testPath, submission)
	if err != nil {
		log.Println("Could not build workspace.")
		return err
	}

	err = res.CompileSubmission(path, submission)

	if err != nil {
		log.Println("Could not compile workspace.", err)
		return err
	}

	executable, err := GetExecutable(path)
	if err != nil {
		log.Println("Could not find the executable.")
		return err
	}

	err = res.RunTests(executable, path, submission)
	if err != nil {
		log.Println("Could not run tests.", err)
		return err
	}

	return nil
}

func generateUUID() uuid.UUID {

	id, err := uuid.NewUUID()

	if err != nil {
		log.Fatalln(err)
	}

	return id
}

func AllocateWorkspace(id uuid.UUID) (string, string, string, error) {

	path := fmt.Sprintf("./temp/%s", id)
	err := os.MkdirAll(path, os.ModePerm)

	testPath := fmt.Sprintf("./temp/%s/tests", id)
	err = os.MkdirAll(testPath, os.ModePerm)

	resultsPath := fmt.Sprintf("./temp/%s/results", id)
	err = os.MkdirAll(resultsPath, os.ModePerm)

	return path, testPath, resultsPath, err
}

func BuildWorkspace(path string, testPath string, submission schemas.SubmissionSchema) error {

	var fileWg sync.WaitGroup
	fileWg.Add(len(submission.SubmissionFiles))

	for _, file := range submission.SubmissionFiles {
		var err error = nil
		go func(file schemas.FileReference, err error) {
			defer fileWg.Done()

			err = EmplaceFile(path, file.FileName, file.FileReference)
		}(file, err)
		if err != nil {
			return err
		}
	}

	var testWg sync.WaitGroup
	testWg.Add(len(submission.SubmissionTests))

	for _, test := range submission.SubmissionTests {
		var err error = nil
		go func(test schemas.SubmissionTest, err error) {
			defer testWg.Done()

			if len(test.TestInput.FileName) > 0 {
				err = EmplaceFile(testPath, test.TestInput.FileName, test.TestInput.FileReference)
			}

			if len(test.TestOutput.FileName) > 0 {
				err = EmplaceFile(testPath, test.TestOutput.FileName, test.TestOutput.FileReference)
			}

			if len(test.TestError.FileName) > 0 {
				err = EmplaceFile(testPath, test.TestError.FileName, test.TestError.FileReference)
			}
		}(test, err)
		if err != nil {
			return err
		}
	}
	fileWg.Wait()
	testWg.Wait()

	return nil
}

func EmplaceFile(path string, fileName string, fileReference string) error {

	filePath := fmt.Sprintf("%s/%s", path, fileName)

	fileLink := fmt.Sprintf("%s/%s/%s", os.Getenv("S3_ENDPOINT"),
		os.Getenv("S3_BUCKET"), fileReference)

	err := DownloadFile(filePath, fileLink)

	return err
}

func (res *SubmissionResultSchema) CompileSubmission(path string, submission schemas.SubmissionSchema) error {

	cmdString := strings.Split(submission.CompilationOptions.CompilationCommand, " ")
	timeout := submission.CompilationOptions.CompilationTimeout
	var ctx, cancel = context.WithTimeout(context.Background(), time.Millisecond*time.Duration(timeout))
	var cmd *exec.Cmd

	if len(cmdString) > 1 {
		cmd = exec.CommandContext(ctx, cmdString[0], cmdString[1:]...)
	} else {
		cmd = exec.CommandContext(ctx, cmdString[0])
	}
	cmd.Dir = path
	defer cancel()

	stdout, err := cmd.CombinedOutput()
	res.CompilationResults.CompilationOutput = strings.Split(string(stdout), "\n")
	if err != nil {
		res.CompilationResults.ErrorFlags = append(res.CompilationResults.ErrorFlags, ErrorFlag{Type: "stderr"})
	}
	if cmd.ProcessState != nil {
		res.CompilationResults.CompilationTime = cmd.ProcessState.SystemTime().String()
	}

	return err
}

func GetExecutable(path string) (string, error) {
	var executable string

	cmd := exec.Command("find", ".", "-perm", "+111", "-type", "f")
	cmd.Dir = path

	stdout, err := cmd.Output()
	executable = string(stdout)

	executable = executable[:len(executable)-1]
	return executable, err
}

func getTestFileBuffer(path string, fileName string) ([]byte, error) {

	res, err := ioutil.ReadFile(fmt.Sprintf("%s/tests/%s", path, fileName))
	if err != nil {
		return res, err
	}

	return res, nil
}

func (res *SubmissionResultSchema) RunTests(executable string, path string, submission schemas.SubmissionSchema) error {

	var testWg sync.WaitGroup
	testWg.Add(len(submission.SubmissionTests))

	for _, test := range submission.SubmissionTests {
		var err error = nil
		go func(res *SubmissionResultSchema, test schemas.SubmissionTest, err error) {
			defer testWg.Done()

			testResult := new(SubmissionTestResult)
			err = testResult.ExecuteTest(executable, path, test)
			if err != nil {
				return
			}
			testResult.ID = test.ID

			if len(test.TestOutput.FileName) > 0 {
				pathOutput := fmt.Sprintf("%s/results/%s", path, test.TestOutput.FileName)
				pathOriginal := fmt.Sprintf("%s/tests/%s", path, test.TestOutput.FileName)
				testResult.TestOutputDiff = GetDiff(pathOutput, pathOriginal)
				if len(testResult.TestOutputDiff) == 1 && testResult.TestExitCode == test.TestExitCode {
					testResult.TestPassed = true
				}
			}

			if len(test.TestError.FileName) > 0 {
				pathOutputError := fmt.Sprintf("%s/results/%s", path, test.TestError.FileName)
				pathOriginalError := fmt.Sprintf("%s/tests/%s", path, test.TestError.FileName)
				testResult.TestErrorDiff = GetDiff(pathOutputError, pathOriginalError)
				if len(testResult.TestErrorDiff) == 1 && testResult.TestExitCode == test.TestExitCode {
					testResult.TestPassed = true
				}
			}

			res.SubmissionTestResults = append(res.SubmissionTestResults, *testResult)

		}(res, test, err)
	}

	testWg.Wait()

	return nil
}

func (res *SubmissionTestResult) ExecuteTest(executable string, path string, test schemas.SubmissionTest) error {

	if test.TestMemoryLeaks {
		var ctx, cancel = context.WithTimeout(context.Background(), time.Millisecond*time.Duration(test.TestTimeout))

		cmd := exec.CommandContext(ctx, "heap", executable)
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
		fmt.Print(err.Error())
		if err.Error() == "signal: killed" {
			res.ErrorFlags = append(res.ErrorFlags, ErrorFlag{Type: "timeout"})
		}
	}

	res.TestSystemTime = cmd.ProcessState.SystemTime().String()
	res.TestUserTime = cmd.ProcessState.SystemTime().String()
	res.TestElapsedTime = cmd.ProcessState.UserTime().String()
	res.TestExitCode = cmd.ProcessState.ExitCode()
	res.BytesUsed = int(cmd.ProcessState.SysUsage().(*syscall.Rusage).Maxrss)

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
