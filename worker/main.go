package worker

import (
	"context"
	"fmt"
	"github.com/sirupsen/logrus"
	"gopkg.in/src-d/go-git.v4"
	"os"
	"path/filepath"
	"time"
)

func Start(config *Config) error {
	logrus.Infoln("Setting up configured repositories...")

	ctx := context.Background()

	// Create clone directory if it doesn't exist
	_, err := os.Stat(config.CloneDirectory)
	if err != nil {
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
	cloneStart := time.Now()

	// Clone non-existing repositories
	for i, repository := range config.Repositories {
		cloneTargetDir := filepath.Join(config.CloneDirectory, repository.Name)

		// Try to open repository, otherwise clone
		localRepo, err := git.PlainOpen(cloneTargetDir)
		if err != nil {
			// If repository doesn't exist, clone it from the remote
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

		// Add local repo to active repository list
		clonedRepos[i] = localRepo
	}

	logrus.Infof("Done cloning repositories in %s", time.Since(cloneStart).String())

	// Watch for changes
	err = watchRepositories(ctx, config, clonedRepos)
	if err != nil {
		return fmt.Errorf("could not watch repositories: %w", err)
	}

	return nil
}
