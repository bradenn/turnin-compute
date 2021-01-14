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
	"syscall"
)

type SubmissionTestResults struct {
	ID             string   `json:"_id"`
	TestOutputDiff []string `json:"testOutputDiff"`
	TestErrorDiff  []string `json:"testErrorDiff"`
}

type SubmissionTestAnalytic struct {
	ID              string   `json:"_id"`
	TestMemoryLeaks bool     `json:"testMemoryLeaks"`
	TestExitCode    int      `json:"testExitCode"`
	TestMemory      []string `json:"testMemory"`
	TestElapsedTime string   `json:"testElapsedTime"`
	TestMemoryUsed  int64    `json:"testMemoryUsed"`
}

type CompilationResults struct {
	CompilationOutput []string `json:"compilationOutput"`
}

type SubmissionResultSchema struct {
	SubmissionTestResults   []SubmissionTestResults  `json:"submissionTestResults"`
	SubmissionTestAnalytics []SubmissionTestAnalytic `json:"submissionTestAnalytics"`
	CompilationResults      CompilationResults       `json:"compilationResults"`
}

/*
	Function Definition:
	This function builds, compiles, runs, tests, and returns the difference of the aforementioned operations.
	returns SubmissionResultSchema & error
*/
func BuildAndCompileSubmission(submission schemas.SubmissionSchema) (SubmissionResultSchema, error) {
	/* Generate a UUID for the workspace operations. */
	id := generateUUID()
	/* Instantiate a new Submission result Schema. */
	result := SubmissionResultSchema{}
	/* Allocate the workspace and log an error if needed. */
	path, testPath, _, err := AllocateWorkspace(id)
	if err != nil {
		log.Println("Could not allocate workspace.")
		return result, err
	}
	/* Given that we have allocated a significant chunk of space for testing this program,
	we use defer to run a function once this function exits. In this case, we just forcibly remove the directory. */
	defer os.RemoveAll(path)
	/* Download all files, tests, and required manifests to compile and test the program */
	err = BuildWorkspace(path, testPath, submission)
	if err != nil {
		log.Println("Could not build workspace.")
		return result, err
	}
	/* At this point, we should have all of the required files downloaded.
	So now we can go about compiling the program files. */
	compilationOutput, err := CompileSubmission(path, submission)
	if err != nil {
		log.Println("Could not compile workspace.")
		return result, err
	}
	/* Here we can push the StdOut from the compilation to the SubmissionResult Object from earlier */
	result.CompilationResults = compilationOutput
	/* Next, we need to verify that an executable exists */
	executable, err := GetExecutable(path)
	if err != nil {
		log.Println("Could not find the executable.")
		return result, err
	}
	/* At this point we have compiled the program, now we can run the tests. */
	analytics, err := RunTests(executable, path, submission.SubmissionTests)
	if err != nil {
		log.Println("Could not run tests.", err)
		return result, err
	}
	/* Now we push the analytics to the object */
	result.SubmissionTestAnalytics = analytics
	/* At this point, if the function hasn't errored out, we should have the fully tested program files.
	The next order of business is to get the differences and return them to the router for further processing. */
	testResults, err := GetDiffs(path, submission)
	if err != nil {
		log.Println("Could not process diffs.")
		return result, err
	}
	/* Once we've verified the integrity of the test results, we add it to the object. */
	result.SubmissionTestResults = testResults
	/* Now we are ready to return the object to the router. */
	return result, nil
}

func generateUUID() uuid.UUID {
	/* All we're doing here is generating a UUID */
	id, err := uuid.NewUUID()
	/* If for some reason we cannot generate a UUID, fatal exit */
	if err != nil {
		log.Fatalln(err)
	}
	/* Return the final UUID */
	return id
}

func AllocateWorkspace(id uuid.UUID) (string, string, string, error) {
	/* We start by defining a path and making a directory at that path. */
	path := fmt.Sprintf("./temp/%s", id)
	err := os.MkdirAll(path, os.ModePerm)
	/* os.ModePerm means that the folder will have recursive 777 permissions, this will come in handy later. */
	/* We do essentially the same thing except specify the new tests directory */
	testPath := fmt.Sprintf("./temp/%s/tests", id)
	err = os.MkdirAll(testPath, os.ModePerm)
	/* Again, we define the path and create the new results directory. */
	resultsPath := fmt.Sprintf("./temp/%s/results", id)
	err = os.MkdirAll(resultsPath, os.ModePerm)
	/* Then we return all of the paths and the error. I should be reactively returning the error on occurrence,
	but in this case, if the first one fails, they all fail. */
	return path, testPath, resultsPath, err
}

