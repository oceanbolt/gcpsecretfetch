// Package gcpsecretfetch is a utility library for getting secrets from GCP Secret Manager.
package gcpsecretfetch

import (
	"fmt"
	"github.com/pkg/errors"
	secretmanagerpb "google.golang.org/genproto/googleapis/cloud/secretmanager/v1"
)

func UpdateSecrets(project string, secrets map[string]string, disablePrior bool) error {
	grabber, err := newClient(project)
	if err != nil {
		return err
	}
	for k, v := range secrets {
		if err := grabber.updateSingleVersion(k, v, disablePrior); err != nil {
			return err
		}
	}
	return nil
}

func (svc *secretClient) updateSingleVersion(secretName string, secretValue string, disablePrior bool) error {
	versions, err := svc.listVersions(secretName)
	if err != nil {
		return err
	}
	_, err = svc.addVersion(secretName, secretValue)
	if err != nil {
		return err
	}

	if disablePrior {
		return svc.deletePrior(versions)
	}
	return nil
}

func (svc *secretClient) addVersion(name string, value string) (*secretmanagerpb.SecretVersion, error) {

	// Declare the payload to store.
	payload := []byte(value)

	// Build the request.

	addSecretVersionReq := &secretmanagerpb.AddSecretVersionRequest{
		Parent: fmt.Sprintf("projects/%s/secrets/%s", svc.project, name),
		Payload: &secretmanagerpb.SecretPayload{
			Data: payload,
		},
	}

	// Call the API.
	version, err := svc.client.AddSecretVersion(svc.ctx, addSecretVersionReq)
	if err != nil {
		return nil, errors.Wrap(err, "failed to add secret version: "+addSecretVersionReq.String())
	}
	return version, nil
}

func (svc *secretClient) getSecret(name string) (*secretmanagerpb.Secret, error) {
	getSecretReq := &secretmanagerpb.GetSecretRequest{
		Name: fmt.Sprintf("projects/%s/secrets/%s", svc.project, name),
	}

	secret, err := svc.client.GetSecret(svc.ctx, getSecretReq)
	if err != nil {
		return nil, err
	}
	return secret, nil
}

func (svc *secretClient) getLatestSecretVersion(name string) (*secretmanagerpb.SecretVersion, error) {
	req := &secretmanagerpb.GetSecretVersionRequest{
		Name: fmt.Sprintf("projects/%s/secrets/%s/versions/latest", svc.project, name),
	}

	secret, err := svc.client.GetSecretVersion(svc.ctx, req)
	if err != nil {
		return nil, err
	}
	return secret, nil
}

func (svc *secretClient) listVersions(name string) ([]*secretmanagerpb.SecretVersion, error) {
	req := &secretmanagerpb.ListSecretVersionsRequest{
		PageSize: 1000,
		Parent:   fmt.Sprintf("projects/%s/secrets/%s", svc.project, name),
	}

	versions := svc.client.ListSecretVersions(svc.ctx, req)

	var output []*secretmanagerpb.SecretVersion

	for {
		if version, err := versions.Next(); err != nil {
			if err.Error() == "no more items in iterator" {
				break
			} else {
				return nil, err
			}
		} else {
			output = append(output, version)
		}
	}

	return output, nil
}

func (svc *secretClient) deletePrior(versions []*secretmanagerpb.SecretVersion) error {
	for _, v := range versions {
		req := &secretmanagerpb.DestroySecretVersionRequest{Name: v.Name}
		_, err := svc.client.DestroySecretVersion(svc.ctx, req)
		if err != nil {
			return err
		}
	}
	return nil
}
