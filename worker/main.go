package worker

import (
	"context"
	"crypto/sha1"
	"errors"
	"fmt"
	"github.com/go-git/go-git/v5"
	"github.com/sirupsen/logrus"
	"os"
	"path/filepath"
	"time"
)

type ProjectRepository struct {
	GitRepository *git.Repository
	LocalPath     string
}

type Project struct {
	Name         string
	Config       ProjectConfig
	Repositories map[string]*ProjectRepository
}

func Start(ctx context.Context, config *Config, logger *logrus.Logger) error {
	logger.Infoln("Setting up configured repositories...")

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

	// Keep track of projects
	projects := make([]*Project, 0)
	cloneStart := time.Now()

	// Clone non-existing project repositories
	for projectName, projectConfig := range config.Projects {
		repos := make(map[string]*ProjectRepository, len(projectConfig.Branches))

		for branchName := range projectConfig.Branches {
			nameHash := sha1.New()
			nameHash.Write([]byte(projectName))
			nameHash.Write([]byte(branchName))
			generatedPath := fmt.Sprintf("%x", nameHash.Sum(nil))

			// Clone into clone directory + repository name (e.g. .helferlein/<generated name>)
			cloneTargetDir := filepath.Join(config.CloneDirectory, generatedPath)

			// Try to open repository, otherwise clone
			localRepo, err := git.PlainOpen(cloneTargetDir)
			if err != nil {
				// If repository doesn't exist, clone it from the remote
				if errors.Is(err, git.ErrRepositoryNotExists) {
					// Clone repository
					clonedRepo, err := cloneProjectRepository(ctx, projectName, projectConfig, branchName, cloneTargetDir)
					if err != nil {
						return fmt.Errorf("could not clone repository for branch %q of project %q: %w", branchName, projectName, err)
					}

					repos[branchName] = &ProjectRepository{
						GitRepository: clonedRepo,
						LocalPath:     cloneTargetDir,
					}

					continue
				} else {
					return fmt.Errorf("could not open local repository for project: %q: %w", projectName, err)
				}
			}

			repos[branchName] = &ProjectRepository{
				GitRepository: localRepo,
				LocalPath:     cloneTargetDir,
			}
		}

		projects = append(projects, &Project{
			Name:         projectName,
			Config:       projectConfig,
			Repositories: repos,
		})
	}

	logger.Infof("Done cloning repositories in %s", time.Since(cloneStart).String())

	// Watch for changes
	err = watchProjects(ctx, projects, logger)
	if err != nil {
		return fmt.Errorf("could not watch repositories: %w", err)
	}

	return nil
}
