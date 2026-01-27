package sentry

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/attestantio/go-eth2-client/api"
	"github.com/attestantio/go-eth2-client/spec"
	"github.com/attestantio/go-eth2-client/spec/deneb"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	xatuethv1 "github.com/ethpandaops/xatu/pkg/proto/eth/v1"
	"github.com/ethpandaops/xatu/pkg/sentry/ethereum"
	v1 "github.com/ethpandaops/xatu/pkg/sentry/event/beacon/eth/v1"
	"github.com/sirupsen/logrus"
)

// fetchAndEmitBeaconBlobs fetches the block and emits beacon blob events for each KZG commitment.
// This captures blob metadata with versioned_hash for joining with execution_engine_get_blobs events.
func (s *Sentry) fetchAndEmitBeaconBlobs(ctx context.Context, blockRoot string, slot phase0.Slot) {
	block, err := s.beacon.Node().FetchBlock(ctx, blockRoot)
	if err != nil {
		var apiErr *api.Error
		if errors.As(err, &apiErr) {
			switch apiErr.StatusCode {
			case 404:
				s.log.WithFields(logrus.Fields{
					"block_root": blockRoot,
					"slot":       slot,
				}).Debug("Block not found for beacon blob extraction")

				return
			case 503:
				s.log.WithFields(logrus.Fields{
					"block_root": blockRoot,
					"slot":       slot,
				}).Warn("Beacon node is syncing, cannot fetch block for beacon blob extraction")

				return
			}
		}

		s.log.WithError(err).WithFields(logrus.Fields{
			"block_root": blockRoot,
			"slot":       slot,
		}).Error("Failed to fetch block for beacon blob extraction")

		return
	}

	if block == nil {
		s.log.WithFields(logrus.Fields{
			"block_root": blockRoot,
			"slot":       slot,
		}).Debug("Block is nil for beacon blob extraction")

		return
	}

	// Extract KZG commitments from the block body based on the block version
	kzgCommitments, err := extractKZGCommitments(block)
	if err != nil {
		s.log.WithFields(logrus.Fields{
			"block_root": blockRoot,
			"slot":       slot,
			"version":    block.Version.String(),
		}).Debug("Block version does not support KZG commitments")

		return
	}

	if len(kzgCommitments) == 0 {
		s.log.WithFields(logrus.Fields{
			"block_root": blockRoot,
			"slot":       slot,
		}).Debug("No KZG commitments found in block")

		return
	}

	// Extract block metadata
	proposerIndex, err := block.ProposerIndex()
	if err != nil {
		s.log.WithError(err).WithFields(logrus.Fields{
			"block_root": blockRoot,
			"slot":       slot,
		}).Error("Failed to get proposer index from block")

		return
	}

	parentRoot, err := block.ParentRoot()
	if err != nil {
		s.log.WithError(err).WithFields(logrus.Fields{
			"block_root": blockRoot,
			"slot":       slot,
		}).Error("Failed to get parent root from block")

		return
	}

	now := time.Now().Add(s.clockDrift)

	blockRootParsed, err := xatuethv1.StringToRoot(blockRoot)
	if err != nil {
		s.log.WithError(err).WithFields(logrus.Fields{
			"block_root": blockRoot,
			"slot":       slot,
		}).Error("Failed to parse block root for beacon blob extraction")

		return
	}

	for i, commitment := range kzgCommitments {
		versionedHash := ethereum.ConvertKzgCommitmentToVersionedHash(commitment[:])

		blobData := &v1.BlobData{
			Slot:            slot,
			Index:           uint64(i), //nolint:gosec // G115: blob index is always small
			BlockRoot:       blockRootParsed,
			BlockParentRoot: parentRoot,
			ProposerIndex:   proposerIndex,
			KZGCommitment:   commitment,
			VersionedHash:   versionedHash,
		}

		meta, err := s.createNewClientMeta(ctx)
		if err != nil {
			s.log.WithError(err).Error("Failed to create client meta for beacon blob")

			continue
		}

		event := v1.NewBeaconBlob(
			s.log,
			blobData,
			now,
			s.beacon,
			s.duplicateCache.BeaconEthV1BeaconBlob,
			meta,
		)

		ignore, err := event.ShouldIgnore(ctx)
		if err != nil {
			s.log.WithError(err).WithFields(logrus.Fields{
				"block_root": blockRoot,
				"slot":       slot,
				"index":      i,
			}).Error("Failed to check if beacon blob event should be ignored")

			continue
		}

		if ignore {
			continue
		}

		decoratedEvent, err := event.Decorate(ctx)
		if err != nil {
			s.log.WithError(err).WithFields(logrus.Fields{
				"block_root": blockRoot,
				"slot":       slot,
				"index":      i,
			}).Error("Failed to decorate beacon blob event")

			continue
		}

		if err := s.handleNewDecoratedEvent(ctx, decoratedEvent); err != nil {
			s.log.WithError(err).WithFields(logrus.Fields{
				"block_root": blockRoot,
				"slot":       slot,
				"index":      i,
			}).Error("Failed to handle beacon blob event")

			continue
		}
	}

	s.log.WithFields(logrus.Fields{
		"block_root": blockRoot,
		"slot":       slot,
		"count":      len(kzgCommitments),
	}).Debug("Fetched and emitted beacon blobs for block")
}

// extractKZGCommitments extracts KZG commitments from a block based on its version.
func extractKZGCommitments(block *spec.VersionedSignedBeaconBlock) ([]deneb.KZGCommitment, error) {
	switch block.Version {
	case spec.DataVersionDeneb:
		if block.Deneb != nil && block.Deneb.Message != nil && block.Deneb.Message.Body != nil {
			return block.Deneb.Message.Body.BlobKZGCommitments, nil
		}
	case spec.DataVersionElectra:
		if block.Electra != nil && block.Electra.Message != nil && block.Electra.Message.Body != nil {
			return block.Electra.Message.Body.BlobKZGCommitments, nil
		}
	case spec.DataVersionFulu:
		if block.Fulu != nil && block.Fulu.Message != nil && block.Fulu.Message.Body != nil {
			return block.Fulu.Message.Body.BlobKZGCommitments, nil
		}
	}

	return nil, fmt.Errorf("block version %s does not support KZG commitments", block.Version)
}
