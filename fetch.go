// Package gcpsecretfetch is a utility library for getting secrets from GCP Secret Manager.
package gcpsecretfetch

import (
	"fmt"
	"github.com/panjf2000/ants/v2"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
	secretmanagerpb "google.golang.org/genproto/googleapis/cloud/secretmanager/v1"
	"strings"

	"reflect"
	"sync"
)

// InitializeConfig initializes a config struct by getting the secrets from GCP Secret Manager
//
// This function works by reflecting on the fields in the passed pointer struct, and
// then dispatching calls to get the secret payloads from GCP for the corresponding secrets.
// The function must be passed a pointer to an arbitrary config struct, and
// the config struct must only have string fields.
func InitializeConfig(cfg interface{}, project string, opts ...ConfigOption) error {

	s, err := validateStruct(cfg)
	if err != nil {
		return err
	}

	client, err := newClient(project, opts)
	if err != nil {
		return err
	}
	defer client.gcpClient.Close()

	return client.fetch(s)
}

func (svc *secretClient) accessSecretVersion(name string) ([]byte, error) {
	req := &secretmanagerpb.AccessSecretVersionRequest{
		Name: fmt.Sprintf("projects/%s/secrets/%s/versions/latest", svc.project, name),
	}

	secret, err := svc.gcpClient.AccessSecretVersion(svc.ctx, req)
	if err != nil {
		return nil, err
	}

	return secret.Payload.Data, nil
}

type params struct {
	v     *reflect.Value
	name  string
	viper *viper.Viper
}

func validateStruct(cfg interface{}) (reflect.Value, error) {
	t := reflect.TypeOf(cfg)
	var s reflect.Value

	if t.Kind() != reflect.Ptr {
		return s, errors.New("cfg argument must be a pointer to a struct, got:" + t.Kind().String())
	}

	s = reflect.ValueOf(cfg).Elem()
	if s.Kind() != reflect.Struct {
		return s, errors.New("cfg argument must be a pointer to a struct, got: " + s.Kind().String())
	}
	for i := 0; i < s.NumField(); i++ {

		if err := validateField(s, i); err != nil {
			return s, err
		}
	}

	return s, nil
}

func validateField(s reflect.Value, i int) error {
	f := s.Field(i)
	name := s.Type().Field(i).Name

	if !f.IsValid() || !f.CanSet() {
		return errors.New(fmt.Sprintf("field %s is not valid - check if field is value and that it is exported from struct", name))
	}

	if f.Kind() != reflect.String {
		return errors.New(fmt.Sprintf("pointer struct can only contain string fields - field '%s' is of type '%s'", name, f.Type().Name()))
	}

	return nil
}

func (svc *secretClient) fetch(s reflect.Value) error {
	var wg sync.WaitGroup

	type errorStruct struct {
		err  error
		name string
	}

	c := make(chan errorStruct, s.NumField())

	p, _ := ants.NewPoolWithFunc(svc.concurrency, func(i interface{}) {
		defer wg.Done()
		p := i.(params)

		err := svc.setValueFromGcp(p)
		if err == nil {
			return
		}

		c <- errorStruct{err, p.name}

	})

	defer ants.Release()

	for i := 0; i < s.NumField(); i++ {
		f := s.Field(i)
		name := s.Type().Field(i).Name

		wg.Add(1)
		err := p.Invoke(params{v: &f, name: name, viper: svc.v})
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

var mux sync.Mutex

func (svc *secretClient) setValueFromGcp(p params) error {
	secretString, err := svc.accessSecretVersion(strings.ToUpper(p.name))
	if err != nil {
		return err
	}

	if p.viper != nil {
		mux.Lock()
		p.viper.Set(p.name, secretString)
		mux.Unlock()
	}

	p.v.SetString(string(secretString))
	return nil
}
