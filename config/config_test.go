package config

import "testing"

func TestMustLoadConfigFile(t *testing.T) {
	var c Config
	MustLoadConfigFiles("config.json", &c)
	logAndcheckUrls(&c)
}
