package secret

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestSecretRef_FromEnv(t *testing.T) {
	const secretEnvVar = "SOME_SECRET"
	const expectedSecret = "123"

	type withRef struct {
		SomeSecret Ref `yaml:"some-secret"`
	}

	conf := fmt.Sprintf(`some-secret: env:%s`, secretEnvVar)
	t.Setenv(secretEnvVar, expectedSecret)

	holder := withRef{}
	require.NoError(t, yaml.Unmarshal([]byte(conf), &holder))

	secret, err := holder.SomeSecret.Read()
	require.NoError(t, err)

	assert.Equal(t, expectedSecret, secret)
}

func tempFile(t *testing.T, prefix string) *os.File {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", "*")
	require.NoError(t, err)
	t.Cleanup(func() {
		err := os.RemoveAll(tmpDir)
		if err != nil {
			t.Log(err.Error())
		}
	})

	tmpFile, err := os.CreateTemp(tmpDir, fmt.Sprintf("%s_*", prefix))
	require.NoError(t, err)

	return tmpFile
}

func TestSecretRef_FromFile(t *testing.T) {
	const expectedSecret = "1234"

	type withRef struct {
		AnotherFileSecret Ref `yaml:"another-file-secret"`
	}

	tmpFile := tempFile(t, "test-secret-config")

	_, err := tmpFile.Write([]byte(expectedSecret))
	require.NoError(t, err)
	require.NoError(t, tmpFile.Close())

	conf := fmt.Sprintf("another-file-secret: %s", tmpFile.Name())

	holder := withRef{}
	require.NoError(t, yaml.Unmarshal([]byte(conf), &holder))

	res, err := holder.AnotherFileSecret.Read()
	require.NoError(t, err)

	assert.Equal(t, expectedSecret, res)
}
