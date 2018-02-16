// Copyright 2017 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package cfg holds configuration shared by multiple parts
// of the go command.
package cfg

import (
	"bytes"
	"fmt"
	"go/build"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
)

// These are general "build flags" used by build and other commands.
var (
	BuildA                 bool   // -a flag
	BuildBuildmode         string // -buildmode flag
	BuildContext           = build.Default
	BuildI                 bool               // -i flag
	BuildLinkshared        bool               // -linkshared flag
	BuildMSan              bool               // -msan flag
	BuildN                 bool               // -n flag
	BuildO                 string             // -o flag
	BuildP                 = runtime.NumCPU() // -p flag
	BuildPkgdir            string             // -pkgdir flag
	BuildRace              bool               // -race flag
	BuildToolexec          []string           // -toolexec flag
	BuildToolchainName     string
	BuildToolchainCompiler func() string
	BuildToolchainLinker   func() string
	BuildV                 bool // -v flag
	BuildWork              bool // -work flag
	BuildX                 bool // -x flag

	CmdName string // "build", "install", "list", etc.

	DebugActiongraph string // -debug-actiongraph flag (undocumented, unstable)
)

func init() {
	BuildToolchainCompiler = func() string { return "missing-compiler" }
	BuildToolchainLinker = func() string { return "missing-linker" }
}

// An EnvVar is an environment variable Name=Value.
type EnvVar struct {
	Name  string
	Value string
}

// OrigEnv is the original environment of the program at startup.
var OrigEnv []string

// CmdEnv is the new environment for running go tool commands.
// User binaries (during go test or go run) are run with OrigEnv,
// not CmdEnv.
var CmdEnv []EnvVar

// Global build parameters (used during package load)
var (
	Goarch    = BuildContext.GOARCH
	Goos      = BuildContext.GOOS
	ExeSuffix string
	Gopath    = filepath.SplitList(BuildContext.GOPATH)
)

func init() {
	if Goos == "windows" {
		ExeSuffix = ".exe"
	}
}

var (
	GOROOT       = findGOROOT()
	GOBIN        = os.Getenv("GOBIN")
	GOROOTbin    = filepath.Join(GOROOT, "bin")
	GOROOTpkg    = filepath.Join(GOROOT, "pkg")
	GOROOTsrc    = filepath.Join(GOROOT, "src")
	GOROOT_FINAL = findGOROOT_FINAL()

	// Used in envcmd.MkEnv and build ID computations.
	GOARM, GO386, GOMIPS = objabi()
)

// Update build context to use our computed GOROOT.
func init() {
	BuildContext.GOROOT = GOROOT
	// Note that we must use runtime.GOOS and runtime.GOARCH here,
	// as the tool directory does not move based on environment variables.
	// This matches the initialization of ToolDir in go/build,
	// except for using GOROOT rather than runtime.GOROOT().
	build.ToolDir = filepath.Join(GOROOT, "pkg/tool/"+runtime.GOOS+"_"+runtime.GOARCH)
}

func objabi() (GOARM, GO386, GOMIPS string) {
	data, err := ioutil.ReadFile(filepath.Join(GOROOT, "src/cmd/internal/objabi/zbootstrap.go"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "vgo objabi: %v\n", err)
	}

	find := func(key string) string {
		if env := os.Getenv(key); env != "" {
			return env
		}
		i := bytes.Index(data, []byte("default"+key+" = `"))
		if i < 0 {
			fmt.Fprintf(os.Stderr, "vgo objabi: cannot find %s\n", key)
			os.Exit(2)
		}
		line := data[i:]
		line = line[bytes.IndexByte(line, '`')+1:]
		return string(line[:bytes.IndexByte(line, '`')])
	}

	return find("GOARM"), find("GO386"), find("GOMIPS")
}

func findGOROOT() string {
	if env := os.Getenv("VGOROOT"); env != "" {
		return filepath.Clean(env)
	}
	if env := os.Getenv("GOROOT"); env != "" {
		return filepath.Clean(env)
	}
	def := filepath.Clean(runtime.GOROOT())
	exe, err := os.Executable()
	if err == nil {
		exe, err = filepath.Abs(exe)
		if err == nil {
			if dir := filepath.Join(exe, "../.."); isGOROOT(dir) {
				// If def (runtime.GOROOT()) and dir are the same
				// directory, prefer the spelling used in def.
				if isSameDir(def, dir) {
					return def
				}
				return dir
			}
			exe, err = filepath.EvalSymlinks(exe)
			if err == nil {
				if dir := filepath.Join(exe, "../.."); isGOROOT(dir) {
					if isSameDir(def, dir) {
						return def
					}
					return dir
				}
			}
		}
	}
	return def
}

func findGOROOT_FINAL() string {
	def := GOROOT
	if env := os.Getenv("GOROOT_FINAL"); env != "" {
		def = filepath.Clean(env)
	}
	return def
}

// isSameDir reports whether dir1 and dir2 are the same directory.
func isSameDir(dir1, dir2 string) bool {
	if dir1 == dir2 {
		return true
	}
	info1, err1 := os.Stat(dir1)
	info2, err2 := os.Stat(dir2)
	return err1 == nil && err2 == nil && os.SameFile(info1, info2)
}

// isGOROOT reports whether path looks like a GOROOT.
//
// It does this by looking for the path/pkg/tool directory,
// which is necessary for useful operation of the cmd/go tool,
// and is not typically present in a GOPATH.
func isGOROOT(path string) bool {
	stat, err := os.Stat(filepath.Join(path, "pkg", "tool"))
	if err != nil {
		return false
	}
	return stat.IsDir()
}