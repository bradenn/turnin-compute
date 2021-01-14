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
	"sync"
	"syscall"
)

type SubmissionTestResult struct {
	ID              string   `json:"_id"`
	BytesUsed       int      `json:"bytesUsed"`
	TestExitCode    int      `json:"testExitCode"`
	TestElapsedTime string   `json:"testElapsedTime"`
	TestStatus      string   `json:"testStatus"`
	TestOutputDiff  []string `json:"testOutputDiff"`
	TestErrorDiff   []string `json:"testErrorDiff"`
}

type CompilationResults struct {
	CompilationOutput []string `json:"compilationOutput"`
	CompilationTime   string   `json:"compilationTime"`
}

type SubmissionResultSchema struct {
	SubmissionTestResults []SubmissionTestResult `json:"submissionTestResults"`
	CompilationResults    CompilationResults     `json:"compilationResults"`
}

/*
	Function Definition:
	This function builds, compiles, runs, tests, and returns the difference of the aforementioned operations.
	returns SubmissionResultSchema & error
*/
func BuildAndCompileSubmission(submission schemas.SubmissionSchema) (*SubmissionResultSchema, error) {
	/* Generate a UUID for the workspace operations. */
	id := generateUUID()
	/* Instantiate a new Submission result Schema. */
	var result = new(SubmissionResultSchema)
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
	err = CompileSubmission(path, submission, result)
	if err != nil {
		log.Println("Could not compile workspace.")
		return result, err
	}
	/* Next, we need to verify that an executable exists */
	executable, err := GetExecutable(path)
	if err != nil {
		log.Println("Could not find the executable.")
		return result, err
	}
	/* At this point we have compiled the program, now we can run the tests. */
	err = RunTests(executable, path, submission, result)
	if err != nil {
		log.Println("Could not run tests.", err)
		return result, err
	}
	/* At this point, if the function hasn't errored out, we should have the fully tested program files.
	The next order of business is to get the differences and return them to the router for further processing. */
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
	/* This wait group basically works like async/await from js/ts... */
	var fileWg sync.WaitGroup
	fileWg.Add(len(submission.SubmissionFiles))
	/* Here we iterate through all of the provided files,
	these files are both the submission files and the provided files. */
	for _, file := range submission.SubmissionFiles {
		var err error = nil
		/* We are essentially making this a parallel for loop with these go routines */
		go func(file schemas.SubmissionFile, err error) {
			defer fileWg.Done()
			/* We call emplace file to download and put the fil into our workspace */
			err = EmplaceFile(path, file.FileName, file.FileReference)
		}(file, err)
		if err != nil {
			return err
		}
	}
	/* Again, we use a wait group to ensure the function doesn't proceed before all files are downloaded  */
	var testWg sync.WaitGroup
	testWg.Add(len(submission.SubmissionTests))
	/* This loop iterates through all of the submission test files. A submission can have one, two, three,
	or no test files. */
	for _, test := range submission.SubmissionTests {
		/* We are essentially making this a parallel for loop with these go routines */
		var err error = nil
		go func(test schemas.SubmissionTest, err error) {
			defer testWg.Done()
			/* If the input file exists, emplace the file and report any errors. */
			if len(test.TestInput.FileName) > 0 {
				err = EmplaceFile(testPath, test.TestInput.FileName, test.TestInput.FileReference)
			}
			/* If the output file exists, emplace the file and report any errors. */
			if len(test.TestOutput.FileName) > 0 {
				err = EmplaceFile(testPath, test.TestOutput.FileName, test.TestOutput.FileReference)
			}
			/* If the error file exists, emplace the file and report any errors. */
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

func CompileSubmission(path string, submission schemas.SubmissionSchema,
	result *SubmissionResultSchema) error {
	/* This is an oddly small function, very important nonetheless...
	We start by defining our command and exec reference. The command comes straight from the submission schema.
	We aren't using bash, so we should be safe from any funny business. */
	cmd := exec.Command(submission.CompilationOptions.CompilationCommand)
	cmd.Dir = path
	/* Using CombinedOutput, we group the stdout and stderr together so we only need on field. */
	stdout, err := cmd.CombinedOutput()
	result.CompilationResults.CompilationOutput = strings.Split(string(stdout), "\n")
	/* A few notes before we go */
	result.CompilationResults.CompilationTime = cmd.ProcessState.SystemTime().String()
	/* Finally we return the Result object */
	return err
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

func RunTests(executable string, path string, submission schemas.SubmissionSchema,
	result *SubmissionResultSchema) error {
	/* We start by allocating a portion of the heap to hold a predefined object instance. */
	var testWg sync.WaitGroup
	testWg.Add(len(submission.SubmissionTests))
	/* Iterate through all of the tests */
	for _, test := range submission.SubmissionTests {
		var err error = nil
		go func(res *SubmissionResultSchema, test schemas.SubmissionTest, err error) {
			defer testWg.Done()
			/* Run each tests and return errors as needed */
			analytic, err := ExecuteTest(executable, path, test)
			var session = SubmissionTestResult{
				ID:              test.ID,
				BytesUsed:       analytic.BytesUsed,
				TestElapsedTime: analytic.TestElapsedTime,
				TestExitCode:    analytic.TestExitCode,
			}
			/* If the test has an output file, compare with the produced output */
			if len(test.TestOutput.FileName) > 0 {
				pathOutput := fmt.Sprintf("%s/results/%s", path, test.TestOutput.FileName)
				pathOriginal := fmt.Sprintf("%s/tests/%s", path, test.TestOutput.FileName)
				session.TestOutputDiff = GetDiff(pathOutput, pathOriginal)
			}
			/* If the test has an error file, compare with the produced error */
			if len(test.TestError.FileName) > 0 {
				pathOutputError := fmt.Sprintf("%s/results/%s", path, test.TestError.FileName)
				pathOriginalError := fmt.Sprintf("%s/tests/%s", path, test.TestError.FileName)
				session.TestErrorDiff = GetDiff(pathOutputError, pathOriginalError)
			}
			/* Push the results to the main res array. */
			res.SubmissionTestResults = append(res.SubmissionTestResults, session)

		}(result, test, err)
		/* Push test analytics to the array */
	}
	testWg.Wait()
	/* Return the final results */
	return nil
}

func ExecuteTest(executable string, path string, test schemas.SubmissionTest) (SubmissionTestResult, error) {
	/* We can define a new schema here for what our return should look like.
	We are only interested in the exit code and the time for now. */
	var result = SubmissionTestResult{}
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
		return result, err
	}
	buffer.Write(writeStream)
	/* The buffer is piped into the StdIn, unlike how JS & TS pipe. */
	cmd.Stdin = &buffer
	/* If the provided test has a test output (we check the length of the file name to test),
	then we can create a file stream and pipe the Stdout to the new file. */
	if len(test.TestOutput.FileName) > 0 {
		outfile, err := os.Create(fmt.Sprintf("%s/results/%s", path, test.TestOutput.FileName))
		if err != nil {
			return result, err
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
			return result, err
		}
		/* Again we defer the closing until after the function returns (or just before) */
		defer errfile.Close()
		/* Again, piping the error to an output file. */
		cmd.Stderr = errfile
	}
	/* Now that we've defined our pipework, we can actually start the operation. */
	err = cmd.Start()
	if err != nil {
		return result, err
	}
	/* We wait on cmd to finish all operations and nt our pipes. The error will no be nil if the exit code is 1,
	since that is a normal part of operation, we will leave it here. */
	_ = cmd.Wait()
	/* Here we can assign the process state analytics to our return */
	result.TestElapsedTime = cmd.ProcessState.SystemTime().String()
	result.TestExitCode = cmd.ProcessState.ExitCode()
	result.BytesUsed = int(cmd.ProcessState.SysUsage().(*syscall.Rusage).Maxrss)
	/* If we have no errors, we return nil */
	return result, nil
}

func GetDiff(pathOutput string, pathOriginal string) []string {
	/* Another shortcut, here we just run diff as we would in any other capacity. */
	cmd := exec.Command("diff", pathOriginal, pathOutput)
	/* Again, we combine the stdout and stderr to make things more practical */
	stdout, _ := cmd.CombinedOutput()
	/* After it all we return a string array of the lines. */
	return strings.Split(string(stdout), "\n")
}
