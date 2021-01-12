package schemas

type CompileOptions struct {
	Timeout uint16 `json:"compilationTimeout"`
	Command string `json:"compilationCommand"`
}

type File struct {
	Name      string `json:"fileName"`
	Reference string `json:"fileReference"`
}

type Test struct {
	Id          uint16   `json:"_id"`
	Timeout     uint16   `json:"testTimeout"`
	MemoryLeaks bool     `json:"testMemoryLeaks"`
	Arguments   []string `json:"testArguments"`
	ExitCode    uint8    `json:"testExitCode"`
	Input       File     `json:"testInput"`
	Output      File     `json:"testOutput"`
	TestError   File     `json:"testError"`
}

type Submission struct {
	Files          []File         `json:"submissionFiles" binding:"required"`
	Tests          []Test         `json:"submissionTests" binding:"required"`
	CompileOptions CompileOptions `json:"compilationOptions" binding:"required"`
}

type SubmissionSchema struct {
	SubmissionFiles []struct {
		FileName      string `json:"fileName"`
		FileReference string `json:"fileReference"`
	} `json:"submissionFiles"`
	SubmissionTests []struct {
		ID              string   `json:"_id"`
		TestTimeout     int      `json:"testTimeout"`
		TestMemoryLeaks bool     `json:"testMemoryLeaks"`
		TestArguments   []string `json:"testArguments"`
		TestExitCode    int      `json:"testExitCode"`
		TestOutput      struct {
			FileName      string `json:"fileName"`
			FileReference string `json:"fileReference"`
		} `json:"testOutput"`
		TestInput struct {
			FileName      string `json:"fileName"`
			FileReference string `json:"fileReference"`
		} `json:"testInput"`
	} `json:"submissionTests"`
	CompilationOptions struct {
		CompilationTimeout int    `json:"compilationTimeout"`
		CompilationCommand string `json:"compilationCommand"`
	} `json:"compilationOptions"`
}
