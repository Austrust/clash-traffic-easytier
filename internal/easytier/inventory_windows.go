package easytier

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/sys/windows"
)

type commandRecord struct {
	Command string `json:"command"`
	Path    string `json:"path"`
	Source  string `json:"source"`
	RPC     string `json:"rpc"`
}

// Only EasyTier processes/services are inspected. Full command lines stay in
// local memory; only executable directories, RPC and config-dir are retained.
func hostInventory(ctx context.Context) inventory {
	ctx, cancel := context.WithTimeout(ctx, 6*time.Second)
	defer cancel()
	shell := filepath.Join(os.Getenv("SystemRoot"), "System32", "WindowsPowerShell", "v1.0", "powershell.exe")
	script := `[Console]::OutputEncoding = [System.Text.UTF8Encoding]::new($false)
$ErrorActionPreference = 'SilentlyContinue'
$rows = @()
Get-CimInstance Win32_Service -Filter "Name LIKE '%easytier%' OR PathName LIKE '%easytier%'" | Where-Object State -eq 'Running' | ForEach-Object {
 $rows += @{command=$_.PathName;path='';source='Windows service';rpc=''}
}
$et = @(Get-CimInstance Win32_Process -Filter "Name = 'easytier-gui.exe' OR Name = 'easytier-core.exe'")
foreach ($p in $et) { $rows += @{command=$p.CommandLine;path=$p.ExecutablePath;source='EasyTier process';rpc=''} }
if ($et.Count -gt 0) {
 Get-NetTCPConnection -State Listen | Where-Object { $_.OwningProcess -in $et.ProcessId -and $_.LocalAddress -in @('127.0.0.1','0.0.0.0','::1','::') } | ForEach-Object {
  $rows += @{command='';path='';source='EasyTier listener';rpc=('127.0.0.1:' + $_.LocalPort)}
 }
}
ConvertTo-Json -InputObject @($rows) -Depth 3 -Compress`
	cmd := exec.CommandContext(ctx, shell, "-NoProfile", "-NonInteractive", "-Command", script)
	hideWindow(cmd)
	out, err := cmd.Output()
	if err != nil {
		return inventory{Warning: "Windows service discovery unavailable; default ports were tried"}
	}
	var rows []commandRecord
	if err := json.Unmarshal(out, &rows); err != nil {
		return inventory{Warning: "Windows service discovery returned invalid metadata"}
	}
	return inventoryFromRecords(rows)
}

func inventoryFromRecords(rows []commandRecord) inventory {
	var out inventory
	for _, r := range rows {
		args, _ := windows.DecomposeCommandLine(r.Command)
		if r.Path == "" && len(args) > 0 {
			r.Path = args[0]
		}
		if r.Path != "" {
			out.Dirs = append(out.Dirs, filepath.Dir(r.Path))
		}
		rpc, dir := r.RPC, ""
		for i, arg := range args {
			if strings.HasPrefix(arg, "--rpc-portal=") {
				rpc = strings.TrimPrefix(arg, "--rpc-portal=")
			}
			if strings.HasPrefix(arg, "--config-dir=") {
				dir = strings.TrimPrefix(arg, "--config-dir=")
			}
			if i+1 < len(args) {
				switch arg {
				case "--rpc-portal":
					rpc = args[i+1]
				case "--config-dir":
					dir = args[i+1]
				}
			}
		}
		if v, err := normalizeRPC(rpc, true); err == nil {
			out.Endpoints = append(out.Endpoints, endpoint{RPC: v, Dir: dir, Source: r.Source})
		}
	}
	return out
}
