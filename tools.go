// +build tools

package main

// This file contains import statements to the binary tools used by the
// project.  This is the recommended approach per
// https://github.com/golang/go/wiki/Modules#how-can-i-track-tool-dependencies-for-a-module

import (
	// golangci-lint is used to lint the project
	_ "github.com/golangci/golangci-lint/cmd/golangci-lint"
)
