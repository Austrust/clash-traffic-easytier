package easytier

import "testing"

func TestWindowsServiceCommandMetadata(t *testing.T) {
	r := inventoryFromRecords([]commandRecord{{Command: `"C:\Program Files\EasyTier\easytier-gui.exe" --config-dir "C:\Users\A B\config.d" --rpc-portal 0.0.0.0:16789 --daemon`, Source: "Windows service"}})
	if len(r.Endpoints) != 1 || r.Endpoints[0].RPC != "127.0.0.1:16789" || r.Endpoints[0].Dir != `C:\Users\A B\config.d` || r.Dirs[0] != `C:\Program Files\EasyTier` {
		t.Fatalf("%+v", r)
	}
}
