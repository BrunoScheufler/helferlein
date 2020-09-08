package worker

import (
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestWithPasswordAuth(t *testing.T) {
	auth := configureAuth(ProjectConfig{
		Auth: GitAuthConfig{
			User:     "user",
			Password: "password",
		},
	})

	assert.Equal(t, auth, &http.BasicAuth{
		Username: "user",
		Password: "password",
	})
}

func TestWithTokenAuth(t *testing.T) {
	auth := configureAuth(ProjectConfig{
		Auth: GitAuthConfig{
			AccessToken: "mytoken",
		},
	})

	assert.Equal(t, auth, &http.BasicAuth{
		Username: "helferlein",
		Password: "mytoken",
	})
}

func TestWithoutAuth(t *testing.T) {
	auth := configureAuth(ProjectConfig{})

	assert.Equal(t, auth, nil)
}
