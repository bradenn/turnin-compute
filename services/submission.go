package services

import (
	"bytes"
	"fmt"
	"github.com/bradenn/turnin-compute/schemas"
	"github.com/google/uuid"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strings"
)

type SubmissionTestResults struct {
	ID              string   `json:"_id"`
	TestTimeout     int      `json:"testTimeout"`
	TestMemoryLeaks bool     `json:"testMemoryLeaks"`
	TestExitCode    int      `json:"testExitCode"`
	TestOutputDiff  []string `json:"testOutputDiff"`
	TestErrorDiff   []string `json:"testErrorDiff"`
}
type CompilationResults struct {
	CompilationOutput   []string `json:"compilationOutput"`
}

type SubmissionResultSchema struct {
	SubmissionTestResults []SubmissionTestResults `json:"submissionTestResults"`
	CompilationResults    CompilationResults      `json:"compilationResults"`
}


func generateUUID() uuid.UUID {
	id, err := uuid.NewUUID()
	if err != nil {
		fmt.Println(err)
	}
	return id
}

func BuildWorkspace(submission schemas.SubmissionSchema) SubmissionResultSchema{
	id := generateUUID()

	result := SubmissionResultSchema{}

	path := fmt.Sprintf("./temp/%s", id)
	_ = os.MkdirAll(path, os.ModePerm)
	//defer os.RemoveAll(path)

	for _, file := range submission.SubmissionFiles {
		GenerateFile(path, file.FileName, file.FileReference)
	}

	compilationOutput := CompileSubmission(path, submission)
	result.CompilationResults = compilationOutput

	testPath := fmt.Sprintf("./temp/%s/tests", id)
	_ = os.MkdirAll(testPath, os.ModePerm)
	//defer os.RemoveAll(testPath)

	for _, test := range submission.SubmissionTests {
		GenerateFile(testPath, test.TestInput.FileName, test.TestInput.FileReference)
		GenerateFile(testPath, test.TestOutput.FileName, test.TestOutput.FileReference)
		GenerateFile(testPath, test.TestError.FileName, test.TestError.FileReference)
	}

	resultsPath := fmt.Sprintf("./temp/%s/results", id)
	_ = os.MkdirAll(resultsPath, os.ModePerm)

	RunTests(path, submission)

	result.SubmissionTestResults = GetDiffs(path, submission)

	return result

}

func CompileSubmission(path string, submission schemas.SubmissionSchema) CompilationResults {
	cmd := exec.Command(submission.CompilationOptions.CompilationCommand)
	cmd.Dir = path
	stdout, _ := cmd.Output()
	return CompilationResults{CompilationOutput: strings.Split(string(stdout), "\n")}
}

func GenerateFile(path string, fileName string, fileReference string) {
	filePath := fmt.Sprintf("%s/%s", path, fileName)
	fileLink := fmt.Sprintf("http://10.0.0.6:8333/csuchico/%s", fileReference)
	_ = downloadFile(filePath, fileLink)
}

func GetExecutable(path string) string {
	cmd := exec.Command("find", ".", "-perm", "+111", "-type", "f")
	cmd.Dir = path
	stdout, _ := cmd.Output()
	return string(stdout)
}

func getTestFile(path string, fileName string) []byte {
	bytes, err := ioutil.ReadFile(fmt.Sprintf("%s/tests/%s", path, fileName))
	if err != nil {
		log.Fatal(err)
	}
	return bytes
}

func RunTests(path string, submission schemas.SubmissionSchema) {
	executable := GetExecutable(path)
	executable = executable[:len(executable)-1] // Pesky \n
	for _, test := range submission.SubmissionTests {
		test := test
		go func() {
			cmd := exec.Command(executable)
			cmd.Dir = path

			buffer := bytes.Buffer{}
			buffer.Write(getTestFile(path, test.TestInput.FileName))
			cmd.Stdin = &buffer

			if len(test.TestOutput.FileName) > 0 {
				outfile, err := os.Create(fmt.Sprintf("%s/results/%s", path, test.TestOutput.FileName))
				if err != nil {
					log.Fatal(err)
				}
				defer outfile.Close()

				cmd.Stdout = outfile
			}

			if len(test.TestError.FileName) > 0 {
				errfile, err := os.Create(fmt.Sprintf("%s/results/%s", path, test.TestError.FileName))
				if err != nil {
					log.Fatal(err)
				}
				defer errfile.Close()

				cmd.Stderr = errfile
			}

			err := cmd.Start()
			if err != nil {
				log.Fatal(err)
			}

			cmd.Wait()
		}()
	}
}

func GetDiffs(path string, submission schemas.SubmissionSchema) []SubmissionTestResults {
	res := make([]SubmissionTestResults, len(submission.SubmissionTests))
	for _, test := range submission.SubmissionTests {
		results := SubmissionTestResults{
			ID:              test.ID,
			TestTimeout:     test.TestTimeout,
			TestMemoryLeaks: test.TestMemoryLeaks,
			TestExitCode:    test.TestExitCode,
			TestOutputDiff:  nil,
			TestErrorDiff:   nil,
		}
		if len(test.TestOutput.FileName) > 0 {
			pathOutput := fmt.Sprintf("%s/results/%s", path, test.TestOutput.FileName)
			pathOriginal := fmt.Sprintf("%s/tests/%s", path, test.TestOutput.FileName)
			results.TestOutputDiff = GetDiff(pathOutput, pathOriginal)
		}
		if len(test.TestError.FileName) > 0 {
			pathOutputError := fmt.Sprintf("%s/results/%s", path, test.TestError.FileName)
			pathOriginalError := fmt.Sprintf("%s/tests/%s", path, test.TestError.FileName)
			results.TestErrorDiff = GetDiff(pathOutputError, pathOriginalError)
		}
		res = append(res, results)
	}
	return res
}

func GetDiff(pathOutput string, pathOriginal string) []string {
	cmd := exec.Command("diff", pathOriginal, pathOutput)
	stdout, _ := cmd.Output()
	return strings.Split(string(stdout), "\n")
}
