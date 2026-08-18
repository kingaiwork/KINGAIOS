package installer

import (
	"errors"
	"os"
	"runtime"
)

// ExecuteIoT closes the existing IoT planner/executor gap without broadening
// the reviewed destructive-write architecture. It reuses the exact GPT/EFI,
// A/B root, encrypted STATE and GRUB transaction used by the server/desktop
// installer, but is callable only for the IoT profile on amd64 UEFI systems.
func ExecuteIoT(opts ExecuteOptions) (InstallResult, error) {
	if runtime.GOARCH != "amd64" {
		return InstallResult{}, errors.New("IoT destructive installer execution is currently reviewed only for amd64 UEFI systems")
	}
	if opts.Profile != "iot" {
		return InstallResult{}, errors.New("IoT installer execution requires profile=iot")
	}
	if os.Geteuid() != 0 {
		return InstallResult{}, errors.New("installer execute requires root")
	}
	if os.Getenv("KINGAI_INSTALLER_ALLOW_WRITE") != "1" {
		return InstallResult{}, errors.New("disk writing is disabled; set KINGAI_INSTALLER_ALLOW_WRITE=1 explicitly")
	}
	if opts.Confirmation != "ERASE:"+opts.Target {
		return InstallResult{}, errors.New("confirmation mismatch; expected ERASE:<exact target>")
	}
	devs, err := Discover()
	if err != nil {
		return InstallResult{}, err
	}
	plan, err := BuildPlan(devs, opts.Target, opts.Profile)
	if err != nil {
		return InstallResult{}, err
	}
	if err := validateExecutionInputs(opts); err != nil {
		return InstallResult{}, err
	}
	return executePlan(osRunner{}, plan, opts)
}
