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
	return fmt.Sprintf("%s/%s", WithModule(module), Short())
}

func WithModule(module ModuleName) string {
	return WithComponent(module.String())
}

// WithComponent returns the telemetry service name for a Xatu component.
func WithComponent(component string) string {
	return fmt.Sprintf("%s-%s", ImplementationLower(), formatComponentName(component))
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

func formatComponentName(component string) string {
	return strings.ReplaceAll(strings.ToLower(component), "_", "-")
}
