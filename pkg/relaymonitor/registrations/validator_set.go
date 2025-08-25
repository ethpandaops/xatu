package registrations

import (
	"sort"
	"sync"

	"math/rand/v2"

	apiv1 "github.com/attestantio/go-eth2-client/api/v1"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/pkg/errors"
)

// ValidatorSetWalker is a struct that walks through the validator set and returns the next validator index to register
type ValidatorSetWalker struct {
	validators   []phase0.ValidatorIndex
	validatorMap map[phase0.ValidatorIndex]*apiv1.Validator
	marker       *phase0.ValidatorIndex
	mu           sync.RWMutex
	shard        int
	numShards    int

	min phase0.ValidatorIndex
	max phase0.ValidatorIndex
}

// NewValidatorSetWalker creates a new ValidatorSetWalker
func NewValidatorSetWalker(shard, numShards int) *ValidatorSetWalker {
	return &ValidatorSetWalker{
		shard:     shard,
		numShards: numShards,
	}
}

func (v *ValidatorSetWalker) Shard() int {
	return v.shard
}

func (v *ValidatorSetWalker) NumShards() int {
	return v.numShards
}

func (v *ValidatorSetWalker) Validators() []phase0.ValidatorIndex {
	return v.validators
}

func (v *ValidatorSetWalker) Marker() *phase0.ValidatorIndex {
	return v.marker
}

func (v *ValidatorSetWalker) Max() phase0.ValidatorIndex {
	return v.max
}

func (v *ValidatorSetWalker) Min() phase0.ValidatorIndex {
	return v.min
}

func (v *ValidatorSetWalker) NumValidatorsInShard() uint64 {
	return uint64(v.max - v.min)
}

func (v *ValidatorSetWalker) updateMin() {
	validatorsPerShard := len(v.validators) / v.numShards

	//nolint:gosec // not concerned about this in reality
	v.min = phase0.ValidatorIndex(v.shard * validatorsPerShard)
}

func (v *ValidatorSetWalker) updateMax() {
	validatorsPerShard := len(v.validators) / v.numShards

	m := (v.shard + 1) * validatorsPerShard

	if v.shard == v.numShards-1 {
		m = len(v.validators)
	}

	//nolint:gosec // not concerned about this in reality
	v.max = phase0.ValidatorIndex(m)
}

func (v *ValidatorSetWalker) ValidatorMap() map[phase0.ValidatorIndex]*apiv1.Validator {
	return v.validatorMap
}

func (v *ValidatorSetWalker) GetValidator(index *phase0.ValidatorIndex) (*apiv1.Validator, error) {
	validator, ok := v.validatorMap[*index]
	if !ok {
		return nil, errors.New("validator not found")
	}

	return validator, nil
}

func (v *ValidatorSetWalker) Update(validators map[phase0.ValidatorIndex]*apiv1.Validator) {
	v.mu.Lock()
	defer v.mu.Unlock()

	v.validators = make([]phase0.ValidatorIndex, 0, len(validators))
	for validatorIndex := range validators {
		v.validators = append(v.validators, validatorIndex)
	}

	sort.Slice(v.validators, func(i, j int) bool {
		return v.validators[i] < v.validators[j]
	})

	v.updateMin()
	v.updateMax()

	// Adjust marker to stay within shard bounds
	if v.marker == nil {
		// First time we're updating the validator set, so we start at a random point
		//nolint:gosec // not concerned about this in reality
		index := phase0.ValidatorIndex(rand.IntN(int(v.max-v.min)) + int(v.min))
		v.marker = &index
	} else if *v.marker < v.min || *v.marker >= v.max {
		*v.marker = v.min
	}

	v.validatorMap = validators
}

func (v *ValidatorSetWalker) Next() (phase0.ValidatorIndex, error) {
	if v.marker == nil {
		return 0, errors.New("marker is nil -- update validator set first")
	}

	v.mu.Lock()
	defer v.mu.Unlock()

	validatorIndex := v.validators[*v.marker]
	*v.marker++

	// Loop back to start of shard if we hit the end
	if *v.marker >= v.Max() {
		*v.marker = v.Min()
	}

	return validatorIndex, nil
}
