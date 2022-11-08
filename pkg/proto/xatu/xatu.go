package xatu

import (
	"fmt"
	"runtime"
)

var (
	Release        = "dev"
	GitCommit      = "dev"
	Implementation = "Xatu"
)

func Full() string {
	return fmt.Sprintf("%s/%s", Implementation, Short())
}

func Short() string {
	return fmt.Sprintf("%s-%s", Release, GitCommit)
}

func FullVWithGOOS() string {
	return fmt.Sprintf("%s/%s", Full(), runtime.GOOS)
}
