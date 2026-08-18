package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/kingaiwork/KINGAIOS/internal/installer"
)

var version = "0.1.0-dev"

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	switch os.Args[1] {
	case "list":
		list()
	case "plan":
		plan(os.Args[2:])
	case "execute":
		execute(os.Args[2:])
	case "version":
		fmt.Printf("KINGAI Installer %s\n", version)
	default:
		usage()
		os.Exit(2)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, `usage: kingai-installer <list|plan|execute|version>
  kingai-installer plan --target /dev/DEVICE --profile server|desktop|iot
  KINGAI_INSTALLER_ALLOW_WRITE=1 kingai-installer execute \
    --target /dev/DEVICE --profile server|desktop|iot \
    --source-root /path/to/rootfs --state-key /root/state.key \
    --confirm ERASE:/dev/DEVICE

execute is destructive and fail-closed. It requires root, an exact target confirmation,
a verified KINGAI OS source root, a private STATE key file and explicit write enablement.
IoT destructive execution is currently reviewed only for amd64 UEFI systems.
Operational formatter/partitioner/bootloader output is sent to stderr; stdout is JSON only.`)
}

func list() {
	devs, err := installer.Discover()
	if err != nil {
		fail(err)
	}
	writeJSON(installer.CandidateDisks(devs))
}

func plan(args []string) {
	fs := flag.NewFlagSet("plan", flag.ExitOnError)
	target := fs.String("target", "", "target disk")
	profile := fs.String("profile", "server", "distribution profile")
	_ = fs.Parse(args)
	devs, err := installer.Discover()
	if err != nil {
		fail(err)
	}
	p, err := installer.BuildPlan(devs, *target, *profile)
	if err != nil {
		fail(err)
	}
	writeJSON(p)
}

func execute(args []string) {
	fs := flag.NewFlagSet("execute", flag.ExitOnError)
	target := fs.String("target", "", "target disk")
	profile := fs.String("profile", "server", "distribution profile")
	sourceRoot := fs.String("source-root", "", "verified KINGAI OS source root filesystem")
	stateKey := fs.String("state-key", "", "private file used to initialize LUKS2 STATE")
	confirm := fs.String("confirm", "", "must exactly equal ERASE:/dev/DEVICE")
	_ = fs.Parse(args)

	// Keep stdout machine-readable. Destructive-operation diagnostics are sent
	// to stderr; only the final install result is serialized to stdout.
	jsonOut := os.Stdout
	os.Stdout = os.Stderr
	opts := installer.ExecuteOptions{
		Target:       *target,
		Profile:      *profile,
		SourceRoot:   *sourceRoot,
		StateKey:     *stateKey,
		Confirmation: *confirm,
	}
	var res installer.InstallResult
	var err error
	if *profile == "iot" {
		res, err = installer.ExecuteIoT(opts)
	} else {
		res, err = installer.Execute(opts)
	}
	if err == nil {
		err = installer.FinalizeInstalledSystem(res)
	}
	if err == nil {
		err = installer.FinalizeUEFIBoot(res)
	}
	os.Stdout = jsonOut
	if err != nil {
		fail(err)
	}
	writeJSON(res)
}

func writeJSON(v any) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	_ = enc.Encode(v)
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "installer failed:", err)
	os.Exit(1)
}
