package canonical

// setOptionalEpoch sets an optional epoch value only when it is non-zero and not the sentinel
// far-future epoch (^uint64(0)).
func setOptionalEpoch(wrapped interface{ GetValue() uint64 }) (uint64, bool) {
	if wrapped == nil {
		return 0, false
	}

	value := wrapped.GetValue()
	if value == 0 || value == ^uint64(0) {
		return 0, false
	}

	return value, true
}
