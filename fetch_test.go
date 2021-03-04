package gcpsecretfetch

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
)

const GCP_PROJECT = "ob-playground"

func TestInitializeConfig(t *testing.T) {

	type config struct {
		SECRET_IDENTIFIER   string
		FALLBACK_IDENTIFIER string
	}

	var cfg config

	err := InitializeConfig(&cfg, GCP_PROJECT, PRIORITIZE)
	assert.NoError(t, err)
	assert.Equal(t, cfg.FALLBACK_IDENTIFIER, "VALUE")
	assert.Equal(t, cfg.SECRET_IDENTIFIER, "SECRET_VALUE")

}

func TestMissingSecret(t *testing.T) {

	type config struct {
		MISSING_SECRET string
	}

	var cfg config

	err := InitializeConfig(&cfg, GCP_PROJECT, PRIORITIZE)
	assert.Error(t, err)

}

func TestInitializeConfigMissing(t *testing.T) {

	type config struct {
		SECRET_IDENTIFIER   string
		FALLBACK_IDENTIFIER string
	}

	var cfg config

	err := InitializeConfig(&cfg, GCP_PROJECT, DISABLE)
	assert.Error(t, err)

}

func TestInitializeConfigEnvPrioritize(t *testing.T) {

	type config struct {
		BOTH_IDENTIFIER string
	}

	var cfg config

	err := InitializeConfig(&cfg, GCP_PROJECT, PRIORITIZE)
	assert.NoError(t, err)
	assert.Equal(t, cfg.BOTH_IDENTIFIER, "ENV")

	err = InitializeConfig(&cfg, GCP_PROJECT, FALLBACK)
	assert.NoError(t, err)
	assert.Equal(t, cfg.BOTH_IDENTIFIER, "GCP")

}

func TestSetSecrets(t *testing.T) {

	err := UpdateSecrets(GCP_PROJECT, map[string]string{"SECRET_IDENTIFIER": "SECRET_VALUE", "BOTH_IDENTIFIER": "GCP"}, true)
	assert.NoError(t, err)

}

func TestBadProjectUpdate(t *testing.T) {

	err := UpdateSecrets("bad-project-name-alkdjwopiunhauwihd", map[string]string{"SECRET_IDENTIFIER": "SECRET_VALUE", "BOTH_IDENTIFIER": "GCP"}, true)
	assert.Error(t, err)

}

func TestBadProjectInitialize(t *testing.T) {

	type config struct {
		BOTH_IDENTIFIER string
	}

	var cfg config

	err := InitializeConfig(&cfg, "aawdwadawdawdwaawdawdawdawd", DISABLE)
	assert.Error(t, err)
	fmt.Println(cfg)
}

func TestNotAPointer(t *testing.T) {

	type config struct {
		BOTH_IDENTIFIER string
	}
	var cfg config
	err := InitializeConfig(cfg, "bad-project-name-alkdjwopiunhauwihd", FALLBACK)
	assert.Error(t, err)

}

func TestNotAStruct(t *testing.T) {

	err := InitializeConfig("cfg", "bad-project-name-alkdjwopiunhauwihd", FALLBACK)
	assert.Error(t, err)

}

func TestStructWithNonStringField(t *testing.T) {

	type config struct {
		BOTH_IDENTIFIER string
		INTFIELD        int
	}
	var cfg config
	err := InitializeConfig(cfg, "bad-project-name-alkdjwopiunhauwihd", FALLBACK)
	assert.Error(t, err)

}
