package worker

import (
	"context"
	"errors"
	"fmt"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/sirupsen/logrus"
	"os"
	"os/exec"
	"sync"
	"time"
)

// Watch a list of projects
func watchProjects(ctx context.Context, projects []*Project, logger *logrus.Logger) error {
	wg := &sync.WaitGroup{}

	// Run watcher for every project
	for _, project := range projects {
		wg.Add(1)
		go watchProject(ctx, wg, logger, project)
	}

	logger.Infof("Watching configured projects")

	wg.Wait()

	logger.Debugln("Watching ended for all projects")

	return nil
}

// Watch a repository
func watchProject(ctx context.Context, wg *sync.WaitGroup, logger *logrus.Logger, project *Project) {
	defer wg.Done()

	branchWg := &sync.WaitGroup{}

	for branchName, branchConfig := range project.Config.Branches {
		repository := project.Repositories[branchName]
		branchWg.Add(1)
		go branchWatcher(ctx, branchWg, logger, repository, branchName, branchConfig, project)
	}

	logger.Infof("Waiting for changes to branches of project %q", project.Name)

	branchWg.Wait()

	logger.Debugf("Watching ended for project %q", project.Name)
}

func branchWatcher(ctx context.Context, wg *sync.WaitGroup, logger *logrus.Logger, repository *ProjectRepository, branchName string, branchConfig BranchConfig, project *Project) {
	defer wg.Done()

	for {
		if ctx.Err() != nil {
			break
		}

		err := watchProjectBranch(ctx, logger, repository, branchName, branchConfig, project)
		if err != nil {
			logger.WithError(err).Errorf("Failed to watch branch %q of project %q", branchName, project.Name)
		}

		select {
		case <-ctx.Done():
			return
		case <-time.After(project.Config.FetchInterval):
			break
		}
	}
}

func watchProjectBranch(ctx context.Context, logger *logrus.Logger, repository *ProjectRepository, branchName string, branchConfig BranchConfig, project *Project) error {
	gitRepo := repository.GitRepository

	// Get head commit before pulling
	currentHead, err := gitRepo.Head()
	if err != nil {
		return fmt.Errorf("could not get current head for branch %q: %w", branchName, err)
	}

	wt, err := gitRepo.Worktree()
	if err != nil {
		return fmt.Errorf("could not get work tree for branch %q: %w", branchName, err)
	}

	// Pull remote changes
	err = wt.PullContext(ctx, &git.PullOptions{
		RemoteName:    "origin",
		ReferenceName: plumbing.NewBranchReferenceName(branchName),
		SingleBranch:  true,
		Auth:          configureAuth(project.Config),
	})
	if err != nil {
		// Continue with next branch if there are no fetch changes
		// We still need to compare the branch commit heads since there could be
		// changes to the repository like new branches or tags even though
		// the watched branch didn't update, in which case the error below won't be returned
		if err == git.NoErrAlreadyUpToDate {
			return nil
		}

		return fmt.Errorf("could not pull fresh contents into branchName %q: %w", branchName, err)
	}

	// Get head after the git merge run by the pull operation
	mergedHead, err := gitRepo.Head()
	if err != nil {
		return fmt.Errorf("could not get merged head for branch %q: %w", branchName, err)
	}

	previousCommit := currentHead.Hash().String()
	nextCommit := mergedHead.Hash().String()

	// Check if we really got new commits
	if previousCommit == nextCommit {
		logger.Debugf(
			"Skipping refresh triggers of branch %q since previous and new head don't differ: %q (prev), %q (new)",
			branchName,
			previousCommit,
			nextCommit,
		)
		return nil
	}

	commonLogFields := logrus.Fields{
		"Commit":  nextCommit,
		"Branch":  branchName,
		"Project": project.Name,
	}

	progressLogger := logger.WithFields(commonLogFields)

	progressLogger.Infof("Pulled changes for branch %q", branchName)

	return runSteps(ctx, progressLogger, repository, branchName, branchConfig, wt)
}

func runSteps(ctx context.Context, progressLogger *logrus.Entry, repository *ProjectRepository, branchName string, branchConfig BranchConfig, wt *git.Worktree) error {
	start := time.Now()

	// Execute branch steps
	for i, step := range branchConfig.Steps {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		progressLogger.Debugf("Running step %d of %d", i+1, len(branchConfig.Steps))

		shouldContinue, err := runCommand(repository.LocalPath, step)

		if !shouldContinue {
			// Perform hard reset to make sure any changes performed are undone
			resetError := wt.Reset(&git.ResetOptions{Mode: git.HardReset})
			if resetError != nil {
				progressLogger.Errorf("failed to perform hard reset: %s", resetError.Error())
			}
		}

		if err != nil {
			return fmt.Errorf("could not run step %d of branch %q: %w", i+1, branchName, err)
		}

		if !shouldContinue {
			break
		}

		progressLogger.Infof("Completed step %d of %d", i+1, len(branchConfig.Steps))
	}

	progressLogger.Infof("Done syncing branch in %s (%s)", time.Since(start).String(), time.Now().Format(time.RFC3339))

	return nil
}

func runCommand(workDir string, command string) (bool, error) {
	// Create cmd
	cmd := exec.Command("bash", "-c", command)

	// Set working directory to clone directory + repo name (e.g. .helferlein/<repo>)
	cmd.Dir = workDir

	// Pipe stdout and stderr to helferlein's output
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Run command
	err := cmd.Run()
	if err != nil {
		var exitError *exec.ExitError
		if !errors.As(err, &exitError) {
			return false, fmt.Errorf("could not run command: %w", err)
		}

		return false, nil
	}

	return true, nil
}
