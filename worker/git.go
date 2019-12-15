package worker

import (
	"context"
	"fmt"
	"github.com/sirupsen/logrus"
	"gopkg.in/src-d/go-git.v4"
	"time"
)

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
