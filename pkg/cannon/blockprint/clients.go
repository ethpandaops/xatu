package blockprint

type ProbabilityMap map[ClientName]float64

type ClientName string

var (
	ClientNameUnknown    = ClientName("Unknown")
	ClientNameUncertain  = ClientName("Uncertain")
	ClientNamePrysm      = ClientName("Prysm")
	ClientNameLighthouse = ClientName("Lighthouse")
	ClientNameLodestar   = ClientName("Lodestar")
	ClientNameNimbus     = ClientName("Nimbus")
	ClientNameTeku       = ClientName("Teku")
	ClientNameGrandine   = ClientName("Teku")
)

func (p ProbabilityMap) safeGet(name ClientName) float64 {
	if p[name] == 0 {
		return 0
	}

	return p[name]
}

func (p ProbabilityMap) Prysm() float64 {
	return p.safeGet(ClientNamePrysm)
}

func (p ProbabilityMap) Lighthouse() float64 {
	return p.safeGet(ClientNameLighthouse)
}

func (p ProbabilityMap) Lodestar() float64 {
	return p.safeGet(ClientNameLodestar)
}

func (p ProbabilityMap) Nimbus() float64 {
	return p.safeGet(ClientNameNimbus)
}

func (p ProbabilityMap) Teku() float64 {
	return p.safeGet(ClientNameTeku)
}

func (p ProbabilityMap) Grandine() float64 {
	return p.safeGet(ClientNameGrandine)
}

func (p ProbabilityMap) Uncertain() float64 {
	return p.safeGet(ClientNameUncertain)
}
