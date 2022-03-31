# gcpsecretfetch

Utility library for getting secrets from GCP Secret Manager.
The library works by accepting a pointer to a cfg struct that contains only string values, and then it populates the struct values with values from GCP for the corresponding secrets.

## Example

```go
type config struct {
		MY_SECRET_NAME string
	}

var cfg config

err := InitializeConfig(&cfg, "my-gcp-project")
```

## Viper support
The library also supports setting the values of the config struct directly in a `viper.Viper` instance.
```go
v := viper.New()

type config struct {
		MY_SECRET_NAME string
	}

var cfg config

err := InitializeConfig(&cfg, "my-gcp-project", WithViper(v))
```

## Extract
