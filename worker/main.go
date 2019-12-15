package worker

import (
	"context"
	"fmt"
	"github.com/sirupsen/logrus"
	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/config"
	"os"
	"path/filepath"
	"time"
)

func Start(config *Config) error {
	logrus.Infoln("Setting up configured repositories...")

	ctx := context.Background()

	_, err := os.Stat(config.CloneDirectory)
	if err != nil {
		// Create directory if it doesn't exist
		if os.IsNotExist(err) {
			err := os.MkdirAll(config.CloneDirectory, os.ModePerm)
			if err != nil {
				return fmt.Errorf("could not create clone directory: %w", err)
			}
		} else {
			return fmt.Errorf("could not find or create clone directory: %w", err)
		}
	}

	// Keep track of cloned repositories
	clonedRepos := make([]*git.Repository, len(config.Repositories))

	// Clone non-existing repositories
	for i, repository := range config.Repositories {
		cloneTargetDir := filepath.Join(config.CloneDirectory, repository.Name)

		// Try to open repository, otherwise clone
		localRepo, err := git.PlainOpen(cloneTargetDir)
		if err != nil {
			if err == git.ErrRepositoryNotExists {
				// Clone repository
				clonedRepo, err := cloneRepository(ctx, repository, config, cloneTargetDir)
				if err != nil {
					return err
				}

				clonedRepos[i] = clonedRepo
				continue
			} else {

				return fmt.Errorf("could not open local repository: %q: %w", repository.Name, err)
			}
		}

		clonedRepos[i] = localRepo

	}

	logrus.Infoln("Done cloning repositories")

	// Watch for changes
	err = watchRepositories(ctx, config, clonedRepos)
	if err != nil {
		return fmt.Errorf("could not watch repositories: %w", err)
	}

	return nil
}

func cloneRepository(ctx context.Context, repository Repository, config *Config, cloneTargetDir string) (*git.Repository, error) {
	cloneContext, _ := context.WithTimeout(ctx, time.Minute*5)

	credentials, err := config.GetAuthCredentials()
	if err != nil {
		return nil, fmt.Errorf("could not retrieve git auth credentials from config: %w", err)
	}

	// Clone repository
	clonedRepo, err := git.PlainCloneContext(cloneContext, cloneTargetDir, false, &git.CloneOptions{
		URL:        repository.CloneUrl,
		Auth:       credentials,
		RemoteName: "origin",
	})
	if err != nil {
		return nil, fmt.Errorf("could not clone repository %q: %w", repository.Name, err)
	}

	logrus.Infof("Successfully cloned repository %q", repository.Name)

	return clonedRepo, nil
}

func watchRepositories(ctx context.Context, config *Config, repositories []*git.Repository) error {
	// Parse fetch duration
	fetchInterval, err := time.ParseDuration(config.FetchInterval)
	if err != nil {
		return fmt.Errorf("could not parse fetch interval %q: %w", config.FetchInterval, err)
	}

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

	for i, _ := range repositories {
		go func(currentGitRepository *git.Repository, currentRepository Repository) {
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

func watchRepository(ctx context.Context, options watchRepositoryOptions) error {
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

		// Fetch repository
		err := options.gitRepository.Fetch(&git.FetchOptions{
			RefSpecs: []config.RefSpec{
				// Fetches all remote references to origin/<ref> -> master will be fetched to origin/master
				"+refs/heads/*:refs/remotes/origin/*",
			},
			Auth:       gitCredentials,
			RemoteName: "origin",
		})
		if err != nil {
			// If no new changes exist, continue instantly
			if err == git.NoErrAlreadyUpToDate {
				logrus.Debugf("Nothing to fetch for repository %q", options.repoConfig.Name)
				<-time.After(options.fetchInterval)
				continue
			}

			// If the error should be handled, quit watching
			return fmt.Errorf("could not fetch repository %q: %w", options.repoConfig.Name, err)
		}

		logrus.Debugf("Found new contents for repository %q", options.repoConfig.Name)

		// Check for changes to branches
		for _, branchName := range options.repoConfig.Branches {
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

			// Pull remote changes
			err = wt.PullContext(ctx, &git.PullOptions{
				RemoteName:    "origin",
				ReferenceName: branch.Merge,
				SingleBranch:  true,
				Auth:          gitCredentials,
			})
			if err != nil {
				// Continue with next branchName if there are no changes
				if err == git.NoErrAlreadyUpToDate {
					continue
				}

				return fmt.Errorf("could not pull fresh contents into branchName %q of repository %q: %w", branch, options.repoConfig.Name, err)
			}

			logrus.Infoln("pulled new changes")
		}

		logrus.Infof("Done with repository %q", options.repoConfig.Name)

		// Wait a bit until the next fetch
		<-time.After(options.fetchInterval)
	}
}
