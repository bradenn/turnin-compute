package enclave

import (
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
)

// Some day I would like to migrate the git FS to in-stream. Ram is expensive though.

// The Repository struct contains all of the relevant information required to evaluate and maintain a repository file
// system.
type Repository struct {
	URL     string
	Commit  string
	Enclave *Enclave

	repo     *git.Repository
	workTree *git.Worktree
}

// Instantiate a new Repository reference within an enclave
func NewRepository(e *Enclave) (r *Repository) {
	r = &Repository{
		Enclave: e,
	}
	return r
}

// CloneRepository will clone the default branch of the repository from the url found in Repository.URL.
// The repository will be cloned into the the provided Enclave reference.
func (r *Repository) CloneRepository(e *Enclave) (err error) {
	// Clone the repository containing the submission files
	r.repo, err = git.PlainClone(e.Cwd, false, &git.CloneOptions{
		URL: r.URL,
	})

	if err != nil {
		return err
	}

	r.workTree, err = r.repo.Worktree()
	if err != nil {
		return err
	}

	return nil
}

// CheckoutCommit can be used to select an earlier version of the submission if desired. By default,
// the latest commit will be used.
func (r *Repository) CheckoutCommit(commit string) (err error) {
	r.Commit = commit
	err = r.workTree.Checkout(&git.CheckoutOptions{
		Hash: plumbing.NewHash(r.Commit),
	})
	if err != nil {
		return err
	}
	return nil
}
