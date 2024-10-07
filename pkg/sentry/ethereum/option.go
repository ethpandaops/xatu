package ethereum

type Options struct {
	FetchBeaconCommittees bool
	FetchProposerDuties   bool
}

func (o *Options) WithFetchBeaconCommittees(fetchBeaconCommittees bool) *Options {
	o.FetchBeaconCommittees = fetchBeaconCommittees

	return o
}

func (o *Options) WithFetchProposerDuties(fetchProposerDuties bool) *Options {
	o.FetchProposerDuties = fetchProposerDuties

	return o
}
