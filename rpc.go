package libwallet

import (
	"fmt"

	"github.com/EllipX/libwallet/wltbase"
	"github.com/KarpelesLab/apirouter"
)

// Methods exposed to the application to setup an environment

// MakeRPC generates and return a socket
func MakeRPC(dataDir string) (int, error) {
	e, err := wltbase.InitEnv(dataDir)
	if err != nil {
		return -1, fmt.Errorf("failed to init env: %w", err)
	}

	return apirouter.MakeJsonSocketFD(map[string]any{"@env": e})
}

// MakeSocket creates a socket
func MakeSocket(dataDir, socketName string) error {
	e, err := wltbase.InitEnv(dataDir)
	if err != nil {
		return fmt.Errorf("failed to init env: %w", err)
	}

	return apirouter.MakeJsonUnixListener(socketName, map[string]any{"@env": e})
}
