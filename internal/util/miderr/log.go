package miderr

import "gitlab.com/thorchain/midgard/internal/util/midlog"

func Printf(format string, v ...interface{}) {
	midlog.WarnF(format, v...)
}
