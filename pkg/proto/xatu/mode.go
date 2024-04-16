package xatu

type Mode string

const (
	ModeUnknown   Mode = ""
	ModeSentry    Mode = "sentry"
	ModeCannon    Mode = "cannon"
	ModeServer    Mode = "server"
	ModeMimicry   Mode = "mimicry"
	ModeDiscovery Mode = "discovery"
	ModeSage      Mode = "sage"
	ModeCLMimicry Mode = "cl-mimicry"
	ModeELMimicry Mode = "el-mimicry"
)
