package util

import (
	"github.com/sirupsen/logrus"
)

// TODO(acsaba): move this under websockets.
// TODO(acsaba): consider if we want to use this in other places too. If yes move it under
//     internal/util/midlog
func NewLogger() *logrus.Logger {
	l := logrus.New()
	l.SetFormatter(&logrus.TextFormatter{TimestampFormat: "2006-01-02 15:04:05", FullTimestamp: true})
	return l
}
