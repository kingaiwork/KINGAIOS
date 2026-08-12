package installer

import (
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"sort"
)

const GiB int64 = 1024 * 1024 * 1024
const MiB int64 = 1024 * 1024

type Device struct {
	Name        string   `json:"name"`
	Path        string   `json:"path"`
	Type        string   `json:"type"`
	Size        int64    `json:"size"`
	Model       string   `json:"model,omitempty"`
	Transport   string   `json:"tran,omitempty"`
	ReadOnly    bool     `json:"ro"`
	Removable   bool     `json:"rm"`
	Mountpoints []string `json:"mountpoints,omitempty"`
	Children    []Device `json:"children,omitempty"`
}

type lsblkOutput struct { BlockDevices []Device `json:"blockdevices"` }

type Partition struct {
	Name       string `json:"name"`
	SizeBytes  int64  `json:"size_bytes"`
	Filesystem string `json:"filesystem"`
	Purpose    string `json:"purpose"`
}

type Plan struct {
	Target       string      `json:"target"`
	Profile      string      `json:"profile"`
	DiskBytes    int64       `json:"disk_bytes"`
	Destructive  bool        `json:"destructive"`
	Executable   bool        `json:"executable"`
	Partitions   []Partition `json:"partitions"`
	Requirements []string    `json:"requirements"`
}

func Discover() ([]Device, error) {
	cmd := exec.Command("lsblk", "-J", "-b", "-o", "NAME,PATH,TYPE,SIZE,MODEL,TRAN,RO,RM,MOUNTPOINTS")
	b, err := cmd.Output()
	if err != nil { return nil, fmt.Errorf("lsblk: %w", err) }
	return ParseDevices(b)
}

func ParseDevices(b []byte) ([]Device, error) {
	var out lsblkOutput
	if err := json.Unmarshal(b, &out); err != nil { return nil, fmt.Errorf("decode lsblk: %w", err) }
	return out.BlockDevices, nil
}

func CandidateDisks(devices []Device) []Device {
	var out []Device
	for _, d := range devices {
		if d.Type == "disk" { out = append(out, d) }
	}
	sort.Slice(out, func(i,j int) bool { return out[i].Path < out[j].Path })
	return out
}

func BuildPlan(devices []Device, target, profile string) (Plan, error) {
	if target == "" { return Plan{}, errors.New("target device is required") }
	var disk *Device
	for i := range devices { if devices[i].Type == "disk" && devices[i].Path == target { disk = &devices[i]; break } }
	if disk == nil { return Plan{}, errors.New("target must be a top-level block disk returned by lsblk") }
	if disk.ReadOnly { return Plan{}, errors.New("target disk is read-only") }
	if hasMount(*disk) { return Plan{}, errors.New("target disk or one of its partitions is mounted") }

	min, rootSlot, err := profileSizes(profile)
	if err != nil { return Plan{}, err }
	if disk.Size < min { return Plan{}, fmt.Errorf("target disk too small: need at least %d bytes", min) }
	const efi = 512 * MiB
	state := disk.Size - efi - 2*rootSlot
	if state < 2*GiB { return Plan{}, errors.New("not enough space for persistent state") }

	return Plan{
		Target: target, Profile: profile, DiskBytes: disk.Size, Destructive: true, Executable: false,
		Partitions: []Partition{
			{Name:"EFI",SizeBytes:efi,Filesystem:"vfat",Purpose:"UEFI system partition"},
			{Name:"ROOT_A",SizeBytes:rootSlot,Filesystem:"erofs/ext4 (release dependent)",Purpose:"active verified system slot"},
			{Name:"ROOT_B",SizeBytes:rootSlot,Filesystem:"erofs/ext4 (release dependent)",Purpose:"inactive atomic update slot"},
			{Name:"STATE",SizeBytes:state,Filesystem:"luks2 + ext4/btrfs (release dependent)",Purpose:"encrypted persistent user and KINGAI state"},
		},
		Requirements: []string{"explicit user confirmation","verified installer image","power-safe transaction","post-install boot validation","rollback path"},
	}, nil
}

func profileSizes(profile string)(minimum,rootSlot int64,err error){
	switch profile {
	case "server": return 24*GiB,6*GiB,nil
	case "desktop": return 40*GiB,12*GiB,nil
	case "iot": return 8*GiB,2*GiB,nil
	default: return 0,0,fmt.Errorf("unknown profile %q",profile)
	}
}

func hasMount(d Device) bool {
	for _,m:=range d.Mountpoints { if m!="" { return true } }
	for _,c:=range d.Children { if hasMount(c) { return true } }
	return false
}
