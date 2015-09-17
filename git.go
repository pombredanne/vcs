package vcs

import (
	"os"
	"os/exec"
	"strings"
)

// NewGitRepo creates a new instance of GitRepo. The remote and local directories
// need to be passed in.
func NewGitRepo(remote, local string) (*GitRepo, error) {
	ltype, err := DetectVcsFromFS(local)

	// Found a VCS other than Git. Need to report an error.
	if err == nil && ltype != Git {
		return nil, ErrWrongVCS
	}

	r := &GitRepo{}
	r.setRemote(remote)
	r.setLocalPath(local)
	r.RemoteLocation = "origin"
	r.Logger = Logger

	// Make sure the local Git repo is configured the same as the remote when
	// A remote value was passed in.
	if err == nil && r.CheckLocal() == true {
		oldDir, err := os.Getwd()
		if err != nil {
			return nil, err
		}
		os.Chdir(local)
		defer os.Chdir(oldDir)
		out, err := exec.Command("git", "config", "--get", "remote.origin.url").CombinedOutput()
		if err != nil {
			return nil, err
		}

		localRemote := strings.TrimSpace(string(out))
		if remote != "" && localRemote != remote {
			return nil, ErrWrongRemote
		}

		// If no remote was passed in but one is configured for the locally
		// checked out Git repo use that one.
		if remote == "" && localRemote != "" {
			r.setRemote(localRemote)
		}
	}

	return r, nil
}

// GitRepo implements the Repo interface for the Git source control.
type GitRepo struct {
	base
	RemoteLocation string
}

// Vcs retrieves the underlying VCS being implemented.
func (s GitRepo) Vcs() Type {
	return Git
}

// Get is used to perform an initial clone of a repository.
func (s *GitRepo) Get() error {
	_, err := s.run("git", "clone", s.Remote(), s.LocalPath())
	return err
}

// Update performs an Git fetch and pull to an existing checkout.
func (s *GitRepo) Update() error {
	// Perform a fetch to make sure everything is up to date.
	_, err := s.runFromDir("git", "fetch", s.RemoteLocation)
	if err != nil {
		return err
	}
	_, err = s.runFromDir("git", "pull")
	return err
}

// UpdateVersion sets the version of a package currently checked out via Git.
func (s *GitRepo) UpdateVersion(version string) error {
	_, err := s.runFromDir("git", "checkout", version)
	return err
}

// Version retrieves the current version.
func (s *GitRepo) Version() (string, error) {
	out, err := s.runFromDir("git", "rev-parse", "HEAD")
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(out)), nil
}

// Branches returns a list of available branches on the RemoteLocation
func (s *GitRepo) Branches() ([]string, error) {
	out, err := s.runFromDir("git", "show-ref")
	if err != nil {
		return []string{}, err
	}
	branches := s.referenceList(string(out), `(?m-s)(?:`+s.RemoteLocation+`)/(\S+)$`)
	return branches, nil
}

// Tags returns a list of available tags on the RemoteLocation
func (s *GitRepo) Tags() ([]string, error) {
	out, err := s.runFromDir("git", "show-ref")
	if err != nil {
		return []string{}, err
	}
	tags := s.referenceList(string(out), `(?m-s)(?:tags)/(\S+)$`)
	return tags, nil
}

// CheckLocal verifies the local location is a Git repo.
func (s *GitRepo) CheckLocal() bool {
	if _, err := os.Stat(s.LocalPath() + "/.git"); err == nil {
		return true
	}

	return false

}
