package gcpsecretfetch

import (
	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	"context"
	"github.com/pkg/errors"
)

type secretClient struct {
	client  *secretmanager.Client
	project string
	ctx     context.Context
}

func newClient(project string) (*secretClient, error) {
	ctx := context.Background()
	client, err := secretmanager.NewClient(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "could not create secretmanager client")
	}

	grabber := secretClient{
		client:  client,
		project: project,
		ctx:     ctx,
	}

	return &grabber, nil

}
