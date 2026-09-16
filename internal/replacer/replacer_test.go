package replacer_test

import (
	"testing"

	"github.com/81beastie/ssr/internal/detector"
	"github.com/81beastie/ssr/internal/replacer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReplace_ShouldSubstituteSingleSecret_WithTypePlaceholder(t *testing.T) {
	text := "password=hunter2 and more text"

	findings := detector.New().Detect(text)
	require.Len(t, findings, 1)

	result := replacer.New().Replace(text, findings)

	assert.Equal(t, "password=<ssr password> and more text", result)
}

func TestReplace_ShouldNumberPlaceholders_WhenSeveralSecretsOfSameType(t *testing.T) {
	text := "TOKEN_A=alpha TOKEN_B=beta"

	findings := detector.New().Detect(text)
	require.Len(t, findings, 2)

	result := replacer.New().Replace(text, findings)

	assert.Equal(t, "TOKEN_A=<ssr token:1> TOKEN_B=<ssr token:2>", result)
}

func TestReplace_ShouldReusePlaceholder_WhenSameSecretValueAppearsTwice(t *testing.T) {
	secret := "ghp_16C7e42F292c6912E7710c838347Ae178B4a"
	text := "first " + secret + " second " + secret

	findings := detector.New().Detect(text)
	require.Len(t, findings, 2)

	result := replacer.New().Replace(text, findings)

	assert.Equal(t, "first <ssr token> second <ssr token>", result)
}

func TestReplace_ShouldNumberIndependently_WhenDifferentTypes(t *testing.T) {
	text := "password=hunter2 TOKEN_A=alpha"

	findings := detector.New().Detect(text)
	require.Len(t, findings, 2)

	result := replacer.New().Replace(text, findings)

	assert.Equal(t, "password=<ssr password> TOKEN_A=<ssr token>", result)
}

func TestReplace_ShouldReturnOriginalText_WhenNoFindings(t *testing.T) {
	text := "nothing to hide"

	result := replacer.New().Replace(text, nil)

	assert.Equal(t, "nothing to hide", result)
}

func TestReplace_ShouldKeepSurroundingText_WhenSecretInMiddle(t *testing.T) {
	jwt := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.dozjgNryP4J3jVmNHl0w5N_XgL0n3I9PlFUP0THsR8U"
	text := "curl -H 'Authorization: Bearer " + jwt + "' http://host"

	findings := detector.New().Detect(text)
	require.NotEmpty(t, findings)

	result := replacer.New().Replace(text, findings)

	assert.Equal(t, "curl -H 'Authorization: Bearer <ssr token>' http://host", result)
}
