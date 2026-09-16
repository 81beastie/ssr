package detector_test

import (
	"testing"

	"github.com/81beastie/ssr/internal/detector"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDetect_ShouldFindShadowHash_WhenYescryptHash(t *testing.T) {
	text := "root:$y$j9T$GvX8rQ2pWz1a$L.4Kq0mN7wE1cJ9vHkF3bZ0pQrS5tUvWxYz1234567:19000:0:99999:7:::"

	findings := detector.New().Detect(text)

	require.Len(t, findings, 1)
	assert.Equal(t, "password", findings[0].Type)
	assert.Contains(t, findings[0].Value, "$y$j9T$")
}

func TestDetect_ShouldFindShadowHash_WhenSHA512Hash(t *testing.T) {
	text := "admin:$6$rounds500$abcdefgh$Xk2mNpQrStUvWxYz0123456789AbCdEfGhIjKlMnOpQrSt:19000"

	findings := detector.New().Detect(text)

	require.Len(t, findings, 1)
	assert.Equal(t, "password", findings[0].Type)
	assert.Contains(t, findings[0].Value, "$6$")
}

func TestDetect_ShouldFindShadowHash_WhenBcryptHash(t *testing.T) {
	text := "svc:$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy:19000"

	findings := detector.New().Detect(text)

	require.Len(t, findings, 1)
	assert.Equal(t, "password", findings[0].Type)
	assert.Contains(t, findings[0].Value, "$2a$10$")
}

func TestDetect_ShouldFindToken_WhenLowercaseTokenAssignment(t *testing.T) {
	findings := detector.New().Detect("token=FFFDTRTR")

	require.Len(t, findings, 1)
	assert.Equal(t, "token", findings[0].Type)
	assert.Equal(t, "FFFDTRTR", findings[0].Value)
}

func TestDetect_ShouldFindToken_WhenTokenColonSeparator(t *testing.T) {
	findings := detector.New().Detect("token: abc-123-xyz")

	require.Len(t, findings, 1)
	assert.Equal(t, "token", findings[0].Type)
	assert.Equal(t, "abc-123-xyz", findings[0].Value)
}

func TestDetect_ShouldFindPassword_WhenPasswordAliases(t *testing.T) {
	for _, key := range []string{"pwd", "pass", "pswd", "passwd"} {
		findings := detector.New().Detect(key + ":jkfhsjdhfsdf")

		require.Len(t, findings, 1, "key %s", key)
		assert.Equal(t, "password", findings[0].Type, "key %s", key)
		assert.Equal(t, "jkfhsjdhfsdf", findings[0].Value, "key %s", key)
	}
}

func TestDetect_ShouldFindToken_WhenSecretKeyAssignment(t *testing.T) {
	findings := detector.New().Detect("client_secret=zzzTopSecret999")

	require.Len(t, findings, 1)
	assert.Equal(t, "token", findings[0].Type)
	assert.Equal(t, "zzzTopSecret999", findings[0].Value)
}

func TestDetect_ShouldFindToken_WhenNewGitHubPATFormat(t *testing.T) {
	prefix := "github" + "_pat_11FAKEFAKEFAKE0TEST_"
	suffix := "FAKEFAKEFAKEFAKEFAKEFAKEFAKEFAKEFAKEFAKEFAKEFAKEFAKEFAKEFAKE"
	pat := prefix + suffix

	findings := detector.New().Detect(pat)

	require.Len(t, findings, 1)
	assert.Equal(t, "token", findings[0].Type)
	assert.Equal(t, pat, findings[0].Value)
}

func TestDetect_ShouldFindPassword_WhenUserPassHostWithoutScheme(t *testing.T) {
	findings := detector.New().Detect("user1:shgdhagsd@host")

	require.Len(t, findings, 1)
	assert.Equal(t, "password", findings[0].Type)
	assert.Equal(t, "shgdhagsd", findings[0].Value)
}

func TestDetect_ShouldReturnNothing_WhenPlainEmail(t *testing.T) {
	findings := detector.New().Detect("write to user1@example.com about user1@jkfhsjdhfsdf")

	assert.Empty(t, findings)
}
