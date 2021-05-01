package worker

import (
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

func TestPassingCommand(t *testing.T) {
	someWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	shouldContinue, err := runCommand(someWd, `exit 0`)
	assert.Equal(t, nil, err, "expected err to be nil")
	assert.Equal(t, true, shouldContinue, "expected to continue after passing command")
}

func TestFailingCommand(t *testing.T) {
	someWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	shouldContinue, err := runCommand(someWd, "exit 1")
	assert.Equal(t, nil, err, "expected err to be nil")
	assert.Equal(t, false, shouldContinue, "expected not to continue after failing command")
}
