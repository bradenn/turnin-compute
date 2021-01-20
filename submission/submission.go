package submission

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"log"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"
)

type Submission struct {
	Id     string `json:"_id"`
	Files  []File `json:"files"`
	Tests  []Test `json:"tests"`
	Result Result `json:"result"`
}

type File struct {
	Id        string `json:"_id"`
	Name      string `json:"name"`
	Reference string `json:"reference"`
	Link      string `json:"link"`
}

type Test struct {
	Id      string `json:"_id"`
	Name    string `json:"name"`
	Args    string `json:"args"`
	Exit    string `json:"exit"`
	Leaks   bool   `json:"leaks"`
	Timeout int    `json:"timeout"`
	Stdin   string `json:"stdin"`
	Stdout  string `json:"stdout"`
	Stderr  string `json:"stderr"`
}

type Result struct {
	Id        string `json:"_id"`
	Name      string `json:"name"`
	Reference string `json:"reference"`
	Link      string `json:"link"`
}

type ResultSchema struct {
	SubmissionTestResults []TestResult       `json:"submissionTestResults"`
	CompilationResults    CompilationResults `json:"compilationResults"`
	SubmissionFileLint    []FileLint         `json:"submissionFileLint"`
	GradingOptions        GradingOptions     `json:"gradingOptions"`
}

type TestResult struct {
	ID                string      `json:"_id"`
	TestPassed        bool        `json:"testPassed"`
	TestExitCode      int         `json:"testExitCode"`
	TestUserTime      string      `json:"testUserTime"`
	TestSystemTime    string      `json:"testSystemTime"`
	TestElapsedTime   string      `json:"testElapsedTime"`
	TestStatus        string      `json:"testStatus"`
	BytesUsed         int         `json:"bytesUsed"`
	TestOutputDiff    []string    `json:"testOutputDiff"`
	TestErrorDiff     []string    `json:"testErrorDiff"`
	MemoryLeaksReport MemoryLeak  `json:"memoryLeak"`
	ErrorFlags        []ErrorFlag `json:"errorFlags"`
}

type FileLint struct {
	FileID    string   `json:"fileId"`
	LintLines []string `json:"lintLines"`
}

type CompilationResults struct {
	CompilationOutput []string    `json:"compilationOutput"`
	CompilationTime   string      `json:"compilationTime"`
	ErrorFlags        []ErrorFlag `json:"errorFlags"`
}

type CodeLint struct {
	Location bool `json:"lintLocation"`
	Message  bool `json:"lintMessage"`
	Severity bool `json:"listSeverity"`
}

type MemoryLeak struct {
	Summary     []string `json:"leakSummary"`
	Message     bool     `json:"lintMessage"`
	Severity    bool     `json:"listSeverity"`
	ElapsedTime string   `json:"elapsedTime"`
}

type GradingOptions struct {
	LintFiles      bool `json:"lintFiles"`
	SubmissionLang bool `json:"submissionLang"`
}

type ErrorFlag struct {
	Type string `json:"errorType"`
}

func (res *ResultSchema) BuildAndCompileSubmission(submission SubmissionSchema) error {

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
	defer os.RemoveAll(path)

	err = res.LintCode(path, submission)

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

func BuildWorkspace(path string, testPath string, submission SubmissionSchema) error {

	var fileWg sync.WaitGroup
	fileWg.Add(len(submission.SubmissionFiles))

	for _, file := range submission.SubmissionFiles {
		var err error = nil
		go func(file FileReference, err error) {
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
		go func(test SubmissionTest, err error) {
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

func (res *ResultSchema) CompileSubmission(path string, submission SubmissionSchema) error {

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
