package secret

import (
	"encoding/json"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

func ExampleNewRef() {
	_ = os.Setenv("SOME_VAR", "secret")

	ref := NewEnvRef("SOME_VAR")
	sec, _ := ref.Read()
	fmt.Println(sec)
	// Output:
	// secret
}

func ExampleRef_UnmarshalJSON() {
	_ = os.Setenv("SOME_VAR", "secret_json")

	type JSONConfig struct {
		SecretVal Ref `json:"secret_val"`
	}

	configValue := []byte(`{
	"secret_val": "env:SOME_VAR"
}`)
	cfg := JSONConfig{}
	_ = json.Unmarshal(configValue, &cfg)

	sec, _ := cfg.SecretVal.Read()
	fmt.Println(sec)
	// Output:
	// secret_json
}

func ExampleRef_UnmarshalYAML() {
	_ = os.Setenv("SOME_VAR", "secret_yaml")

	type YAMLConfig struct {
		SecretVal Ref `yaml:"secret-val"`
	}

	configValue := []byte(`secret-val: env:SOME_VAR`)
	cfg := YAMLConfig{}
	_ = yaml.Unmarshal(configValue, &cfg)

	sec, _ := cfg.SecretVal.Read()
	fmt.Println(sec)
	// Output:
	// secret_yaml
}
