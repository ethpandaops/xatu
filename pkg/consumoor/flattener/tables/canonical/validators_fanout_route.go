package canonical

import (
	"time"

	"github.com/ethpandaops/xatu/pkg/consumoor/flattener"
	"github.com/ethpandaops/xatu/pkg/consumoor/metadata"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

type validatorsFanoutKind string

const (
	validatorsFanoutKindValidators           validatorsFanoutKind = "validators"
	validatorsFanoutKindPubkeys              validatorsFanoutKind = "pubkeys"
	validatorsFanoutKindWithdrawalCredential validatorsFanoutKind = "withdrawal_credentials"
)

type validatorsFanoutRoute struct {
	table flattener.TableName
	kind  validatorsFanoutKind
}

func newValidatorsFanoutRoute(table flattener.TableName, kind validatorsFanoutKind) flattener.Route {
	return &validatorsFanoutRoute{table: table, kind: kind}
}

func (r *validatorsFanoutRoute) EventNames() []xatu.Event_Name {
	return []xatu.Event_Name{xatu.Event_BEACON_API_ETH_V1_BEACON_VALIDATORS}
}

func (r *validatorsFanoutRoute) TableName() string {
	return string(r.table)
}

func (r *validatorsFanoutRoute) ShouldProcess(_ *xatu.DecoratedEvent) bool {
	return true
}

func (r *validatorsFanoutRoute) Flatten(event *xatu.DecoratedEvent, meta *metadata.CommonMetadata) ([]map[string]any, error) {
	if event == nil || event.GetEvent() == nil || event.GetEthV1Validators() == nil {
		return nil, nil
	}

	var (
		epoch      any
		epochStart any
	)

	if client := event.GetMeta().GetClient(); client != nil {
		if extra := client.GetEthV1Validators(); extra != nil {
			if epochData := extra.GetEpoch(); epochData != nil {
				if number := epochData.GetNumber(); number != nil {
					epoch = number.GetValue()
				}

				if start := epochData.GetStartDateTime(); start != nil {
					epochStart = start.AsTime().Unix()
				}
			}
		}
	}

	base := make(map[string]any, 40)
	if meta != nil {
		meta.CopyTo(base)
	}

	base["updated_date_time"] = time.Now().Unix()
	base["epoch"] = epoch
	base["epoch_start_date_time"] = epochStart

	validators := event.GetEthV1Validators().GetValidators()
	rows := make([]map[string]any, 0, len(validators))

	for _, validator := range validators {
		if validator == nil {
			continue
		}

		row := cloneRow(base)

		if validator.GetIndex() != nil {
			row["index"] = validator.GetIndex().GetValue()
		}

		switch r.kind {
		case validatorsFanoutKindValidators:
			if validator.GetStatus() != nil {
				row["status"] = validator.GetStatus().GetValue()
			}

			if data := validator.GetData(); data != nil {
				if data.GetSlashed() != nil {
					row["slashed"] = data.GetSlashed().GetValue()
				}

				if data.GetEffectiveBalance() != nil && data.GetEffectiveBalance().GetValue() != 0 {
					row["effective_balance"] = data.GetEffectiveBalance().GetValue()
				}

				setOptionalEpoch(row, "activation_epoch", data.GetActivationEpoch())
				setOptionalEpoch(row, "activation_eligibility_epoch", data.GetActivationEligibilityEpoch())
				setOptionalEpoch(row, "exit_epoch", data.GetExitEpoch())
				setOptionalEpoch(row, "withdrawable_epoch", data.GetWithdrawableEpoch())
			}

			if validator.GetBalance() != nil && validator.GetBalance().GetValue() != 0 {
				row["balance"] = validator.GetBalance().GetValue()
			}
		case validatorsFanoutKindPubkeys:
			if data := validator.GetData(); data != nil && data.GetPubkey() != nil {
				row["pubkey"] = data.GetPubkey().GetValue()
			}
		case validatorsFanoutKindWithdrawalCredential:
			if data := validator.GetData(); data != nil && data.GetWithdrawalCredentials() != nil {
				row["withdrawal_credentials"] = data.GetWithdrawalCredentials().GetValue()
			}
		}

		rows = append(rows, row)
	}

	return rows, nil
}

func setOptionalEpoch(row map[string]any, key string, wrapped interface{ GetValue() uint64 }) {
	if wrapped == nil {
		return
	}

	value := wrapped.GetValue()
	if value == 0 || value == ^uint64(0) {
		return
	}

	row[key] = value
}

func cloneRow(row map[string]any) map[string]any {
	clone := make(map[string]any, len(row)+8)
	for k, v := range row {
		clone[k] = v
	}

	return clone
}
