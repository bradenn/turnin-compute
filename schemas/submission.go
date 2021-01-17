package schemas

type SubmissionSchema struct {
	SubmissionFiles    []FileReference    `json:"submissionFiles"`
	SubmissionTests    []SubmissionTest   `json:"submissionTests"`
	CompilationOptions CompilationOptions `json:"compilationOptions"`
}

type SubmissionTest struct {
	ID              string        `json:"_id"`
	TestName        string        `json:"TestName"`
	TestMemoryLeaks bool          `json:"testMemoryLeaks"`
	TestTimeout     int           `json:"testTimeout"`
	TestArguments   []string      `json:"testArguments"`
	TestInput       FileReference `json:"testInput"`
	TestOutput      FileReference `json:"testOutput"`
	TestError       FileReference `json:"testError"`
	TestExitCode    int           `json:"testExitCode"`
}

type FileReference struct {
	ID            string `json:"_id"`
	FileName      string `json:"fileName"`
	FileReference string `json:"fileReference"`
}

type CompilationOptions struct {
	CompilationCommand string `json:"compilationCommand"`
	CompilationTimeout int    `json:"compilationTimeout"`
}
