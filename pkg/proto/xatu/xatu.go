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
)

func Full() string {
	return fmt.Sprintf("%s/%s", Implementation, Short())
}

func FullWithModule(module ModuleName) string {
	return fmt.Sprintf("%s-%s/%s", Implementation, module, Short())
}

func WithModule(module ModuleName) string {
	return fmt.Sprintf("%s-%s", Implementation, module)
}

func Short() string {
	return fmt.Sprintf("%s-%s", Release, GitCommit)
}

func FullVWithGOOS() string {
	return fmt.Sprintf("%s/%s", Full(), runtime.GOOS)
}

func ImplementationLower() string {
	return strings.ToLower(Implementation)
}
