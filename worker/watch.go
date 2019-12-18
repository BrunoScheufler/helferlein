package worker

import (
	"bytes"
	"context"
	"fmt"
	"github.com/sirupsen/logrus"
	"gopkg.in/src-d/go-git.v4"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// Watch a list of repositories
func watchRepositories(ctx context.Context, config *Config, repositories []*git.Repository) error {
	// Parse fetch duration
	fetchInterval, err := time.ParseDuration(config.FetchInterval)
	if err != nil {
		return fmt.Errorf("could not parse fetch interval %q: %w", config.FetchInterval, err)
	}

	// Create cancellable context to prevent running operations when a shutdown is in progress
	cancellableCtx, cancel := context.WithCancel(ctx)
	errors := make(chan error, 0)
	done := make(chan int, 0)

	// Handle errors
	go func() {
		for {
			select {
			case err := <-errors:
				logrus.Errorf("Received watch error: %s", err.Error())

				// If context is already cancelled, return
				if cancellableCtx.Err() == context.Canceled {
					return
				}

				// Stop all watching
				cancel()

				// Close down
				done <- 1

				return
			}
		}
	}()

	// Run watcher for every repository
	for i := range repositories {
		go func(currentGitRepository *git.Repository, currentRepository Repository) {
			// Watch and send errors to channel
			err := watchRepository(cancellableCtx, watchRepositoryOptions{
				config:        config,
				repoConfig:    currentRepository,
				gitRepository: currentGitRepository,
				fetchInterval: fetchInterval,
			})
			if err != nil {
				errors <- err
			}
		}(repositories[i], config.Repositories[i])
	}

	logrus.Infof("Watching %d repositories", len(repositories))

	// Wait until the error handle closes the watching phase
	<-done

	return nil
}

type watchRepositoryOptions struct {
	config        *Config
	repoConfig    Repository
	gitRepository *git.Repository
	fetchInterval time.Duration
}

// Watch a repository
func watchRepository(ctx context.Context, options watchRepositoryOptions) error {
	// Retrieve git credentials from configuration
	gitCredentials, err := options.config.GetAuthCredentials()
	if err != nil {
		return fmt.Errorf("could not retrieve git credentials: %w", err)
	}

	for {
		logrus.Debugf("Fetching repository %q", options.repoConfig.Name)

		// Check if context was closed
		if ctx.Err() != nil {
			logrus.Debugf("Cancellable context for watching was closed, ending watcher for repository %q", options.repoConfig.Name)
			return nil
		}

		// Check for changes to branches
		for branchName, branchConfig := range options.repoConfig.Branches {
			// Pull changes from origin branchName into local branchName
			wt, err := options.gitRepository.Worktree()
			if err != nil {
				return fmt.Errorf("could not retrieve git work tree of repository %q: %w", options.repoConfig.Name, err)
			}

			branch, err := options.gitRepository.Branch(branchName)
			if err != nil {
				return fmt.Errorf("could not find branch for name %q in repository %q: %w", branchName, options.repoConfig.Name, err)
			}

			// Check out branchName
			err = wt.Checkout(&git.CheckoutOptions{
				Branch: branch.Merge,
				Force:  true,
			})
			if err != nil {
				return fmt.Errorf("could not check out branchName %q of repository %q: %w", branch, options.repoConfig.Name, err)
			}

			// Get head commit before pulling
			currentHead, err := options.gitRepository.Head()
			if err != nil {
				return fmt.Errorf("could not get current head for branch %q of repository %q: %w", branch, options.repoConfig.Name, err)
			}

			// Pull remote changes
			err = wt.PullContext(ctx, &git.PullOptions{
				RemoteName:    "origin",
				ReferenceName: branch.Merge,
				SingleBranch:  true,
				Auth:          gitCredentials,
			})
			if err != nil {
				// Continue with next branch if there are no fetch changes
				// We still need to compare the branch commit heads since there could be
				// changes to the repository like new branches or tags even though
				// the watched branch didn't update, in which case the error below won't be returned
				if err == git.NoErrAlreadyUpToDate {
					continue
				}

				return fmt.Errorf("could not pull fresh contents into branchName %q of repository %q: %w", branch, options.repoConfig.Name, err)
			}

			// Get head after the git merge run by the pull operation
			mergedHead, err := options.gitRepository.Head()
			if err != nil {
				return fmt.Errorf("could not get merged head for branch %q of repository %q: %w", branch, options.repoConfig.Name, err)
			}

			// Check if we really got new commits
			if currentHead.Hash().String() == mergedHead.Hash().String() {
				logrus.Debugf(
					"Skipping refresh triggers of branch %q of repository %q since previous and new head don't differ: %q (prev), %q (new)",
					branchName,
					options.repoConfig.Name,
					currentHead.Hash().String(),
					mergedHead.Hash().String(),
				)
				continue
			}

			logrus.Infof("Pulled branch %q of repository %q with new content", branchName, options.repoConfig.Name)

			// Execute branch steps
			for i, step := range branchConfig.Steps {
				logrus.Debugf("Running step %d of branch %q of repository %q", i+1, branchName, options.repoConfig.Name)

				// Create cmd
				stepCmd := strings.Split(step, " ")
				cmd := exec.Command(stepCmd[0], stepCmd[1:]...)

				// Set working directory to clone directory + repo name (e.g. .helferlein/<repo>)
				cmd.Dir = filepath.Join(options.config.CloneDirectory, options.repoConfig.Name)

				// Create stdout buffer and link to cmd
				output := bytes.Buffer{}
				cmd.Stdout = &output

				// Run command
				err = cmd.Run()
				if err != nil {
					return fmt.Errorf("could not run step %d of branch %q of repository %q: %w", i+1, branchName, options.repoConfig.Name, err)
				}

				logrus.Infof("Completed step %d of branch %q of repository %q: %s", i+1, branchName, options.repoConfig.Name, output.String())
			}
		}

		logrus.Debugf("Done refreshing repository %q", options.repoConfig.Name)

		// Wait a bit until the next fetch
		<-time.After(options.fetchInterval)
	}
}
