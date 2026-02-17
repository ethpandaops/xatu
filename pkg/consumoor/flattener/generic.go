package flattener

import (
	"fmt"
	"hash/fnv"
	"strconv"
	"strings"
	"time"

	"github.com/ethpandaops/xatu/pkg/consumoor/metadata"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// EventPredicate determines whether a route should process a specific event.
type EventPredicate func(event *xatu.DecoratedEvent) bool

// RowMutator can amend or fan-out rows after generic flattening.
type RowMutator func(event *xatu.DecoratedEvent, meta *metadata.CommonMetadata, row map[string]any) ([]map[string]any, error)

// GenericStep is one composable flattening operation.
type GenericStep func(event *xatu.DecoratedEvent, meta *metadata.CommonMetadata, row map[string]any) error

type compiledGenericStep struct {
	name string
	run  GenericStep
}

// GenericFlattener provides proto-driven flattening for one table/event set.
type GenericFlattener struct {
	eventNames []xatu.Event_Name
	tableName  TableName
	steps      []compiledGenericStep
	should     EventPredicate
	rowMutator RowMutator
}

// NewGenericFlattenerWithSteps creates a generic proto-driven flattener
// with an explicit stage pipeline.
func NewGenericFlattenerWithSteps(
	table TableName,
	events []xatu.Event_Name,
	steps []compiledGenericStep,
	predicate EventPredicate,
	mutator RowMutator,
) *GenericFlattener {
	return &GenericFlattener{
		eventNames: append([]xatu.Event_Name(nil), events...),
		tableName:  table,
		steps:      append([]compiledGenericStep(nil), steps...),
		should:     predicate,
		rowMutator: mutator,
	}
}

func (f *GenericFlattener) EventNames() []xatu.Event_Name {
	return f.eventNames
}

func (f *GenericFlattener) TableName() string {
	return string(f.tableName)
}

func (f *GenericFlattener) ShouldProcess(event *xatu.DecoratedEvent) bool {
	if f.should == nil {
		return true
	}

	return f.should(event)
}

func (f *GenericFlattener) Flatten(event *xatu.DecoratedEvent, meta *metadata.CommonMetadata) ([]map[string]any, error) {
	if event == nil || event.GetEvent() == nil {
		return nil, fmt.Errorf("nil event")
	}

	row := make(map[string]any, 96)

	for _, step := range f.steps {
		if step.run == nil {
			continue
		}

		if err := step.run(event, meta, row); err != nil {
			return nil, fmt.Errorf("flatten step %s: %w", step.name, err)
		}
	}

	if f.rowMutator != nil {
		return f.rowMutator(event, meta, row)
	}

	return []map[string]any{row}, nil
}

func commonMetadataStep() GenericStep {
	return func(_ *xatu.DecoratedEvent, meta *metadata.CommonMetadata, row map[string]any) error {
		if meta != nil {
			meta.CopyTo(row)
		}

		return nil
	}
}

func runtimeColumnsStep() GenericStep {
	return func(event *xatu.DecoratedEvent, _ *metadata.CommonMetadata, row map[string]any) error {
		row["updated_date_time"] = time.Now().Unix()
		row["unique_key"] = hashKey(event.GetEvent().GetId())

		if ts := event.GetEvent().GetDateTime(); ts != nil {
			row["event_date_time"] = ts.AsTime().UnixMilli()
		}

		return nil
	}
}

func eventDataStep() GenericStep {
	return func(event *xatu.DecoratedEvent, _ *metadata.CommonMetadata, row map[string]any) error {
		return flattenEventData(event, row)
	}
}

func clientAdditionalDataStep() GenericStep {
	return func(event *xatu.DecoratedEvent, _ *metadata.CommonMetadata, row map[string]any) error {
		flattenClientAdditionalData(event, row)

		return nil
	}
}

func serverAdditionalDataStep() GenericStep {
	return func(event *xatu.DecoratedEvent, _ *metadata.CommonMetadata, row map[string]any) error {
		flattenServerAdditionalData(event, row)

		return nil
	}
}

func tableAliasesStep(table TableName) GenericStep {
	tableName := string(table)

	return func(_ *xatu.DecoratedEvent, _ *metadata.CommonMetadata, row map[string]any) error {
		applyTableAliases(tableName, row)

		return nil
	}
}

func routeAliasesStep(aliases map[string]string) GenericStep {
	local := cloneAliases(aliases)

	return func(_ *xatu.DecoratedEvent, _ *metadata.CommonMetadata, row map[string]any) error {
		applyAliases(row, local)

		return nil
	}
}

func normalizeDateTimesStep() GenericStep {
	return func(_ *xatu.DecoratedEvent, _ *metadata.CommonMetadata, row map[string]any) error {
		normalizeDateTimeColumns(row)

		return nil
	}
}

func commonEnrichmentStep() GenericStep {
	return func(event *xatu.DecoratedEvent, _ *metadata.CommonMetadata, row map[string]any) error {
		enrichCommon(row, event.GetEvent().GetName())

		return nil
	}
}

func flattenEventData(event *xatu.DecoratedEvent, row map[string]any) error {
	msg := event.ProtoReflect()

	oneof := msg.Descriptor().Oneofs().ByName("data")
	if oneof == nil {
		return nil
	}

	field := msg.WhichOneof(oneof)
	if field == nil {
		return nil
	}

	value := msg.Get(field)

	if field.Kind() == protoreflect.MessageKind {
		flattenProtoMessage("", value.Message(), row)
		promoteMessageOneof(value.Message(), row)

		return nil
	}

	row[string(field.Name())] = scalarFromValue(field, value)

	return nil
}

func flattenClientAdditionalData(event *xatu.DecoratedEvent, row map[string]any) {
	if event.GetMeta() == nil || event.GetMeta().GetClient() == nil {
		return
	}

	client := event.GetMeta().GetClient().ProtoReflect()

	oneof := findAdditionalDataOneof(client)
	if oneof == nil {
		return
	}

	field := client.WhichOneof(oneof)
	if field == nil || field.Kind() != protoreflect.MessageKind {
		return
	}

	flattenProtoMessage("", client.Get(field).Message(), row)
}

func flattenServerAdditionalData(event *xatu.DecoratedEvent, row map[string]any) {
	if event.GetMeta() == nil || event.GetMeta().GetServer() == nil {
		return
	}

	server := event.GetMeta().GetServer().ProtoReflect()

	oneof := findAdditionalDataOneof(server)
	if oneof == nil {
		return
	}

	field := server.WhichOneof(oneof)
	if field == nil || field.Kind() != protoreflect.MessageKind {
		return
	}

	flattenProtoMessage("", server.Get(field).Message(), row)
}

func findAdditionalDataOneof(msg protoreflect.Message) protoreflect.OneofDescriptor {
	oneofs := msg.Descriptor().Oneofs()

	for i := 0; i < oneofs.Len(); i++ {
		oneof := oneofs.Get(i)

		name := strings.ToLower(string(oneof.Name()))
		if strings.Contains(name, "additional") {
			return oneof
		}
	}

	if oneofs.Len() > 0 {
		return oneofs.Get(0)
	}

	return nil
}

func promoteMessageOneof(msg protoreflect.Message, row map[string]any) {
	oneofs := msg.Descriptor().Oneofs()

	for i := 0; i < oneofs.Len(); i++ {
		oneof := oneofs.Get(i)
		if strings.ToLower(string(oneof.Name())) != "message" {
			continue
		}

		field := msg.WhichOneof(oneof)
		if field == nil || field.Kind() != protoreflect.MessageKind {
			continue
		}

		child := make(map[string]any, 64)
		flattenProtoMessage("", msg.Get(field).Message(), child)

		for k, v := range child {
			if _, ok := row[k]; !ok {
				row[k] = v
			}
		}
	}
}

func flattenProtoMessage(prefix string, msg protoreflect.Message, out map[string]any) {
	if !msg.IsValid() {
		return
	}

	fields := msg.Descriptor().Fields()

	for i := 0; i < fields.Len(); i++ {
		fd := fields.Get(i)

		if fd.ContainingOneof() != nil && !fd.ContainingOneof().IsSynthetic() && !msg.Has(fd) {
			continue
		}

		if !msg.Has(fd) && !fd.IsList() && !fd.IsMap() {
			continue
		}

		name := string(fd.Name())
		key := joinKey(prefix, name)
		value := msg.Get(fd)

		switch {
		case fd.IsMap():
			out[key] = mapFromValue(value.Map(), fd.MapValue())
		case fd.IsList():
			out[key] = listFromValue(value.List(), fd)
		case fd.Kind() == protoreflect.MessageKind || fd.Kind() == protoreflect.GroupKind:
			if unwrapped, ok := unwrapSpecialMessage(fd.Message(), value.Message()); ok {
				out[key] = unwrapped

				continue
			}

			flattenProtoMessage(key, value.Message(), out)
		default:
			out[key] = scalarFromValue(fd, value)
		}
	}
}

func mapFromValue(m protoreflect.Map, valueDesc protoreflect.FieldDescriptor) map[string]any {
	result := make(map[string]any, m.Len())
	m.Range(func(k protoreflect.MapKey, v protoreflect.Value) bool {
		key := k.String()

		if valueDesc.Kind() == protoreflect.MessageKind {
			if unwrapped, ok := unwrapSpecialMessage(valueDesc.Message(), v.Message()); ok {
				result[key] = unwrapped

				return true
			}

			inner := make(map[string]any)
			flattenProtoMessage("", v.Message(), inner)
			result[key] = inner

			return true
		}

		result[key] = scalarFromValue(valueDesc, v)

		return true
	})

	return result
}

func listFromValue(list protoreflect.List, fd protoreflect.FieldDescriptor) any {
	if list.Len() == 0 {
		return []any{}
	}

	kind := fd.Kind()
	if kind == protoreflect.MessageKind {
		items := make([]any, 0, list.Len())
		for i := 0; i < list.Len(); i++ {
			item := list.Get(i).Message()
			if unwrapped, ok := unwrapSpecialMessage(fd.Message(), item); ok {
				items = append(items, unwrapped)

				continue
			}

			inner := make(map[string]any)
			flattenProtoMessage("", item, inner)
			items = append(items, inner)
		}

		return items
	}

	switch kind {
	case protoreflect.StringKind:
		items := make([]string, 0, list.Len())
		for i := 0; i < list.Len(); i++ {
			items = append(items, list.Get(i).String())
		}

		return items
	case protoreflect.BoolKind:
		items := make([]bool, 0, list.Len())
		for i := 0; i < list.Len(); i++ {
			items = append(items, list.Get(i).Bool())
		}

		return items
	case protoreflect.Uint32Kind, protoreflect.Uint64Kind, protoreflect.Fixed32Kind, protoreflect.Fixed64Kind:
		items := make([]uint64, 0, list.Len())
		for i := 0; i < list.Len(); i++ {
			items = append(items, list.Get(i).Uint())
		}

		return items
	case protoreflect.Int32Kind, protoreflect.Int64Kind, protoreflect.Sfixed32Kind, protoreflect.Sfixed64Kind,
		protoreflect.Sint32Kind, protoreflect.Sint64Kind:
		items := make([]int64, 0, list.Len())
		for i := 0; i < list.Len(); i++ {
			items = append(items, list.Get(i).Int())
		}

		return items
	case protoreflect.FloatKind, protoreflect.DoubleKind:
		items := make([]float64, 0, list.Len())
		for i := 0; i < list.Len(); i++ {
			items = append(items, list.Get(i).Float())
		}

		return items
	case protoreflect.EnumKind:
		items := make([]string, 0, list.Len())
		for i := 0; i < list.Len(); i++ {
			items = append(items, enumString(fd, list.Get(i).Enum()))
		}

		return items
	default:
		items := make([]any, 0, list.Len())
		for i := 0; i < list.Len(); i++ {
			items = append(items, scalarFromValue(fd, list.Get(i)))
		}

		return items
	}
}

func scalarFromValue(fd protoreflect.FieldDescriptor, value protoreflect.Value) any {
	switch fd.Kind() {
	case protoreflect.BoolKind:
		return value.Bool()
	case protoreflect.EnumKind:
		return enumString(fd, value.Enum())
	case protoreflect.Int32Kind, protoreflect.Int64Kind, protoreflect.Sfixed32Kind, protoreflect.Sfixed64Kind,
		protoreflect.Sint32Kind, protoreflect.Sint64Kind:
		return value.Int()
	case protoreflect.Uint32Kind, protoreflect.Uint64Kind, protoreflect.Fixed32Kind, protoreflect.Fixed64Kind:
		return value.Uint()
	case protoreflect.FloatKind, protoreflect.DoubleKind:
		return value.Float()
	case protoreflect.StringKind:
		return value.String()
	case protoreflect.BytesKind:
		return value.Bytes()
	default:
		return value.Interface()
	}
}

func enumString(fd protoreflect.FieldDescriptor, num protoreflect.EnumNumber) string {
	if fd.Enum() == nil {
		return strconv.FormatInt(int64(num), 10)
	}

	value := fd.Enum().Values().ByNumber(num)
	if value == nil {
		return strconv.FormatInt(int64(num), 10)
	}

	return string(value.Name())
}

func unwrapSpecialMessage(desc protoreflect.MessageDescriptor, msg protoreflect.Message) (any, bool) {
	fullName := string(desc.FullName())

	switch fullName {
	case "google.protobuf.Timestamp":
		if ts, ok := msg.Interface().(*timestamppb.Timestamp); ok {
			return ts.AsTime(), true
		}

		return nil, true
	case "google.protobuf.StringValue", "google.protobuf.BytesValue", "google.protobuf.BoolValue",
		"google.protobuf.UInt32Value", "google.protobuf.UInt64Value", "google.protobuf.Int32Value",
		"google.protobuf.Int64Value", "google.protobuf.FloatValue", "google.protobuf.DoubleValue":
		fields := msg.Descriptor().Fields()
		if fields.Len() == 0 {
			return nil, true
		}

		valueField := fields.ByName("value")
		if valueField == nil || !msg.Has(valueField) {
			return nil, true
		}

		return scalarFromValue(valueField, msg.Get(valueField)), true
	}

	return nil, false
}

func joinKey(prefix, key string) string {
	if prefix == "" {
		return key
	}

	return prefix + "_" + key
}

func normalizeDateTimeColumns(row map[string]any) {
	for k, v := range row {
		isEventDateTime := k == "event_date_time"
		isDateTimeSuffix := strings.HasSuffix(k, "_date_time")

		if !isEventDateTime && !isDateTimeSuffix {
			continue
		}

		t, ok := asTime(v)
		if !ok {
			continue
		}

		if isEventDateTime {
			row[k] = t.UnixMilli()

			continue
		}

		if isDateTimeSuffix {
			row[k] = t.Unix()
		}
	}
}

func asTime(v any) (time.Time, bool) {
	switch tv := v.(type) {
	case time.Time:
		return tv.UTC(), true
	case string:
		if tv == "" {
			return time.Time{}, false
		}

		t, err := time.Parse(time.RFC3339Nano, tv)
		if err != nil {
			return time.Time{}, false
		}

		return t.UTC(), true
	default:
		return time.Time{}, false
	}
}

func applyAliases(row map[string]any, aliases map[string]string) {
	for from, to := range aliases {
		if from == to {
			continue
		}

		value, ok := row[from]
		if !ok {
			continue
		}

		if to == "" {
			delete(row, from)

			continue
		}

		if _, exists := row[to]; !exists {
			row[to] = value
		}

		delete(row, from)
	}
}

func cloneAliases(aliases map[string]string) map[string]string {
	if len(aliases) == 0 {
		return nil
	}

	out := make(map[string]string, len(aliases))
	for from, to := range aliases {
		out[from] = to
	}

	return out
}

func applyTableAliases(table string, row map[string]any) {
	switch table {
	case "beacon_api_eth_v1_beacon_committee", "canonical_beacon_committee":
		setAlias(row, "committee_index", "index")
	case "beacon_api_eth_v1_events_blob_sidecar", "beacon_api_eth_v1_events_data_column_sidecar":
		setAlias(row, "column_index", "index")
	case "canonical_beacon_blob_sidecar":
		setAlias(row, "blob_index", "index")
	case "beacon_api_eth_v1_proposer_duty", "canonical_beacon_proposer_duty":
		setAlias(row, "proposer_validator_index", "validator_index")
		setAlias(row, "proposer_pubkey", "pubkey")
	case "canonical_beacon_block", "beacon_api_eth_v2_beacon_block", "beacon_api_eth_v3_validator_block":
		setAlias(row, "slot", "slot_number")
		setAlias(row, "epoch", "epoch_number")
		setAlias(row, "eth1_data_block_hash", "body_eth1_data_block_hash")
		setAlias(row, "eth1_data_deposit_root", "body_eth1_data_deposit_root")
		setAlias(row, "execution_payload_block_hash", "body_execution_payload_block_hash")
		setAlias(row, "execution_payload_block_number", "body_execution_payload_block_number")
		setAlias(row, "execution_payload_fee_recipient", "body_execution_payload_fee_recipient")
		setAlias(row, "execution_payload_base_fee_per_gas", "body_execution_payload_base_fee_per_gas")
		setAlias(row, "execution_payload_blob_gas_used", "body_execution_payload_blob_gas_used")
		setAlias(row, "execution_payload_excess_blob_gas", "body_execution_payload_excess_blob_gas")
		setAlias(row, "execution_payload_gas_limit", "body_execution_payload_gas_limit")
		setAlias(row, "execution_payload_gas_used", "body_execution_payload_gas_used")
		setAlias(row, "execution_payload_state_root", "body_execution_payload_state_root")
		setAlias(row, "execution_payload_parent_hash", "body_execution_payload_parent_hash")
		setAlias(row, "execution_payload_transactions_count", "transactions_count")
		setAlias(row, "execution_payload_transactions_total_bytes", "transactions_total_bytes")
		setAlias(row, "execution_payload_transactions_total_bytes_compressed", "transactions_total_bytes_compressed")
	case "canonical_beacon_block_proposer_slashing",
		"canonical_beacon_block_attester_slashing",
		"canonical_beacon_block_bls_to_execution_change",
		"canonical_beacon_block_execution_transaction",
		"canonical_beacon_block_voluntary_exit",
		"canonical_beacon_block_deposit",
		"canonical_beacon_block_withdrawal",
		"canonical_beacon_block_sync_aggregate",
		"canonical_beacon_elaborated_attestation":
		setAlias(row, "slot", "block_slot_number")
		setAlias(row, "epoch", "block_epoch_number")
		setAlias(row, "slot_start_date_time", "block_slot_start_date_time")
		setAlias(row, "epoch_start_date_time", "block_epoch_start_date_time")
		setAlias(row, "block_root", "block_root")
		setAlias(row, "block_version", "block_version")
	case "mempool_transaction":
		setAlias(row, "raw", "mempool_transaction")
		setAlias(row, "raw", "mempool_transaction_v2")
	}
}

func setAlias(row map[string]any, to, from string) {
	if _, exists := row[to]; exists {
		return
	}

	if value, ok := row[from]; ok {
		row[to] = value
	}
}

func enrichCommon(row map[string]any, eventName xatu.Event_Name) {
	if strings.HasPrefix(eventName.String(), "LIBP2P_TRACE_") {
		enrichLibP2P(row, eventName)
	}
}

func hashKey(s string) uint64 {
	h := fnv.New64a()
	_, _ = h.Write([]byte(s))

	return h.Sum64()
}

func copyRow(row map[string]any) map[string]any {
	clone := make(map[string]any, len(row)+8)
	for k, v := range row {
		clone[k] = v
	}

	return clone
}

func additionalDataPath(event *xatu.DecoratedEvent, path ...string) (any, bool) {
	msg, ok := additionalDataMessage(event)
	if !ok {
		return nil, false
	}

	if value, ok := messagePathValue(msg, path...); ok {
		return value, true
	}

	flat := messageToMap(msg)

	flatKey := strings.Join(path, "_")
	if value, ok := flat[flatKey]; ok {
		return value, true
	}

	// fallback for direct fields
	if len(path) == 1 {
		if value, ok := flat[path[0]]; ok {
			return value, true
		}
	}

	return nil, false
}

func additionalDataMessage(event *xatu.DecoratedEvent) (protoreflect.Message, bool) {
	if event == nil || event.GetMeta() == nil || event.GetMeta().GetClient() == nil {
		return nil, false
	}

	client := event.GetMeta().GetClient().ProtoReflect()

	oneof := findAdditionalDataOneof(client)
	if oneof == nil {
		return nil, false
	}

	field := client.WhichOneof(oneof)
	if field == nil || field.Kind() != protoreflect.MessageKind {
		return nil, false
	}

	return client.Get(field).Message(), true
}

func messagePathValue(msg protoreflect.Message, path ...string) (any, bool) {
	current := msg

	for i, segment := range path {
		fd := current.Descriptor().Fields().ByName(protoreflect.Name(segment))
		if fd == nil {
			return nil, false
		}

		if !current.Has(fd) && !fd.IsList() && !fd.IsMap() {
			return nil, false
		}

		value := current.Get(fd)
		last := i == len(path)-1

		if last {
			switch {
			case fd.IsMap():
				return mapFromValue(value.Map(), fd.MapValue()), true
			case fd.IsList():
				return listFromValue(value.List(), fd), true
			case fd.Kind() == protoreflect.MessageKind || fd.Kind() == protoreflect.GroupKind:
				if unwrapped, ok := unwrapSpecialMessage(fd.Message(), value.Message()); ok {
					return unwrapped, true
				}

				out := make(map[string]any, 16)
				flattenProtoMessage("", value.Message(), out)

				return out, true
			default:
				return scalarFromValue(fd, value), true
			}
		}

		if fd.Kind() != protoreflect.MessageKind && fd.Kind() != protoreflect.GroupKind {
			return nil, false
		}

		current = value.Message()
	}

	return nil, false
}

func messageToMap(msg protoreflect.Message) map[string]any {
	result := make(map[string]any, 32)
	flattenProtoMessage("", msg, result)

	return result
}

// ValidatorsFanoutFlattener fans out canonical validators events to three tables.
type ValidatorsFanoutFlattener struct {
	table TableName
	kind  ValidatorsFanoutKind
}

type ValidatorsFanoutKind string

const (
	ValidatorsFanoutKindValidators           ValidatorsFanoutKind = "validators"
	ValidatorsFanoutKindPubkeys              ValidatorsFanoutKind = "pubkeys"
	ValidatorsFanoutKindWithdrawalCredential ValidatorsFanoutKind = "withdrawal_credentials"
)

func NewValidatorsFanoutFlattener(table TableName, kind ValidatorsFanoutKind) *ValidatorsFanoutFlattener {
	return &ValidatorsFanoutFlattener{table: table, kind: kind}
}

func (f *ValidatorsFanoutFlattener) EventNames() []xatu.Event_Name {
	return []xatu.Event_Name{xatu.Event_BEACON_API_ETH_V1_BEACON_VALIDATORS}
}

func (f *ValidatorsFanoutFlattener) TableName() string {
	return string(f.table)
}

func (f *ValidatorsFanoutFlattener) ShouldProcess(_ *xatu.DecoratedEvent) bool {
	return true
}

func (f *ValidatorsFanoutFlattener) Flatten(event *xatu.DecoratedEvent, meta *metadata.CommonMetadata) ([]map[string]any, error) {
	if event == nil || event.GetEvent() == nil || event.GetEthV1Validators() == nil {
		return nil, nil
	}

	epochValue, _ := additionalDataPath(event, "epoch", "number")
	epochStartValue, _ := additionalDataPath(event, "epoch", "start_date_time")

	epoch := epochValue

	epochStart := epochStartValue
	if t, ok := asTime(epochStartValue); ok {
		epochStart = t.Unix()
	}

	base := make(map[string]any, 40)
	meta.CopyTo(base)
	base["updated_date_time"] = time.Now().Unix()
	base["epoch"] = epoch
	base["epoch_start_date_time"] = epochStart

	validators := event.GetEthV1Validators().GetValidators()
	rows := make([]map[string]any, 0, len(validators))

	for _, validator := range validators {
		if validator == nil {
			continue
		}

		row := copyRow(base)

		if validator.GetIndex() != nil {
			row["index"] = validator.GetIndex().GetValue()
		}

		switch f.kind {
		case ValidatorsFanoutKindValidators:
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
		case ValidatorsFanoutKindPubkeys:
			if data := validator.GetData(); data != nil && data.GetPubkey() != nil {
				row["pubkey"] = data.GetPubkey().GetValue()
			}
		case ValidatorsFanoutKindWithdrawalCredential:
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