func BuildWorkspace(path string, testPath string, submission schemas.SubmissionSchema) error {
	/* Here we iterate through all of the provided files,
	these files are both the submission files and the provided files. */
	for _, file := range submission.SubmissionFiles {
		/* We call emplace file to download and put the fil into our workspace */
		err := EmplaceFile(path, file.FileName, file.FileReference)
		/* If we run into any errors, we return and exit the function for further error handling */
		if err != nil {
			return err
		}
	}
	/* This loop iterates through all of the submission test files. A submission can have one, two, three,
	or no test files. */
	for _, test := range submission.SubmissionTests {
		/* If the input file exists, emplace the file and report any errors. */
		if len(test.TestInput.FileName) > 0 {
			err := EmplaceFile(testPath, test.TestInput.FileName, test.TestInput.FileReference)
			if err != nil {
				return err
			}
		}
		/* If the output file exists, emplace the file and report any errors. */
		if len(test.TestOutput.FileName) > 0 {
			err := EmplaceFile(testPath, test.TestOutput.FileName, test.TestOutput.FileReference)
			if err != nil {
				return err
			}
		}
		/* If the error file exists, emplace the file and report any errors. */
		if len(test.TestError.FileName) > 0 {
			err := EmplaceFile(testPath, test.TestError.FileName, test.TestError.FileReference)
			if err != nil {
				return err
			}
		}
	}
	/* Finally, we can exit without errors. */
	return nil
}

func EmplaceFile(path string, fileName string, fileReference string) error {
	/* Here we are defining the target file path and the link to the s3 bucket */
	filePath := fmt.Sprintf("%s/%s", path, fileName)
	/* Using the config from /config/config.go,
	we use the values loaded in from dotenv with the actual data hidden in .env */
	fileLink := fmt.Sprintf("%s/%s/%s", os.Getenv("S3_ENDPOINT"),
		os.Getenv("S3_BUCKET"), fileReference)
	/* Alas we download the file from the server and add it to the workspace */
	err := DownloadFile(filePath, fileLink)
	/* Here we can return any errors induced by downloading the file */
	return err
}

func CompileSubmission(path string, submission schemas.SubmissionSchema) (CompilationResults, error) {
	/* This is an oddly small function, very important nonetheless...
	We start by defining our command and exec reference. The command comes straight from the submission schema.
	We aren't using bash, so we should be safe from any funny business. */
	cmd := exec.Command(submission.CompilationOptions.CompilationCommand)
	cmd.Dir = path
	/* Using CombinedOutput, we group the stdout and stderr together so we only need on field. */
	stdout, err := cmd.CombinedOutput()
	/* Finally we return the Result object */
	return CompilationResults{CompilationOutput: strings.Split(string(stdout), "\n")}, err
}

func GetExecutable(path string) (string, error) {
	var executable string
	/* We are finding the executable by checking it's permissions */
	cmd := exec.Command("find", ".", "-perm", "+111", "-type", "f")
	cmd.Dir = path
	/* This is a very quick and dirty way of finding the executable, but it works very nicely */
	stdout, err := cmd.Output()
	executable = string(stdout)
	/* Here we trim the final carriage return from the end of the returned executable. */
	executable = executable[:len(executable)-1]
	return executable, err
}

func getTestFileBuffer(path string, fileName string) ([]byte, error) {
	/* All we do here is aggregate the path and read in the file buffer as a byte stream. */
	res, err := ioutil.ReadFile(fmt.Sprintf("%s/tests/%s", path, fileName))
	if err != nil {
		return res, err
	}
	/* If all goes well, we return res. */
	return res, nil
}

func RunTests(executable string, path string, submissionTests []schemas.SubmissionTest) ([]SubmissionTestAnalytic,
	error) {
	/* We start by allocating a portion of the heap to hold a predefined object instance. */
	res := make([]SubmissionTestAnalytic, 0)
	/* Iterate through all of the tests */
	for _, test := range submissionTests {
		/* Run each tests and return errors as needed */
		analytic, err := ExecuteTest(executable, path, test)
		if err != nil {
			return res, err
		}
		/* Push test analytics to the array */
		res = append(res, analytic)
	}
	/* Return the final results */
	return res, nil
}

