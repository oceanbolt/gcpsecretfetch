// Package gcpsecretfetch is a utility library for getting secrets from GCP Secret Manager.
package gcpsecretfetch

import (
	"fmt"
	"github.com/joho/godotenv"
	"github.com/panjf2000/ants/v2"
	"github.com/pkg/errors"
	secretmanagerpb "google.golang.org/genproto/googleapis/cloud/secretmanager/v1"
	"log"
	"os"
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
func InitializeConfig(cfg interface{}, project string, envFileAction EnvFileAction) error {

	grabber, err := newClient(project)
	defer grabber.client.Close()

	if err != nil {
		return err
	}
	t := reflect.TypeOf(cfg)

	if t.Kind() != reflect.Ptr {
		return errors.New("cfg argument must be a pointer to a struct")
	}

	s := reflect.ValueOf(cfg).Elem()
	if s.Kind() != reflect.Struct {
		return errors.New("cfg argument must be a pointer to a struct")
	}

	var wg sync.WaitGroup

	type errorStruct struct {
		err  error
		name string
	}

	if envFileAction != DISABLE {
		err := godotenv.Load()
		if err != nil {
			//return errors.New("Error loading .env file. If you dont want to use .env files, then specify `DISABLE` as envFileAction.")
			log.Print("Error loading .env file. If you dont want to use .env files, then specify `DISABLE` as envFileAction.")
		}
	}

	c := make(chan errorStruct, s.NumField())

	p, _ := ants.NewPoolWithFunc(10, func(i interface{}) {
		defer wg.Done()
		p := i.(params)

		secret := os.Getenv(p.name)

		if envFileAction == PRIORITIZE && secret != "" {
			grabber.setValueLiteral(p, secret)
			return
		}

		err := grabber.setValueFromGcp(p)
		if err == nil {
			return
		}

		if envFileAction == FALLBACK && secret != "" {
			grabber.setValueLiteral(p, secret)
			return
		}

		c <- errorStruct{err, p.name}

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
		if err.err != nil {
			return err.err
		}
	}

	return nil
}

func (svc *secretClient) setValueFromGcp(p params) error {
	secretString, err := svc.accessSecretVersion(p.name)
	if err != nil {
		return err
	}

	p.v.SetString(secretString)
	return nil
}

func (svc *secretClient) setValueLiteral(p params, secretString string) {
	p.v.SetString(secretString)
}

type EnvFileAction string

const (
	PRIORITIZE EnvFileAction = "prioritize"
	DISABLE    EnvFileAction = "disable"
	FALLBACK   EnvFileAction = "fallback"
)
