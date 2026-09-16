package detector_test

import (
	"testing"

	"github.com/81beastie/ssr/internal/detector"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDetect_ShouldFindJWT_WhenTokenInText(t *testing.T) {
	jwt := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.dozjgNryP4J3jVmNHl0w5N_XgL0n3I9PlFUP0THsR8U"
	text := "config uses " + jwt + " for auth"

	findings := detector.New().Detect(text)

	require.Len(t, findings, 1)
	assert.Equal(t, "token", findings[0].Type)
	assert.Equal(t, jwt, findings[0].Value)
	assert.Equal(t, 12, findings[0].Start)
	assert.Equal(t, 12+len(jwt), findings[0].End)
}

func TestDetect_ShouldFindBearerToken_WhenAuthorizationHeader(t *testing.T) {
	text := "curl -H 'Authorization: Bearer abc123def456ghi789' http://host"

	findings := detector.New().Detect(text)

	require.Len(t, findings, 1)
	assert.Equal(t, "token", findings[0].Type)
	assert.Equal(t, "abc123def456ghi789", findings[0].Value)
}

func TestDetect_ShouldFindGitHubToken_WhenGhpPrefix(t *testing.T) {
	text := "export GITHUB_TOKEN=ghp_16C7e42F292c6912E7710c838347Ae178B4a"

	findings := detector.New().Detect(text)

	require.Len(t, findings, 1)
	assert.Equal(t, "token", findings[0].Type)
	assert.Equal(t, "ghp_16C7e42F292c6912E7710c838347Ae178B4a", findings[0].Value)
}

func TestDetect_ShouldFindAWSAccessKey_WhenAkiaPrefix(t *testing.T) {
	text := "aws_access_key_id = AKIAIOSFODNN7EXAMPLE"

	findings := detector.New().Detect(text)

	require.Len(t, findings, 1)
	assert.Equal(t, "token", findings[0].Type)
	assert.Equal(t, "AKIAIOSFODNN7EXAMPLE", findings[0].Value)
}

func TestDetect_ShouldFindPassword_WhenPasswordAssignment(t *testing.T) {
	text := "password=hunter2"

	findings := detector.New().Detect(text)

	require.Len(t, findings, 1)
	assert.Equal(t, "password", findings[0].Type)
	assert.Equal(t, "hunter2", findings[0].Value)
}

func TestDetect_ShouldFindToken_WhenEnvStyleAssignment(t *testing.T) {
	text := "API_TOKEN=abc123secret"

	findings := detector.New().Detect(text)

	require.Len(t, findings, 1)
	assert.Equal(t, "token", findings[0].Type)
	assert.Equal(t, "abc123secret", findings[0].Value)
}

func TestDetect_ShouldFindPassword_WhenUserPassInURL(t *testing.T) {
	text := "postgres://admin:s3cret@localhost:5432/db"

	findings := detector.New().Detect(text)

	require.Len(t, findings, 1)
	assert.Equal(t, "password", findings[0].Type)
	assert.Equal(t, "s3cret", findings[0].Value)
}

func TestDetect_ShouldReturnNothing_WhenNoSecrets(t *testing.T) {
	text := "just a plain text with no secrets at all"

	findings := detector.New().Detect(text)

	assert.Empty(t, findings)
}
