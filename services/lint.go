package services

import (
	"context"
	"fmt"
	"github.com/bradenn/turnin-compute/schemas"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

func (res *SubmissionResultSchema) LintCode(path string, submission schemas.SubmissionSchema) error {
	absPath, _ := filepath.Abs(path)
	cmd := exec.Command("grep", "-r", "int main()", "-H", ".")
	cmd.Dir = absPath

	grep, _ := cmd.CombinedOutput()

	// [2:] basically removes ./ from the exec output ex: ./exec -> exec
	lintData := LintThisCode(path, strings.Split(string(grep), ":")[0][2:])
	lintMap := make(map[string]SubmissionFileLint)
	for _, line := range lintData {
		tokens := strings.Split(line, ":")
		if tokens[0] == "nofile" {
			break
		}
		fileRef := findFile(tokens[0], submission.SubmissionFiles)
		message := strings.Join(tokens[1:], ":")
		lintMap[fileRef.ID] = SubmissionFileLint{
			FileID:    fileRef.ID,
			LintLines: append(lintMap[fileRef.ID].LintLines, message),
		}
	}
	for _, b := range lintMap {
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

	absPath, _ := filepath.Abs(path)
	cmd := exec.CommandContext(ctx, "bash", "-c", fmt.Sprintf("/opt/homebrew/Cellar/cppcheck/2."+
		"3/bin/cppcheck --enable=all --inconclusive --library=posix --std=c++11 --quiet --template='{file}:{line"+
		"}:{column}:{severity}:{message}' --language=c++ --cppcheck-build-dir=%s *.cpp *.h", absPath))
	defer cancel()
	cmd.Dir = absPath

	_ = cmd.Wait()

	mem, _ := cmd.CombinedOutput()
	return strings.Split(string(mem), "\n")
}
