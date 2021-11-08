package gcpsecretfetch

import (
	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	"context"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
	"google.golang.org/api/option"
)

const defaultConcurrency = 50

type secretClient struct {
	gcpClient    *secretmanager.Client
	project      string
	ctx          context.Context
	v            *viper.Viper
	concurrency  int
	disablePrior bool
}

type ConfigOption func(client *secretClient)

func WithViper(v *viper.Viper) ConfigOption {
	return func(client *secretClient) {
		client.v = v
	}
}

func WithConcurrency(n int) ConfigOption {
	return func(client *secretClient) {
		client.concurrency = n
	}
}

func WithDisablePrior() ConfigOption {
	return func(client *secretClient) {
		client.disablePrior = true
	}
}

func newClient(project string, opts []ConfigOption) (*secretClient, error) {
	ctx := context.Background()
	gcpClient, err := secretmanager.NewClient(ctx, option.WithScopes("https://www.googleapis.com/auth/cloud-platform"))
	if err != nil {
		return nil, errors.Wrap(err, "could not create secretmanager client")
	}

	svc := &secretClient{
		gcpClient:   gcpClient,
		project:     project,
		ctx:         ctx,
		concurrency: defaultConcurrency,
	}

	// Loop through each option
	if opts != nil {
		for _, opt := range opts {
			// Call the option giving the instantiated
			// *House as the argument
			opt(svc)
		}
	}

	return svc, nil
}
