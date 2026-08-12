package main

import (
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"strings"
)

type status struct {
	Name       string `json:"name"`
	Version    string `json:"version"`
	Channel    string `json:"channel"`
	GoVersion  string `json:"go_version"`
	Platform   string `json:"platform"`
	D4         bool   `json:"d4"`
	LocalFirst bool   `json:"local_first"`
}

var version = "0.1.0-dev"

func usage() {
	fmt.Print(`KINGAI OS CLI

Usage:
  kingai version
  kingai status [--json]
  kingai doctor

This is the D4 Developer Foundation CLI. Privileged actions will be routed
through the future capability/policy execution broker instead of direct root shells.
`)
}

func main() {
	if len(os.Args) < 2 {
		usage()
		return
	}

	switch os.Args[1] {
	case "version":
		fmt.Printf("KINGAI OS %s\n", version)
	case "status":
		s := status{
			Name:       "KINGAI OS",
			Version:    version,
			Channel:    "dev",
			GoVersion:  runtime.Version(),
			Platform:   runtime.GOOS + "/" + runtime.GOARCH,
			D4:         true,
			LocalFirst: true,
		}
		if len(os.Args) > 2 && strings.EqualFold(os.Args[2], "--json") {
			_ = json.NewEncoder(os.Stdout).Encode(s)
			return
		}
		fmt.Printf("System:      %s\n", s.Name)
		fmt.Printf("Version:     %s\n", s.Version)
		fmt.Printf("Channel:     %s\n", s.Channel)
		fmt.Printf("Platform:    %s\n", s.Platform)
		fmt.Printf("Architecture: D4 Sovereign Distributed Intelligence\n")
	case "doctor":
		fmt.Println("KINGAI Doctor")
		fmt.Println("[ok] CLI runtime")
		fmt.Println("[info] developer foundation: policy/model/memory daemons are not yet enabled")
	default:
		usage()
		os.Exit(2)
	}
}
