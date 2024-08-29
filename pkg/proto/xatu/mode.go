package xatu

type Mode string

const (
	ModeUnknown   Mode = ""
	ModeSentry    Mode = "sentry"
	ModeCannon    Mode = "cannon"
	ModeServer    Mode = "server"
	ModeMimicry   Mode = "mimicry"
	ModeDiscovery Mode = "discovery"
	ModeCLMimicry Mode = "cl-mimicry"
	ModeELMimicry Mode = "el-mimicry"
)
