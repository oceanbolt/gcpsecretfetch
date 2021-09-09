package gcpsecretfetch

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestSetSecrets(t *testing.T) {

	err := UpdateSecrets(GCP_PROJECT, map[string]string{"SECRET_IDENTIFIER": "SECRET_VALUE", "BOTH_IDENTIFIER": "GCP"}, WithDisablePrior())
	assert.NoError(t, err)

}

func TestDeletePriorSecrets(t *testing.T) {

	client, err := newClient(GCP_PROJECT, nil)
	assert.NoError(t, err)

	_, err = client.addVersion("SECRET_IDENTIFIER", "V1")
	assert.NoError(t, err)

	_, err = client.addVersion("SECRET_IDENTIFIER", "V2")
	assert.NoError(t, err)

	err = UpdateSecrets(GCP_PROJECT, map[string]string{"SECRET_IDENTIFIER": "SECRET_VALUE", "BOTH_IDENTIFIER": "GCP"}, WithDisablePrior())
	assert.NoError(t, err)

}
