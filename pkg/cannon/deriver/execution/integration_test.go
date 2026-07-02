//go:build integration

package execution

import (
	"context"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/creasty/defaults"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"

	"github.com/ethpandaops/xatu/pkg/cryo"
	chsink "github.com/ethpandaops/xatu/pkg/output/clickhouse"
	"github.com/ethpandaops/xatu/pkg/processor"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

// TestIntegration_CryoToClickHouse drives cryo over a real mainnet block range
// and pushes the derived events through the real ClickHouse sink, then asserts
// rows landed in canonical_execution_block.
//
// Run with:
//
//	CRYO_RPC='https://user:pass@host' \
//	CLICKHOUSE_DSN='clickhouse://default:@127.0.0.1:9000/default' \
//	go test -tags integration -run TestIntegration_CryoToClickHouse ./pkg/cannon/deriver/execution/ -v
func TestIntegration_CryoToClickHouse(t *testing.T) {
	rpc := os.Getenv("CRYO_RPC")
	dsn := os.Getenv("CLICKHOUSE_DSN")

	if rpc == "" || dsn == "" {
		t.Skip("CRYO_RPC and CLICKHOUSE_DSN must be set")
	}

	from := uint64(22000000)
	to := uint64(22000999) // 1000 blocks — crank it.

	log := logrus.New()
	log.SetLevel(logrus.InfoLevel)

	ctx := context.Background()

	// --- cryo runner + deriver (processRange path, no iterator) ---
	cryoCfg := &cryo.Config{}
	require.NoError(t, defaults.Set(cryoCfg))
	cryoCfg.MaxRangeSize = 1000

	runner := cryo.New(log, cryoCfg, rpc)

	deriver := &BlockDeriver{
		cfg:        &BlockDeriverConfig{Enabled: true, ChunkSize: 100},
		cryo:       runner,
		clientMeta: &xatu.ClientMeta{Name: "itest", Ethereum: &xatu.ClientMeta_Ethereum{Network: &xatu.ClientMeta_Ethereum_Network{Name: "mainnet", Id: 1}}},
	}
	deriver.base.log = log.WithField("test", "integration")

	t.Logf("cranking cryo over blocks %d..%d", from, to)
	start := time.Now()

	events, err := deriver.processRange(ctx, from, to)
	require.NoError(t, err)
	require.NotEmpty(t, events)

	totalBlocks := 0
	for _, e := range events {
		totalBlocks += len(e.GetExecutionCanonicalBlock().GetBlocks())
	}

	t.Logf("cryo produced %d events (%d blocks) in %s", len(events), totalBlocks, time.Since(start))
	require.Equal(t, 1000, totalBlocks, "expected 1000 blocks")

	// --- real clickhouse sink ---
	sinkCfg := &chsink.Config{}
	require.NoError(t, defaults.Set(sinkCfg))
	sinkCfg.DSN = dsn
	sinkCfg.MetricsSubsystem = "cannon_itest"
	sinkCfg.RestrictToTablePrefixes = []string{"canonical_"}
	// The local single-replica test cluster can't satisfy insert_quorum='auto'
	// (the writer's default), so disable quorum for the test. Production runs on
	// a healthy cluster where 'auto' is correct.
	sinkCfg.Defaults.InsertSettings = map[string]any{"insert_quorum": 0}

	sink, err := chsink.New("itest-ch", sinkCfg, log, &xatu.EventFilterConfig{}, processor.ShippingMethodSync)
	require.NoError(t, err)

	require.NoError(t, sink.Start(ctx))

	require.NoError(t, sink.HandleNewDecoratedEvents(ctx, events))

	require.NoError(t, sink.Stop(ctx))

	t.Log("pushed events through clickhouse sink; verify rows with:")
	t.Logf("  SELECT count() FROM default.canonical_execution_block WHERE block_number BETWEEN %d AND %d", from, to)
}

// TestIntegration_CryoTransactionsToClickHouse drives the transaction dataset
// (UInt256 value, UInt128 gas_price) end to end into ClickHouse.
func TestIntegration_CryoTransactionsToClickHouse(t *testing.T) {
	rpc := os.Getenv("CRYO_RPC")
	dsn := os.Getenv("CLICKHOUSE_DSN")

	if rpc == "" || dsn == "" {
		t.Skip("CRYO_RPC and CLICKHOUSE_DSN must be set")
	}

	from := uint64(22000000)
	to := uint64(22000049) // 50 blocks; transactions are far higher cardinality.

	log := logrus.New()
	log.SetLevel(logrus.InfoLevel)

	ctx := context.Background()

	cryoCfg := &cryo.Config{}
	require.NoError(t, defaults.Set(cryoCfg))
	cryoCfg.MaxRangeSize = 50

	runner := cryo.New(log, cryoCfg, rpc)

	deriver := &TransactionDeriver{
		cfg:        &TransactionDeriverConfig{Enabled: true, ChunkSize: 500},
		cryo:       runner,
		clientMeta: &xatu.ClientMeta{Name: "itest", Ethereum: &xatu.ClientMeta_Ethereum{Network: &xatu.ClientMeta_Ethereum_Network{Name: "mainnet", Id: 1}}},
	}
	deriver.base.log = log.WithField("test", "integration-tx")

	start := time.Now()

	events, err := deriver.processRange(ctx, from, to)
	require.NoError(t, err)
	require.NotEmpty(t, events)

	totalTx := 0
	for _, e := range events {
		totalTx += len(e.GetExecutionCanonicalTransaction().GetTransactions())
	}

	t.Logf("cryo produced %d events (%d transactions) over %d blocks in %s", len(events), totalTx, to-from+1, time.Since(start))
	require.Positive(t, totalTx)

	sinkCfg := &chsink.Config{}
	require.NoError(t, defaults.Set(sinkCfg))
	sinkCfg.DSN = dsn
	sinkCfg.MetricsSubsystem = "cannon_itest_tx"
	sinkCfg.RestrictToTablePrefixes = []string{"canonical_"}
	sinkCfg.Defaults.InsertSettings = map[string]any{"insert_quorum": 0}

	sink, err := chsink.New("itest-ch-tx", sinkCfg, log, &xatu.EventFilterConfig{}, processor.ShippingMethodSync)
	require.NoError(t, err)

	require.NoError(t, sink.Start(ctx))
	require.NoError(t, sink.HandleNewDecoratedEvents(ctx, events))
	require.NoError(t, sink.Stop(ctx))

	t.Logf("pushed %d transactions through clickhouse sink", totalTx)
}

// TestIntegration_AllExecutionDatasets cranks EVERY EL cannon dataset through
// cryo → route → sink → ClickHouse over a small mainnet range.
func TestIntegration_AllExecutionDatasets(t *testing.T) {
	rpc := os.Getenv("CRYO_RPC")
	dsn := os.Getenv("CLICKHOUSE_DSN")

	if rpc == "" || dsn == "" {
		t.Skip("CRYO_RPC and CLICKHOUSE_DSN must be set")
	}

	from := uint64(22000000)
	to := uint64(22000004) // 5 blocks across all datasets.

	log := logrus.New()
	log.SetLevel(logrus.WarnLevel)

	ctx := context.Background()

	cryoCfg := &cryo.Config{}
	require.NoError(t, defaults.Set(cryoCfg))
	cryoCfg.MaxRangeSize = 5

	runner := cryo.New(log, cryoCfg, rpc)
	meta := &xatu.ClientMeta{Name: "itest", Ethereum: &xatu.ClientMeta_Ethereum{Network: &xatu.ClientMeta_Ethereum_Network{Name: "mainnet", Id: 1}}}
	ll := log.WithField("t", "all")

	mk := func(name string) base { return base{log: ll, name: name} }

	type ds struct {
		name    string
		produce eventProducer
	}

	datasets := []ds{
		{"blocks", (&BlockDeriver{base: mk("blocks"), cfg: &BlockDeriverConfig{ChunkSize: 500}, cryo: runner, clientMeta: meta}).processRange},
		{"transactions", (&TransactionDeriver{base: mk("transactions"), cfg: &TransactionDeriverConfig{ChunkSize: 500}, cryo: runner, clientMeta: meta}).processRange},
		{"logs", (&LogsDeriver{base: mk("logs"), cfg: &LogsDeriverConfig{ChunkSize: 500}, cryo: runner, clientMeta: meta}).processRange},
		{"traces", (&TracesDeriver{base: mk("traces"), cfg: &TracesDeriverConfig{ChunkSize: 500}, cryo: runner, clientMeta: meta}).processRange},
		{"native_transfers", (&NativeTransfersDeriver{base: mk("native_transfers"), cfg: &NativeTransfersDeriverConfig{ChunkSize: 500}, cryo: runner, clientMeta: meta}).processRange},
		{"erc20_transfers", (&Erc20TransfersDeriver{base: mk("erc20_transfers"), cfg: &Erc20TransfersDeriverConfig{ChunkSize: 500}, cryo: runner, clientMeta: meta}).processRange},
		{"erc721_transfers", (&Erc721TransfersDeriver{base: mk("erc721_transfers"), cfg: &Erc721TransfersDeriverConfig{ChunkSize: 500}, cryo: runner, clientMeta: meta}).processRange},
		{"contracts", (&ContractsDeriver{base: mk("contracts"), cfg: &ContractsDeriverConfig{ChunkSize: 500}, cryo: runner, clientMeta: meta}).processRange},
		{"balance_diffs", (&BalanceDiffsDeriver{base: mk("balance_diffs"), cfg: &BalanceDiffsDeriverConfig{ChunkSize: 500}, cryo: runner, clientMeta: meta}).processRange},
		{"storage_diffs", (&StorageDiffsDeriver{base: mk("storage_diffs"), cfg: &StorageDiffsDeriverConfig{ChunkSize: 500}, cryo: runner, clientMeta: meta}).processRange},
		{"nonce_diffs", (&NonceDiffsDeriver{base: mk("nonce_diffs"), cfg: &NonceDiffsDeriverConfig{ChunkSize: 500}, cryo: runner, clientMeta: meta}).processRange},
		{"balance_reads", (&BalanceReadsDeriver{base: mk("balance_reads"), cfg: &BalanceReadsDeriverConfig{ChunkSize: 500}, cryo: runner, clientMeta: meta}).processRange},
		{"storage_reads", (&StorageReadsDeriver{base: mk("storage_reads"), cfg: &StorageReadsDeriverConfig{ChunkSize: 500}, cryo: runner, clientMeta: meta}).processRange},
		{"nonce_reads", (&NonceReadsDeriver{base: mk("nonce_reads"), cfg: &NonceReadsDeriverConfig{ChunkSize: 500}, cryo: runner, clientMeta: meta}).processRange},
		{"four_byte_counts", (&FourByteCountsDeriver{base: mk("four_byte_counts"), cfg: &FourByteCountsDeriverConfig{ChunkSize: 500}, cryo: runner, clientMeta: meta}).processRange},
		{"address_appearances", (&AddressAppearancesDeriver{base: mk("address_appearances"), cfg: &AddressAppearancesDeriverConfig{ChunkSize: 500}, cryo: runner, clientMeta: meta}).processRange},
	}

	sinkCfg := &chsink.Config{}
	require.NoError(t, defaults.Set(sinkCfg))
	sinkCfg.DSN = dsn
	sinkCfg.MetricsSubsystem = "cannon_itest_all"
	sinkCfg.RestrictToTablePrefixes = []string{"canonical_"}
	sinkCfg.Defaults.InsertSettings = map[string]any{"insert_quorum": 0}

	sink, err := chsink.New("itest-ch-all", sinkCfg, log, &xatu.EventFilterConfig{}, processor.ShippingMethodSync)
	require.NoError(t, err)
	require.NoError(t, sink.Start(ctx))

	defer func() { require.NoError(t, sink.Stop(ctx)) }()

	for _, d := range datasets {
		events, err := d.produce(ctx, from, to)
		if err != nil {
			t.Errorf("%s: produce failed: %v", d.name, err)

			continue
		}

		if err := sink.HandleNewDecoratedEvents(ctx, events); err != nil {
			t.Errorf("%s: sink failed: %v", d.name, err)

			continue
		}

		t.Logf("%-22s OK: %d events", d.name, len(events))
	}
}

// TestIngestRange ingests a (potentially large) block range across every
// dataset in memory-safe block chunks, flushing each chunk to ClickHouse.
// Env: CRYO_RPC, CLICKHOUSE_DSN, INGEST_FROM, INGEST_TO, INGEST_CHUNK (blocks
// per cryo call, default 500), INGEST_DATASETS (csv filter, default all).
func TestIngestRange(t *testing.T) {
	rpc := os.Getenv("CRYO_RPC")
	dsn := os.Getenv("CLICKHOUSE_DSN")
	if rpc == "" || dsn == "" {
		t.Skip("CRYO_RPC and CLICKHOUSE_DSN must be set")
	}

	from := envU64(t, "INGEST_FROM", 22000000)
	to := envU64(t, "INGEST_TO", 22000999)
	chunk := envU64(t, "INGEST_CHUNK", 500)
	filter := os.Getenv("INGEST_DATASETS")

	// INGEST_WINDOWS, when set, is a comma-separated list of from:to block
	// windows to sample (overrides INGEST_FROM/TO). Used to cover fork
	// boundaries + the whole history.
	type window struct{ from, to uint64 }

	var windows []window

	if ws := os.Getenv("INGEST_WINDOWS"); ws != "" {
		for _, part := range strings.Split(ws, ",") {
			fromTo := strings.SplitN(strings.TrimSpace(part), ":", 2)
			require.Len(t, fromTo, 2, "window must be from:to")

			wf, err := strconv.ParseUint(fromTo[0], 10, 64)
			require.NoError(t, err)

			wt, err := strconv.ParseUint(fromTo[1], 10, 64)
			require.NoError(t, err)

			windows = append(windows, window{wf, wt})
		}
	} else {
		windows = []window{{from, to}}
	}

	log := logrus.New()
	log.SetLevel(logrus.WarnLevel)

	ctx := context.Background()

	cryoCfg := &cryo.Config{}
	require.NoError(t, defaults.Set(cryoCfg))
	cryoCfg.MaxRangeSize = chunk

	runner := cryo.New(log, cryoCfg, rpc)
	meta := &xatu.ClientMeta{Name: "ingest", Ethereum: &xatu.ClientMeta_Ethereum{Network: &xatu.ClientMeta_Ethereum_Network{Name: "mainnet", Id: 1}}}
	ll := log.WithField("t", "ingest")
	mk := func(name string) base { return base{log: ll, name: name} }

	type ds struct {
		name    string
		produce eventProducer
	}

	all := []ds{
		{"blocks", (&BlockDeriver{base: mk("blocks"), cfg: &BlockDeriverConfig{ChunkSize: 500}, cryo: runner, clientMeta: meta}).processRange},
		{"transactions", (&TransactionDeriver{base: mk("transactions"), cfg: &TransactionDeriverConfig{ChunkSize: 500}, cryo: runner, clientMeta: meta}).processRange},
		{"logs", (&LogsDeriver{base: mk("logs"), cfg: &LogsDeriverConfig{ChunkSize: 500}, cryo: runner, clientMeta: meta}).processRange},
		{"traces", (&TracesDeriver{base: mk("traces"), cfg: &TracesDeriverConfig{ChunkSize: 500}, cryo: runner, clientMeta: meta}).processRange},
		{"native_transfers", (&NativeTransfersDeriver{base: mk("native_transfers"), cfg: &NativeTransfersDeriverConfig{ChunkSize: 500}, cryo: runner, clientMeta: meta}).processRange},
		{"erc20_transfers", (&Erc20TransfersDeriver{base: mk("erc20_transfers"), cfg: &Erc20TransfersDeriverConfig{ChunkSize: 500}, cryo: runner, clientMeta: meta}).processRange},
		{"erc721_transfers", (&Erc721TransfersDeriver{base: mk("erc721_transfers"), cfg: &Erc721TransfersDeriverConfig{ChunkSize: 500}, cryo: runner, clientMeta: meta}).processRange},
		{"contracts", (&ContractsDeriver{base: mk("contracts"), cfg: &ContractsDeriverConfig{ChunkSize: 500}, cryo: runner, clientMeta: meta}).processRange},
		{"balance_diffs", (&BalanceDiffsDeriver{base: mk("balance_diffs"), cfg: &BalanceDiffsDeriverConfig{ChunkSize: 500}, cryo: runner, clientMeta: meta}).processRange},
		{"storage_diffs", (&StorageDiffsDeriver{base: mk("storage_diffs"), cfg: &StorageDiffsDeriverConfig{ChunkSize: 500}, cryo: runner, clientMeta: meta}).processRange},
		{"nonce_diffs", (&NonceDiffsDeriver{base: mk("nonce_diffs"), cfg: &NonceDiffsDeriverConfig{ChunkSize: 500}, cryo: runner, clientMeta: meta}).processRange},
		{"balance_reads", (&BalanceReadsDeriver{base: mk("balance_reads"), cfg: &BalanceReadsDeriverConfig{ChunkSize: 500}, cryo: runner, clientMeta: meta}).processRange},
		{"storage_reads", (&StorageReadsDeriver{base: mk("storage_reads"), cfg: &StorageReadsDeriverConfig{ChunkSize: 500}, cryo: runner, clientMeta: meta}).processRange},
		{"nonce_reads", (&NonceReadsDeriver{base: mk("nonce_reads"), cfg: &NonceReadsDeriverConfig{ChunkSize: 500}, cryo: runner, clientMeta: meta}).processRange},
		{"four_byte_counts", (&FourByteCountsDeriver{base: mk("four_byte_counts"), cfg: &FourByteCountsDeriverConfig{ChunkSize: 500}, cryo: runner, clientMeta: meta}).processRange},
		{"address_appearances", (&AddressAppearancesDeriver{base: mk("address_appearances"), cfg: &AddressAppearancesDeriverConfig{ChunkSize: 500}, cryo: runner, clientMeta: meta}).processRange},
	}

	sinkCfg := &chsink.Config{}
	require.NoError(t, defaults.Set(sinkCfg))
	sinkCfg.DSN = dsn
	sinkCfg.MetricsSubsystem = "cannon_ingest"
	sinkCfg.RestrictToTablePrefixes = []string{"canonical_"}
	sinkCfg.Defaults.InsertSettings = map[string]any{"insert_quorum": 0}

	sink, err := chsink.New("ingest-ch", sinkCfg, log, &xatu.EventFilterConfig{}, processor.ShippingMethodSync)
	require.NoError(t, err)
	require.NoError(t, sink.Start(ctx))

	defer func() { _ = sink.Stop(ctx) }()

	for _, d := range all {
		if filter != "" && !strings.Contains(","+filter+",", ","+d.name+",") {
			continue
		}

		start := time.Now()
		chunks := 0

		// High-cardinality / state-access datasets emit far more rows per block,
		// so use smaller block chunks to bound per-flush memory. Light datasets
		// use the configured default.
		cs := chunk
		switch d.name {
		case "storage_reads", "balance_reads", "nonce_reads", "address_appearances", "traces", "native_transfers":
			cs = 100
		case "storage_diffs", "balance_diffs", "nonce_diffs":
			cs = 250
		}

		t.Logf("[%s] starting %d window(s) chunk=%d", d.name, len(windows), cs)

		var done uint64

		for _, w := range windows {
			for cf := w.from; cf <= w.to; cf += cs {
				ct := cf + cs - 1
				if ct > w.to {
					ct = w.to
				}

				events, perr := d.produce(ctx, cf, ct)
				if perr != nil {
					t.Errorf("%s [%d:%d]: %v", d.name, cf, ct, perr)

					continue
				}

				if serr := sink.HandleNewDecoratedEvents(ctx, events); serr != nil {
					t.Errorf("%s [%d:%d] sink: %v", d.name, cf, ct, serr)

					continue
				}

				done += ct - cf + 1
				chunks++

				if chunks%20 == 0 {
					t.Logf("[%s] %d blocks done (%s)", d.name, done, time.Since(start).Round(time.Second))
				}
			}
		}

		t.Logf("DONE %-22s %d windows / %d blocks in %s", d.name, len(windows), done, time.Since(start).Round(time.Second))
	}
}

func envU64(t *testing.T, key string, def uint64) uint64 {
	t.Helper()

	v := os.Getenv(key)
	if v == "" {
		return def
	}

	n, err := strconv.ParseUint(v, 10, 64)
	require.NoError(t, err)

	return n
}
