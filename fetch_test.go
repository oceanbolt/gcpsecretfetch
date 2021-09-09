package gcpsecretfetch

import (
	"fmt"
	"github.com/joho/godotenv"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"log"
	"os"
	"reflect"
	"testing"
	"time"
)

const GCP_PROJECT_ENV_KEY = "GCP_PROJECT_ID"

var GCP_PROJECT string

func init() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	GCP_PROJECT = os.Getenv(GCP_PROJECT_ENV_KEY)
	if GCP_PROJECT == "" {
		log.Fatal(GCP_PROJECT_ENV_KEY + " env var must be defined")
	}
}

func TestBadProjectUpdate(t *testing.T) {
	err := UpdateSecrets("bad-project-name-alkdjwopiunhauwihd", map[string]string{"SECRET_IDENTIFIER": "SECRET_VALUE", "BOTH_IDENTIFIER": "GCP"}, WithDisablePrior())
	assert.Error(t, err)
}

func TestBadProjectInitialize(t *testing.T) {

	type config struct {
		BOTH_IDENTIFIER string
	}

	var cfg config

	err := InitializeConfig(&cfg, "aawdwadawdawdwaawdawdawdawd")
	assert.Error(t, err)
	fmt.Println(cfg)
}

func TestNotAPointer(t *testing.T) {

	type config struct {
		BOTH_IDENTIFIER string
	}
	var cfg config
	err := InitializeConfig(cfg, "bad-project-name-alkdjwopiunhauwihd")
	assert.Error(t, err)

}

func TestNotAStruct(t *testing.T) {
	err := InitializeConfig("cfg", "bad-project-name-alkdjwopiunhauwihd")
	assert.Error(t, err)
}
func TestNotAStructPointer(t *testing.T) {
	cfg := "cfg"
	err := InitializeConfig(&cfg, "bad-project-name-alkdjwopiunhauwihd")
	assert.Error(t, err)
}

func TestStructWithNonStringField(t *testing.T) {

	type config struct {
		BOTH_IDENTIFIER string
		INTFIELD        int
	}
	var cfg config
	err := InitializeConfig(&cfg, "bad-project-name-alkdjwopiunhauwihd")
	assert.Error(t, err)

}

func TestViperCase(t *testing.T) {
	v := viper.New()

	type config struct {
		BOTH_IDENTIFIER string
	}

	var cfg config

	err := InitializeConfig(&cfg, GCP_PROJECT, WithViper(v), WithConcurrency(1))
	assert.NoError(t, err)
	assert.NotEmpty(t, cfg.BOTH_IDENTIFIER)
	assert.Equal(t, v.GetString("both_identifier"), cfg.BOTH_IDENTIFIER)

}

func TestViperCaseMissingVar(t *testing.T) {
	v := viper.New()

	type config struct {
		MissingSecret string
	}

	var cfg config

	err := InitializeConfig(&cfg, GCP_PROJECT, WithViper(v))
	assert.Error(t, err)
	assert.Empty(t, cfg.MissingSecret)

}

func isEmptyValue(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Array, reflect.Map, reflect.Slice, reflect.String:
		return v.Len() == 0
	case reflect.Bool:
		return !v.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return v.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return v.Float() == 0
	case reflect.Interface, reflect.Ptr:
		return v.IsNil()
	case reflect.Struct:
		if t, ok := v.Interface().(time.Time); ok {
			return t.IsZero()
		}
	}
	return false
}
