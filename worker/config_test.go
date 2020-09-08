package worker

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestParse(t *testing.T) {
	exampleConfig := []byte(`clone_directory: ".helferlein"
projects:
  helferlein:
    fetch_interval: "10s"
    clone_url: "https://github.com/BrunoScheufler/helferlein.git"
    branches:
      main:
        steps:
          - echo "Hooray"`)

	cfg := &Config{}
	err := cfg.LoadFromBytes(exampleConfig)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, len(cfg.Projects), 1)

	project := cfg.Projects["helferlein"]
	assert.Equal(t, project.CloneUrl, "https://github.com/BrunoScheufler/helferlein.git")
	assert.Equal(t, len(project.Branches), 1)

	branch := project.Branches["main"]
	assert.Equal(t, len(branch.Steps), 1)

	assert.Equal(t, branch.Steps[0], `echo "Hooray"`)
}