func ExecuteTest(executable string, path string, test schemas.SubmissionTest) (SubmissionTestAnalytic, error) {
	/* We can define a new schema here for what our return should look like.
	We are only interested in the exit code and the time for now. */
	res := SubmissionTestAnalytic{
		ID:              test.ID,
		TestExitCode:    0,
		TestElapsedTime: "",
	}
	/* Below is the run command, something along the lines of ./binary,
	but the test.TestArguments... converts any provided array of strings ['', '', ...] into a flattened
	list of commands '', '', ... */
	cmd := exec.Command(executable, test.TestArguments...)
	cmd.Dir = path
	/* This buffer holds reads the data from the provided .in files in the test folder.
	The StdIn data is no piped with redirects. Here we are piping through the code. */
	buffer := bytes.Buffer{}
	writeStream, err := getTestFileBuffer(path, test.TestInput.FileName)
	if err != nil {
		return res, nil
	}
	buffer.Write(writeStream)
	/* The buffer is piped into the StdIn, unlike how JS & TS pipe. */
	cmd.Stdin = &buffer
	/* If the provided test has a test output (we check the length of the file name to test),
	then we can create a file stream and pipe the Stdout to the new file. */
	if len(test.TestOutput.FileName) > 0 {
		outfile, err := os.Create(fmt.Sprintf("%s/results/%s", path, test.TestOutput.FileName))
		if err != nil {
			return res, err
		}
		/* We defer the operation of closing the stream until after the function exits */
		defer outfile.Close()
		/* Here we pipe the Stdout writer to our os file */
		cmd.Stdout = outfile
	}
	/* Similar to the above if statement, we are only testing if the file spec exists. */
	if len(test.TestError.FileName) > 0 {
		errfile, err := os.Create(fmt.Sprintf("%s/results/%s", path, test.TestError.FileName))
		if err != nil {
			return res, err
		}
		/* Again we defer the closing until after the function returns (or just before) */
		defer errfile.Close()
		/* Again, piping the error to an output file. */
		cmd.Stderr = errfile
	}
	/* Now that we've defined our pipework, we can actually start the operation. */
	err = cmd.Start()
	if err != nil {
		return res, err
	}
	/* We wait on cmd to finish all operations and nt our pipes. The error will no be nil if the exit code is 1,
	since that is a normal part of operation, we will leave it here. */
	_ = cmd.Wait()
	/* Here we can assign the process state analytics to our return */
	res.TestElapsedTime = cmd.ProcessState.SystemTime().String()
	res.TestExitCode = cmd.ProcessState.ExitCode()
	res.TestMemoryUsed = cmd.ProcessState.SysUsage().(*syscall.Rusage).Maxrss
	/* If we have no errors, we return nil */
	return res, nil
}

func GetDiffs(path string, submission schemas.SubmissionSchema) ([]SubmissionTestResults, error) {
	/* First we allocate an empty array for the results */
	res := make([]SubmissionTestResults, 0)
	/* We then iterate through all of the tests */
	for _, test := range submission.SubmissionTests {
		/* Each test has a default object, displayed below */
		results := SubmissionTestResults{
			ID:             test.ID,
			TestOutputDiff: nil,
			TestErrorDiff:  nil,
		}
		/* If the test has an output file, compare with the produced output */
		if len(test.TestOutput.FileName) > 0 {
			pathOutput := fmt.Sprintf("%s/results/%s", path, test.TestOutput.FileName)
			pathOriginal := fmt.Sprintf("%s/tests/%s", path, test.TestOutput.FileName)
			results.TestOutputDiff = GetDiff(pathOutput, pathOriginal)
		}
		/* If the test has an error file, compare with the produced error */
		if len(test.TestError.FileName) > 0 {
			pathOutputError := fmt.Sprintf("%s/results/%s", path, test.TestError.FileName)
			pathOriginalError := fmt.Sprintf("%s/tests/%s", path, test.TestError.FileName)
			results.TestErrorDiff = GetDiff(pathOutputError, pathOriginalError)
		}
		/* Push the results to the main res array. */
		res = append(res, results)
	}
	/* Return all results */
	return res, nil
}

func GetDiff(pathOutput string, pathOriginal string) []string {
	/* Another shortcut, here we just run diff as we would in any other capacity. */
	cmd := exec.Command("diff", pathOriginal, pathOutput)
	/* Again, we combine the stdout and stderr to make things more practical */
	stdout, _ := cmd.CombinedOutput()
	/* After it all we return a string array of the lines. */
	return strings.Split(string(stdout), "\n")
}
