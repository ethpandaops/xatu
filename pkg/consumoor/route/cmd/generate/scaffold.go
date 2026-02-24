package main

import (
	"fmt"
	"os"
	"strings"
	"text/template"
)

// routeScaffoldTmpl is the template for a new route file that compiles but
// returns an error from FlattenTo, signaling that the developer must fill in
// the event names and payload logic.
var routeScaffoldTmpl = template.Must(template.New("route").Parse(`package {{.Package}}

import (
	"fmt"
	"time"

	"github.com/ethpandaops/xatu/pkg/consumoor/route"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

// TODO: Add the xatu.Event_* name(s) that route events to the {{.TableName}} table.
var {{.TypeName}}EventNames = []xatu.Event_Name{}

func init() {
	route, err := route.NewStaticRoute(
		{{.TypeName}}TableName,
		{{.TypeName}}EventNames,
		func() route.ColumnarBatch { return new{{.BatchName}}() },
	)
	if err != nil {
		route.RecordError(err)

		return
	}

	if err := route.Register(route); err != nil {
		route.RecordError(err)
	}
}

func (b *{{.BatchName}}) FlattenTo(
	event *xatu.DecoratedEvent,
) error {
	if event == nil || event.GetEvent() == nil {
		return nil
	}

	b.appendRuntime(event)
	b.appendMetadata(event)

	// TODO: Implement appendPayload to extract event-specific fields from the proto.
	if err := b.appendPayload(event); err != nil {
		return err
	}

	b.rows++

	return nil
}

func (b *{{.BatchName}}) appendRuntime(event *xatu.DecoratedEvent) {
	b.UpdatedDateTime.Append(time.Now())

	if ts := event.GetEvent().GetDateTime(); ts != nil {
		b.EventDateTime.Append(ts.AsTime())
	} else {
		b.EventDateTime.Append(time.Time{})
	}
}

// TODO: Implement this method to extract the event payload fields into the batch columns.
// The generated .gen.go file contains the available column fields for this table.
func (b *{{.BatchName}}) appendPayload(_ *xatu.DecoratedEvent) error {
	return fmt.Errorf("{{.TypeName}}: appendPayload not implemented")
}
`))

// scaffoldData holds the template parameters for a route scaffold.
type scaffoldData struct {
	Package   string
	TypeName  string
	BatchName string
	TableName string
}

// writeRouteScaffold writes a scaffold route file at the given path.
func writeRouteScaffold(path, pkg, typeName, table string) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create %s: %w", path, err)
	}

	defer f.Close()

	data := scaffoldData{
		Package:   pkg,
		TypeName:  typeName,
		BatchName: batchName(typeName),
		TableName: table,
	}

	if err := routeScaffoldTmpl.Execute(f, data); err != nil {
		return fmt.Errorf("execute template: %w", err)
	}

	return nil
}

// routeTestScaffoldTmpl is the template for a new snapshot test file
// that validates the table's FlattenTo output using testfixture.AssertSnapshot.
var routeTestScaffoldTmpl = template.Must(template.New("routeTest").Parse(`package {{.Package}}

import (
	"testing"

	"github.com/ethpandaops/xatu/pkg/consumoor/route/testfixture"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

func TestSnapshot_{{.TableName}}(t *testing.T) {
	if len({{.TypeName}}EventNames) == 0 {
		t.Skip("no event names registered for {{.TableName}}")
	}

	testfixture.AssertSnapshot(t, new{{.BatchName}}(), &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     {{.TypeName}}EventNames[0],
			DateTime: testfixture.TS(),
			Id:       "snapshot-1",
		},
		Meta: testfixture.BaseMeta(),
		// TODO: Add event-specific Data field and MetaWithAdditional for richer assertions.
	}, 1, map[string]any{
		"meta_client_name": "test-client",
		// TODO: Add payload-specific column assertions.
	})
}
`))

// writeRouteTestScaffold writes a scaffold test file at the given path.
func writeRouteTestScaffold(path, pkg, typeName, table string) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create %s: %w", path, err)
	}

	defer f.Close()

	data := scaffoldData{
		Package:   pkg,
		TypeName:  typeName,
		BatchName: batchName(typeName),
		TableName: table,
	}

	if err := routeTestScaffoldTmpl.Execute(f, data); err != nil {
		return fmt.Errorf("execute test template: %w", err)
	}

	return nil
}

// batchName derives the batch struct name from the lowerCamelCase type name.
// e.g. "libp2pIdentify" -> "libp2pIdentifyBatch"
func batchName(typeName string) string {
	return strings.TrimSuffix(typeName, "Row") + "Batch"
}
