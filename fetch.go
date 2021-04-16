// Package gcpsecretfetch is a utility library for getting secrets from GCP Secret Manager.
package gcpsecretfetch

import (
	"fmt"
	"github.com/joho/godotenv"
	"github.com/panjf2000/ants/v2"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
	secretmanagerpb "google.golang.org/genproto/googleapis/cloud/secretmanager/v1"
	"gopkg.in/yaml.v2"
	"log"
	"os"

	"reflect"
	"sync"
)

func (svc *secretClient) accessSecretVersion(name string) ([]byte, error) {
	req := &secretmanagerpb.AccessSecretVersionRequest{
		Name: fmt.Sprintf("projects/%s/secrets/%s/versions/latest", svc.project, name),
	}

	secret, err := svc.client.AccessSecretVersion(svc.ctx, req)
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

// InitializeConfig initializes a config struct by getting the secrets from GCP Secret Manager
//
// This function works by reflecting on the fields in the passed pointer struct, and
// then dispatching calls to get the secret payloads from GCP for the corresponding secrets.
// The function must be passed a pointer to an arbitrary config struct, and
// the config struct must only have string fields.
func InitializeConfigYaml(cfg interface{}, project string, secret string) error {

	grabber, err := newClient(project)
	if err != nil {
		return err
	}
	defer grabber.client.Close()

	t := reflect.TypeOf(cfg)

	if t.Kind() != reflect.Ptr {
		return errors.New("cfg argument must be a pointer to a struct")
	}

	s := reflect.ValueOf(cfg).Elem()
	if s.Kind() != reflect.Struct {
		return errors.New("cfg argument must be a pointer to a struct")
	}

	bytes, err := grabber.accessSecretVersion(secret)
	if err != nil {
		return err
	}

	err = yaml.Unmarshal(bytes, cfg)
	if err != nil {
		return err
	}

	return nil
}

func InitializeConfigViper(cfg interface{}, project string, viperP *viper.Viper) (error, []error) {

	grabber, err := newClient(project)
	if err != nil {
		return err, nil
	}
	defer grabber.client.Close()

	t := reflect.TypeOf(cfg)

	if t.Kind() != reflect.Ptr {
		return errors.New("cfg argument must be a pointer to a struct"), nil
	}

	s := reflect.ValueOf(cfg).Elem()
	if s.Kind() != reflect.Struct {
		return errors.New("cfg argument must be a pointer to a struct"), nil
	}

	var wg sync.WaitGroup

	type errorStruct struct {
		err  error
		name string
	}

	c := make(chan errorStruct, s.NumField())

	p, _ := ants.NewPoolWithFunc(10, func(i interface{}) {
		defer wg.Done()
		p := i.(params)

		err := grabber.setValueFromGcp(p)
		if err == nil {
			return
		}

		c <- errorStruct{err, p.name}

	})
	defer ants.Release()

	for i := 0; i < s.NumField(); i++ {

		f := s.Field(i)
		name := s.Type().Field(i).Name

		if !f.IsValid() || !f.CanSet() {
			return errors.New(fmt.Sprintf("field %s is not valid - check if field is value and that it is exported from struct", name)), nil
		}

		if f.Kind() != reflect.String {
			return errors.New(fmt.Sprintf("pointer struct can only contain string fields - field '%s' is of type '%s'", name, f.Type().Name())), nil
		}
		wg.Add(1)
		err := p.Invoke(params{v: &f, name: name, viper: viperP})
		if err != nil {
			return err, nil
		}

	}

	wg.Wait()

	close(c)

	var errs []error
	for err := range c {
		errs = append(errs, err.err)
	}

	return err, errs
}

func InitializeConfig(cfg interface{}, project string, envFileAction EnvFileAction) error {

	grabber, err := newClient(project)
	if err != nil {
		return err
	}
	defer grabber.client.Close()

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
		err := p.Invoke(params{v: &f, name: name})
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

var mux sync.Mutex

func (svc *secretClient) setValueFromGcp(p params) error {
	secretString, err := svc.accessSecretVersion(p.name)
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

func (svc *secretClient) setValueLiteral(p params, secretString string) {
	p.v.SetString(secretString)
}

type EnvFileAction string

const (
	PRIORITIZE EnvFileAction = "prioritize"
	DISABLE    EnvFileAction = "disable"
	FALLBACK   EnvFileAction = "fallback"
)
