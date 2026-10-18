package couchcore

import (
	"fmt"
	"os"
)

// Only the selected fixture roots are named; no ambient environment dump.
func recoveryLiveEnvironmentMark() string {
	return fmt.Sprintf("fixture-roots home=%q config=%q data=%q cache=%q runtime=%q sockets=%q zellij-config=%q",
		os.Getenv("HOME"), os.Getenv("XDG_CONFIG_HOME"), os.Getenv("XDG_DATA_HOME"),
		os.Getenv("XDG_CACHE_HOME"), os.Getenv("XDG_RUNTIME_DIR"), os.Getenv("ZELLIJ_SOCKET_DIR"), os.Getenv("ZELLIJ_CONFIG_DIR"))
}
