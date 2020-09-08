package worker

import (
	"context"
	"fmt"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/sirupsen/logrus"
	"os"
	"time"
)

func configureAuth(projectConfig ProjectConfig) transport.AuthMethod {
	var accessTokenOrPassword string

	// We prioritize access tokens, with environment variable fallback
	accessTokenOrPassword = os.Getenv("HELFERLEIN_GIT_AUTH_ACCESS_TOKEN")
	if projectConfig.Auth.AccessToken != "" {
		accessTokenOrPassword = projectConfig.Auth.AccessToken
	}

	// If no access token was set, check for regular passwords
	if accessTokenOrPassword == "" {
		accessTokenOrPassword = os.Getenv("HELFERLEIN_GIT_AUTH_PASSWORD")
		if projectConfig.Auth.Password != "" {
			accessTokenOrPassword = projectConfig.Auth.Password
		}
	}

	// If no access token or password was specified, don't add any auth
	if accessTokenOrPassword == "" {
		return nil
	}

	var username string

	username = os.Getenv("HELFERLEIN_GIT_AUTH_USER")
	if projectConfig.Auth.User != "" {
		username = projectConfig.Auth.User
	}

	if username == "" {
		username = "helferlein"
	}

	auth := &http.BasicAuth{
		Username: username,
		Password: accessTokenOrPassword,
	}

	return auth
}

// Clone a repository
func cloneProjectRepository(ctx context.Context, projectName string, projectConfig ProjectConfig, branchName string, cloneTargetDir string) (*git.Repository, error) {
	// Create timed context to limit time we want to spend on cloning
	cloneContext, _ := context.WithTimeout(ctx, time.Minute*5)

	// Clone repository
	clonedRepo, err := git.PlainCloneContext(cloneContext, cloneTargetDir, false, &git.CloneOptions{
		URL:           projectConfig.CloneUrl,
		Auth:          configureAuth(projectConfig),
		RemoteName:    "origin",
		SingleBranch:  true,
		ReferenceName: plumbing.NewBranchReferenceName(branchName),
	})
	if err != nil {
		return nil, fmt.Errorf("could not clone repository for project %q: %w", projectName, err)
	}

	logrus.Infof("Successfully cloned repository of project %q", projectName)

	return clonedRepo, nil
}
