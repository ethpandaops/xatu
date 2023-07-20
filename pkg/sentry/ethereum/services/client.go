package services

import (
	"strings"
)

type Client string

const (
	ClientUnknown    Client = "unknown"
	ClientLighthouse Client = "lighthouse"
	ClientNimbus     Client = "nimbus"
	ClientTeku       Client = "teku"
	ClientPrysm      Client = "prysm"
	ClientLodestar   Client = "lodestar"
)

var AllClients = []Client{
	ClientUnknown,
	ClientLighthouse,
	ClientNimbus,
	ClientTeku,
	ClientPrysm,
	ClientLodestar,
}

func ClientFromString(client string) Client {
	asLower := strings.ToLower(client)

	if strings.Contains(asLower, "lighthouse") {
		return ClientLighthouse
	}

	if strings.Contains(asLower, "nimbus") {
		return ClientNimbus
	}

	if strings.Contains(asLower, "teku") {
		return ClientTeku
	}

	if strings.Contains(asLower, "prysm") {
		return ClientPrysm
	}

	if strings.Contains(asLower, "lodestar") {
		return ClientLodestar
	}

	return ClientUnknown
}
