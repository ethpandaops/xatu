Minimal-preset phase0 cases from [ethereum/consensus-spec-tests](https://github.com/ethereum/consensus-spec-tests) v1.5.0, copied unchanged from `tests/minimal/phase0/<runner>/<handler>/pyspec_tests/<case>`. The Python reference spec generated them, so they are independent of the Go code under test.

- `rewards/*`: a state and the `get_*_deltas` the spec computed for it.
- `random/*`, `finality/*`: a pre-state, the blocks applied to it (with deposits, slashings and exits in `random`) and the post-state.
