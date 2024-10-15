package xatu

import (
	"fmt"
	"runtime"
	"strings"
)

var (
	Release        = "dev"
	GitCommit      = "dev"
	Implementation = "Xatu"
	GOOS           = runtime.GOOS
	GOARCH         = runtime.GOARCH
)

func Full() string {
	return fmt.Sprintf("%s/%s", Implementation, Short())
}

func FullWithModule(module ModuleName) string {
	return fmt.Sprintf("%s-%s/%s", Implementation, module.String(), Short())
}

func WithModule(module ModuleName) string {
	return fmt.Sprintf("%s-%s", Implementation, module.String())
}

func Short() string {
	return fmt.Sprintf("%s-%s", Release, GitCommit)
}

func FullVWithGOOS() string {
	return fmt.Sprintf("%s/%s", Full(), GOOS)
}

func FullVWithPlatform() string {
	return fmt.Sprintf("%s/%s/%s", Full(), GOOS, GOARCH)
}

func ImplementationLower() string {
	return strings.ToLower(Implementation)
}
