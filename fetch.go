// Package gcpsecretfetch is a utility library for getting secrets from GCP Secret Manager.
package gcpsecretfetch

import (
	"fmt"
	"github.com/panjf2000/ants/v2"
	"github.com/pkg/errors"
	secretmanagerpb "google.golang.org/genproto/googleapis/cloud/secretmanager/v1"
	"reflect"
	"sync"
)

func (svc *secretClient) accessSecretVersion(name string) (string, error) {
	req := &secretmanagerpb.AccessSecretVersionRequest{
		Name: fmt.Sprintf("projects/%s/secrets/%s/versions/latest", svc.project, name),
	}

	secret, err := svc.client.AccessSecretVersion(svc.ctx, req)
	if err != nil {
		return "", err
	}
	s := string(secret.Payload.Data)
	return s, nil
}

type params struct {
	v    *reflect.Value
	name string
}

// InitializeConfig initializes a config struct by getting the secrets from GCP Secret Manager
//
// This function works by reflecting on the fields in the passed pointer struct, and
// then dispatching calls to get the secret payloads from GCP for the corresponding secrets.
// The function must be passed a pointer to an arbitrary config struct, and
// the config struct must only have string fields.
func InitializeConfig(cfg interface{}, project string) error {

	grabber, err := newClient(project)
	if err != nil {
		return err
	}

	t := reflect.TypeOf(cfg)
	if t.Kind() == reflect.Struct {
		return errors.New("cfg argument must be a pointer to a struct")
	}

	s := reflect.ValueOf(cfg).Elem()
	if s.Kind() != reflect.Struct {
		return errors.New("cfg argument must be a pointer to a struct")
	}

	var wg sync.WaitGroup

	c := make(chan error, s.NumField())

	p, _ := ants.NewPoolWithFunc(10, func(i interface{}) {
		p := i.(params)
		err := grabber.setValue(p)

		if err != nil {
			c <- err
		}
		wg.Done()
	})
	defer ants.Release()

	for i := 0; i < s.NumField(); i++ {

		f := s.Field(i)
		name := s.Type().Field(i).Name

		if !f.IsValid() || !f.CanSet() {
			return errors.New(fmt.Sprintf("field %s is not valid - check if field is value and that it is exported from struct", name))
		}

		if f.Kind() != reflect.String {
			return errors.New(fmt.Sprintf("pointer struct can only contain string fields - field '%s' is of type '%s'", name, f.Type().Name()))
		}
		wg.Add(1)
		err := p.Invoke(params{&f, name})
		if err != nil {
			return err
		}

	}

	wg.Wait()

	close(c)

	for err := range c {
		if err != nil {
			return err
		}
	}

	return nil
}

func (svc *secretClient) setValue(p params) error {
	secretString, err := svc.accessSecretVersion(p.name)
	if err != nil {
		return err
	}

	p.v.SetString(secretString)
	return nil
}
