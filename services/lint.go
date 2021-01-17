package services

import (
	"context"
	"fmt"
	"github.com/bradenn/turnin-compute/schemas"
	"os/exec"
	"strings"
	"time"
)

func (res *SubmissionResultSchema) LintCode(path string, submission schemas.SubmissionSchema) error {
	cmd := exec.Command("grep", "'int main()'", "*.cpp")

	grep, _ := cmd.CombinedOutput()

	lintData := LintThisCode(path, strings.Split(string(grep), ":")[0])
	lintMap := make(map[string]SubmissionFileLint)
	for _, line := range lintData {
		tokens := strings.Split(line, ":")
		if tokens[0] == "cppcheck" {
			break
		}
		fmt.Println(tokens)
		fileRef := findFile(tokens[0], submission.SubmissionFiles)
		lint := lintMap[fileRef.ID]
		lint.FileID = fileRef.ID
		lint.LintLines = append(lint.LintLines, tokens[1:]...)
	}
	fmt.Println(lintMap)
	for a, b := range lintMap {
		fmt.Println(a)
		fmt.Println(b)
		res.SubmissionFileLint = append(res.SubmissionFileLint, b)
	}

	return nil
}

func findFile(name string, files []schemas.FileReference) schemas.FileReference {
	for _, file := range files {
		if name == file.FileName {
			return file
		}
	}
	return schemas.FileReference{}
}

func LintThisCode(path string, file string) []string {

	var ctx, cancel = context.WithTimeout(context.Background(), time.Millisecond*1000)

	cmd := exec.CommandContext(ctx, "/opt/homebrew/Cellar/cppcheck/2.3/bin/cppcheck", "--enable=all", "--inconclusive",
		"--library=posix", "--template='{file}:{line}:{column}:{severity}:{message}'", "--quiet", "-I", ".", file)
	defer cancel()
	cmd.Dir = path

	_ = cmd.Wait()

	mem, _ := cmd.CombinedOutput()
	fmt.Println(string(mem))
	return strings.Split(string(mem), "\n")
}
