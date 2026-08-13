package main

import (
	"crypto"
	"crypto/ed25519"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/sigstore/sigstore/pkg/signature"
	"github.com/theupdateframework/go-tuf/v2/metadata"
)

func main() {
	output := flag.String("output", "", "path for ephemeral CI root.json")
	expires := flag.Duration("expires", 365*24*time.Hour, "root validity interval")
	mode := flag.Uint("mode", 0o644, "output file permission bits")
	flag.Parse()
	if *output == "" {
		fmt.Fprintln(os.Stderr, "--output is required")
		os.Exit(2)
	}
	if *expires <= 0 {
		fmt.Fprintln(os.Stderr, "--expires must be positive")
		os.Exit(2)
	}

	root := metadata.Root(time.Now().UTC().Add(*expires))
	privateKeys := make([]ed25519.PrivateKey, 0, 2)
	for i := 0; i < 2; i++ {
		_, private, err := ed25519.GenerateKey(nil)
		if err != nil { fatal(err) }
		key, err := metadata.KeyFromPublicKey(private.Public())
		if err != nil { fatal(err) }
		if err := root.Signed.AddKey(key, metadata.ROOT); err != nil { fatal(err) }
		privateKeys = append(privateKeys, private)
	}
	root.Signed.Roles[metadata.ROOT].Threshold = 2
	for _, private := range privateKeys {
		signer, err := signature.LoadSigner(private, crypto.Hash(0))
		if err != nil { fatal(err) }
		if _, err := root.Sign(signer); err != nil { fatal(err) }
	}
	if err := root.VerifyDelegate(metadata.ROOT, root); err != nil { fatal(fmt.Errorf("self-verify generated root: %w", err)) }
	if err := os.MkdirAll(dirOf(*output), 0o755); err != nil { fatal(err) }
	if err := root.ToFile(*output, false); err != nil { fatal(err) }
	if err := os.Chmod(*output, os.FileMode(*mode)&0o777); err != nil { fatal(err) }
}

func dirOf(path string) string {
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '/' {
			if i == 0 { return "/" }
			return path[:i]
		}
	}
	return "."
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
