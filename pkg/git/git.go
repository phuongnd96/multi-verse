package gitClient

import (
	"os"

	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
)

// clone a repository into a specific folder.
func Clone(wd string, gitCommitHash string, username string, token string, url string) error {
	_, err := git.PlainClone(wd, false, &git.CloneOptions{
		Auth: &http.BasicAuth{
			Username: username, // yes, this can be anything except an empty string
			Password: token,
		},
		URL:      url,
		Progress: os.Stdout,
	})
	if err != nil {
		return err
	}
	return nil
}
