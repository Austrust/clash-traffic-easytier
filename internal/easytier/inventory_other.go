//go:build !windows

package easytier

import "context"

// Non-Windows systems retain CLI/PATH discovery and local default RPC probes.
func hostInventory(ctx context.Context) inventory { return inventory{} }
