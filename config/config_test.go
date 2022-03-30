package config

import (
	"bytes"
	"io"
	"os"
	"testing"

	"gitlab.com/thorchain/midgard/internal/util/midlog"
)

func initTest(t *testing.T) {
	midlog.SetExitFunctionForTest(t.FailNow)
	b := bytes.Buffer{}
	midlog.SetGlobalOutput(&b)

	t.Cleanup(func() {
		if t.Failed() {
			io.Copy(os.Stdout, &b)
		}
	})
}

func TestMustLoadConfigFile(t *testing.T) {
	initTest(t)

	var c Config
	MustLoadConfigFiles("config.json", &c)
	logAndcheckUrls(&c)
}
