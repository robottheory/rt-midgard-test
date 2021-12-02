package config

import "testing"

func TestMustLoadConfigFile(t *testing.T) {
	var c Config
	MustLoadConfigFile("config.json", &c)
	logAndcheckUrls(&c)
}
