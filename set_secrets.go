// Package gcpsecretfetch is a utility library for getting secrets from GCP Secret Manager.
package gcpsecretfetch

import (
	"fmt"
	"github.com/panjf2000/ants/v2"
	"github.com/pkg/errors"
	secretmanagerpb "google.golang.org/genproto/googleapis/cloud/secretmanager/v1"
	"strings"
	"sync"
)

func UpdateSecrets(project string, secrets map[string]string, opts ...ConfigOption) error {
	grabber, err := newClient(project, opts)
	if err != nil {
		return err
	}

	return grabber.set(secrets)
}

func (svc *secretClient) set(secrets map[string]string) error {
	var wg sync.WaitGroup

	type errorStruct struct {
		err  error
		name string
	}

	type params struct {
		secretName  string
		secretValue string
	}

	c := make(chan errorStruct, len(secrets))

	p, _ := ants.NewPoolWithFunc(svc.concurrency, func(i interface{}) {
		defer wg.Done()
		p := i.(params)

		err := svc.updateSingleVersion(p.secretName, p.secretValue, svc.disablePrior)
		if err == nil {
			return
		}
		c <- errorStruct{err, p.secretName}

	})

	defer p.Release()

	for k, v := range secrets {
		wg.Add(1)
		err := p.Invoke(params{secretName: k, secretValue: v})
		if err != nil {
			return err
		}
	}

	wg.Wait()

	close(c)

	var errs []string
	for e := range c {
		errs = append(errs, e.err.Error())
	}

	if len(errs) != 0 {
		return errors.New("Error when fetching secrets: " + strings.Join(errs, ", "))
	}

	return nil
}

func (svc *secretClient) updateSingleVersion(secretName string, secretValue string, disablePrior bool) error {
	versions, err := svc.listVersions(secretName)
	if err != nil {
		return err
	}
	latest, err := svc.accessSecretVersion(secretName)
	if string(latest) == secretValue {
		return nil
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
	version, err := svc.gcpClient.AddSecretVersion(svc.ctx, addSecretVersionReq)
	if err != nil {
		return nil, errors.Wrap(err, "failed to add secret version: "+addSecretVersionReq.String())
	}
	return version, nil
}

func (svc *secretClient) getSecret(name string) (*secretmanagerpb.Secret, error) {
	getSecretReq := &secretmanagerpb.GetSecretRequest{
		Name: fmt.Sprintf("projects/%s/secrets/%s", svc.project, name),
	}

	secret, err := svc.gcpClient.GetSecret(svc.ctx, getSecretReq)
	if err != nil {
		return nil, err
	}
	return secret, nil
}

func (svc *secretClient) getLatestSecretVersion(name string) (*secretmanagerpb.SecretVersion, error) {
	req := &secretmanagerpb.GetSecretVersionRequest{
		Name: fmt.Sprintf("projects/%s/secrets/%s/versions/latest", svc.project, name),
	}

	secret, err := svc.gcpClient.GetSecretVersion(svc.ctx, req)
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

	versions := svc.gcpClient.ListSecretVersions(svc.ctx, req)

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
		if v.State == secretmanagerpb.SecretVersion_DESTROYED {
			continue
		}
		req := &secretmanagerpb.DestroySecretVersionRequest{Name: v.Name}
		_, err := svc.gcpClient.DestroySecretVersion(svc.ctx, req)
		if err != nil {
			return err
		}
	}
	return nil
}
