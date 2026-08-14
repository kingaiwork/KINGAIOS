package executor

import (
	"errors"
	"path/filepath"
	"strings"
)

func ValidateServiceUnit(unit string) error {
	unit = strings.TrimSpace(unit)
	if unit == "" || len(unit) > 255 || strings.HasPrefix(unit, "-") || strings.ContainsAny(unit, "/\\\x00\n\r\t ") {
		return errors.New("invalid service unit")
	}
	for _, r := range unit {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || strings.ContainsRune("._@:-", r)) {
			return errors.New("invalid service unit")
		}
	}
	return nil
}

func ValidatePackageName(name string) error {
	name = strings.TrimSpace(name)
	if name == "" || len(name) > 128 || strings.HasPrefix(name, "-") {
		return errors.New("invalid package name")
	}
	for _, r := range name {
		if !((r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '+' || r == '-' || r == '.') {
			return errors.New("invalid package name")
		}
	}
	return nil
}

func ValidatePathWithin(root, target string) (string, error) {
	if root == "" || target == "" || strings.ContainsRune(root, '\x00') || strings.ContainsRune(target, '\x00') {
		return "", errors.New("invalid path")
	}
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	targetAbs, err := filepath.Abs(target)
	if err != nil {
		return "", err
	}
	rel, err := filepath.Rel(rootAbs, targetAbs)
	if err != nil {
		return "", err
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", errors.New("path escapes allowed root")
	}
	return targetAbs, nil
}
