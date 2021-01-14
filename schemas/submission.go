package schemas

type SubmissionFile struct {
	FileName      string `json:"fileName"`
	FileReference string `json:"fileReference"`
}

type SubmissionTest struct {
	ID              string   `json:"_id"`
	TestTimeout     int      `json:"testTimeout"`
	TestMemoryLeaks bool     `json:"testMemoryLeaks"`
	TestArguments   []string `json:"testArguments"`
	TestExitCode    int      `json:"testExitCode"`
	TestInput       struct {
		FileName      string `json:"fileName"`
		FileReference string `json:"fileReference"`
	} `json:"testInput"`
	TestOutput struct {
		FileName      string `json:"fileName"`
		FileReference string `json:"fileReference"`
	} `json:"testOutput"`
	TestError struct {
		FileName      string `json:"fileName"`
		FileReference string `json:"fileReference"`
	} `json:"testError"`
}

type CompilationOptions struct {
	CompilationTimeout int    `json:"compilationTimeout"`
	CompilationCommand string `json:"compilationCommand"`
}

type SubmissionSchema struct {
	SubmissionFiles    []SubmissionFile   `json:"submissionFiles"`
	SubmissionTests    []SubmissionTest   `json:"submissionTests"`
	CompilationOptions CompilationOptions `json:"compilationOptions"`
}
