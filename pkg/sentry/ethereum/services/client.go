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
	ClientGrandine   Client = "grandine"
	ClientCaplin     Client = "caplin"
	ClientTysm       Client = "tysm"
)

var AllClients = []Client{
	ClientUnknown,
	ClientLighthouse,
	ClientNimbus,
	ClientTeku,
	ClientPrysm,
	ClientLodestar,
	ClientGrandine,
	ClientCaplin,
	ClientTysm,
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

	if strings.Contains(asLower, "tysm") {
		return ClientTysm
	}

	if strings.Contains(asLower, "prysm") {
		return ClientPrysm
	}

	if strings.Contains(asLower, "lodestar") {
		return ClientLodestar
	}

	if strings.Contains(asLower, "grandine") {
		return ClientGrandine
	}

	if strings.Contains(asLower, "caplin") {
		return ClientCaplin
	}

	return ClientUnknown
}
