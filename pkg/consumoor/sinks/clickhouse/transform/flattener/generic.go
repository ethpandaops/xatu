package flattener

import (
	"fmt"
	"hash/fnv"
	"strconv"
	"strings"
	"time"

	"github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/metadata"
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

// GenericFlattener provides proto-driven flattening for one table/event set.
type GenericFlattener struct {
	eventNames []xatu.Event_Name
	tableName  TableName
	steps      []GenericStep
	should     EventPredicate
	rowMutator RowMutator
}

// NewGenericFlattenerWithSteps creates a generic proto-driven flattener
// with an explicit stage pipeline.
func NewGenericFlattenerWithSteps(
	table TableName,
	events []xatu.Event_Name,
	steps []GenericStep,
	predicate EventPredicate,
	mutator RowMutator,
) *GenericFlattener {
	return &GenericFlattener{
		eventNames: append([]xatu.Event_Name(nil), events...),
		tableName:  table,
		steps:      append([]GenericStep(nil), steps...),
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

	for i, step := range f.steps {
		if step == nil {
			continue
		}

		if err := step(event, meta, row); err != nil {
			return nil, fmt.Errorf("flatten step %d: %w", i, err)
		}
	}

	if f.rowMutator != nil {
		return f.rowMutator(event, meta, row)
	}

	return []map[string]any{row}, nil
}

// AddCommonMetadataFields copies shared metadata fields onto each row.
func AddCommonMetadataFields(_ *xatu.DecoratedEvent, meta *metadata.CommonMetadata, row map[string]any) error {
	if meta != nil {
		meta.CopyTo(row)
	}

	return nil
}

// AddRuntimeColumns appends writer/runtime columns such as updated_date_time,
// unique_key, and event_date_time.
func AddRuntimeColumns(event *xatu.DecoratedEvent, _ *metadata.CommonMetadata, row map[string]any) error {
	row["updated_date_time"] = time.Now().Unix()
	row["unique_key"] = hashKey(event.GetEvent().GetId())

	if ts := event.GetEvent().GetDateTime(); ts != nil {
		row["event_date_time"] = ts.AsTime().UnixMilli()
	}

	return nil
}

// FlattenEventDataFields flattens the event protobuf payload into row columns.
func FlattenEventDataFields(event *xatu.DecoratedEvent, _ *metadata.CommonMetadata, row map[string]any) error {
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

// FlattenClientAdditionalDataFields flattens client additional_data oneof fields.
func FlattenClientAdditionalDataFields(event *xatu.DecoratedEvent, _ *metadata.CommonMetadata, row map[string]any) error {
	if event.GetMeta() == nil || event.GetMeta().GetClient() == nil {
		return nil
	}

	client := event.GetMeta().GetClient().ProtoReflect()

	oneof := findAdditionalDataOneof(client)
	if oneof == nil {
		return nil
	}

	field := client.WhichOneof(oneof)
	if field == nil || field.Kind() != protoreflect.MessageKind {
		return nil
	}

	flattenProtoMessage("", client.Get(field).Message(), row)

	return nil
}

// FlattenServerAdditionalDataFields flattens server additional_data oneof fields.
func FlattenServerAdditionalDataFields(event *xatu.DecoratedEvent, _ *metadata.CommonMetadata, row map[string]any) error {
	if event.GetMeta() == nil || event.GetMeta().GetServer() == nil {
		return nil
	}

	server := event.GetMeta().GetServer().ProtoReflect()

	oneof := findAdditionalDataOneof(server)
	if oneof == nil {
		return nil
	}

	field := server.WhichOneof(oneof)
	if field == nil || field.Kind() != protoreflect.MessageKind {
		return nil
	}

	flattenProtoMessage("", server.Get(field).Message(), row)

	return nil
}

// CopyFieldIfMissing copies sourceField into targetField if targetField is absent.
func CopyFieldIfMissing(targetField, sourceField string) GenericStep {
	return CopyFieldsIfMissing(map[string]string{
		targetField: sourceField,
	})
}

// CopyFieldsIfMissing copies each source field into its target field when the
// target does not already exist.
func CopyFieldsIfMissing(mappings map[string]string) GenericStep {
	local := cloneAliases(mappings)

	return func(_ *xatu.DecoratedEvent, _ *metadata.CommonMetadata, row map[string]any) error {
		for targetField, sourceField := range local {
			if targetField == "" || sourceField == "" {
				continue
			}

			if _, exists := row[targetField]; exists {
				continue
			}

			if value, ok := row[sourceField]; ok {
				row[targetField] = value
			}
		}

		return nil
	}
}

// ApplyExplicitAliases applies route-specific alias remapping.
func ApplyExplicitAliases(aliases map[string]string) GenericStep {
	local := cloneAliases(aliases)

	return func(_ *xatu.DecoratedEvent, _ *metadata.CommonMetadata, row map[string]any) error {
		applyAliases(row, local)

		return nil
	}
}

// NormalizeDateTimeValues normalizes date/time fields for ClickHouse output.
func NormalizeDateTimeValues(_ *xatu.DecoratedEvent, _ *metadata.CommonMetadata, row map[string]any) error {
	normalizeDateTimeColumns(row)

	return nil
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

func hashKey(s string) uint64 {
	h := fnv.New64a()
	_, _ = h.Write([]byte(s))

	return h.Sum64()
}
