package config

import "testing"

func TestMustLoadConfigFile(t *testing.T) {
	MustLoadConfigFile("config.json")
}
