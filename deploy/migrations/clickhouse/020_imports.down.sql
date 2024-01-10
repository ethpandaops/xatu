DROP TABLE IF EXISTS imported_sources on cluster '{cluster}' SYNC;
DROP TABLE IF EXISTS imported_sources_local on cluster '{cluster}' SYNC;
DROP TABLE IF EXISTS mempool_dumpster_transaction on cluster '{cluster}' SYNC;
DROP TABLE IF EXISTS mempool_dumpster_transaction_local on cluster '{cluster}' SYNC;
DROP TABLE IF EXISTS mempool_dumpster_transaction_source on cluster '{cluster}' SYNC;
DROP TABLE IF EXISTS mempool_dumpster_transaction_source_local on cluster '{cluster}' SYNC;
DROP TABLE IF EXISTS block_native_mempool_transaction on cluster '{cluster}' SYNC;
DROP TABLE IF EXISTS block_native_mempool_transaction_local on cluster '{cluster}' SYNC;
