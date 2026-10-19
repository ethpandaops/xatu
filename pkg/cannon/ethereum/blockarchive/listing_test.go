package blockarchive

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const hoodi10 = "hoodi-00010-0123abcd.era"

func parseListing(network string, bodies ...string) map[uint64]eraFile {
	l := newListing(network)
	for _, body := range bodies {
		l.add([]byte(body))
	}

	return l.files()
}

func TestParseListingDirectoryIndex(t *testing.T) {
	body := `<html><body><pre><a href="../">../</a>
<a href="mainnet-00000-4b363db9.era">mainnet-00000-4b363db9.era</a>   19-Apr-2023 08:01   1810141
<a href="mainnet-00001-40cf2f3c.era">mainnet-00001-40cf2f3c.era</a>   19-Apr-2023 08:01   18580262
<a href="mainnet-01886-653d1f74.era">mainnet-01886-653d1f74.era</a>   17-Oct-2026 10:19   754122718
<a href="mainnet-01886-653d1f74.envelopes.e2s">mainnet-01886-653d1f74.envelopes.e2s</a>
<a href="mainnet-01886-653d1f74.erb">mainnet-01886-653d1f74.erb</a>
<a href="mainnet-01886-653d1f74-noproofs.ere">mainnet-01886-653d1f74-noproofs.ere</a>
<a href="sepolia-01389-5a35096a.era">sepolia-01389-5a35096a.era</a>
<a href="xmainnet-00002-aaaaaaaa.era">xmainnet-00002-aaaaaaaa.era</a>
</pre></body></html>`

	assert.Equal(t, map[uint64]eraFile{
		0:    {name: "mainnet-00000-4b363db9.era"},
		1:    {name: "mainnet-00001-40cf2f3c.era"},
		1886: {name: "mainnet-01886-653d1f74.era", envelopes: "mainnet-01886-653d1f74.envelopes.e2s"},
	}, parseListing("mainnet", body))
}

func TestParseListingBlockArchiverAPI(t *testing.T) {
	body := `{"network": "hoodi", "kinds": ["era", "envelopes", "erb", "ere"], "eras": [
{"era": 10, "start_slot": 73728, "end_slot": 81919, "state_slot": 81920, "files": {
  "era": {"file": "hoodi-00010-0123abcd.era", "profile": null, "url": "/era/hoodi/hoodi-00010-0123abcd.era",
    "size_bytes": 1, "sha256": "00", "entries": 8190, "source": "https://hoodi.era.nimbus.team/hoodi-00010-0123abcd.era",
    "published_at_ns": 1},
  "envelopes": null, "erb": null, "ere": null}},
{"era": 11, "start_slot": 81920, "end_slot": 90111, "state_slot": 90112, "files": {
  "era": {"file": "hoodi-00011-89abcdef.era", "profile": null, "url": "/era/hoodi/hoodi-00011-89abcdef.era",
    "size_bytes": 1, "sha256": "00", "entries": 8190, "source": null, "published_at_ns": 1},
  "envelopes": {"file": "hoodi-00011-89abcdef.envelopes.e2s", "profile": null,
    "url": "/era/hoodi/hoodi-00011-89abcdef.envelopes.e2s", "size_bytes": 1, "sha256": "00", "entries": 8000,
    "source": null, "published_at_ns": 2},
  "erb": null, "ere": null}}],
"coverage": {"era": {"count": 2, "min_era": 10, "max_era": 11, "size_bytes": 2}}, "limit": 100, "offset": 0}`

	assert.Equal(t, map[uint64]eraFile{
		10: {name: hoodi10},
		11: {name: "hoodi-00011-89abcdef.era", envelopes: "hoodi-00011-89abcdef.envelopes.e2s"},
	}, parseListing("hoodi", body))
}

func TestParseListingNamesEndAtTheirSuffix(t *testing.T) {
	body := `<a href="hoodi-00010-0123abcd.era">hoodi-00010-0123abcd.era</a>
<a href="hoodi-00011-89abcdef.era1">hoodi-00011-89abcdef.era1</a>
<a href="hoodi-00012-01234567.era.tmp">hoodi-00012-01234567.era.tmp</a>
<a href="hoodi-00013-aaaaaaaa.era.envelopes.e2s">x</a>
<a href="hoodi-00010-0123abcd.envelopes.e2s.part">x</a>
hoodi-00014-bbbbbbbb.era hoodi-00014-bbbbbbbb.envelopes.e2s
"hoodi-00015-cccccccc.era"`

	assert.Equal(t, map[uint64]eraFile{
		10: {name: hoodi10},
		14: {name: "hoodi-00014-bbbbbbbb.era", envelopes: "hoodi-00014-bbbbbbbb.envelopes.e2s"},
		15: {name: "hoodi-00015-cccccccc.era"},
	}, parseListing("hoodi", body))
}

func TestParseListingPairsSidecarsByBasename(t *testing.T) {
	assert.Equal(t, map[uint64]eraFile{
		10: {name: hoodi10},
		11: {name: "hoodi-00011-89abcdef.era", envelopes: "hoodi-00011-89abcdef.envelopes.e2s"},
	}, parseListing("hoodi",
		`hoodi-00010-0123abcd.era hoodi-00010-ffffffff.envelopes.e2s hoodi-00012-0123abcd.envelopes.e2s`,
		`hoodi-00011-89abcdef.era`,
		`hoodi-00011-89abcdef.envelopes.e2s`,
	), "a sidecar pairs only with the era file of its basename, across pages")
}

func TestNextPage(t *testing.T) {
	const other = "https://archive/x"

	tests := map[string]struct {
		target string
		body   string
		want   string
	}{
		"full page": {
			target: "https://archive/api/v1/era?network=hoodi",
			body:   `{"eras": [{"era": 1}, {"era": 2}], "limit": 2, "offset": 0}`,
			want:   "https://archive/api/v1/era?limit=2&network=hoodi&offset=2",
		},
		"later full page": {
			target: "https://archive/api/v1/era?network=hoodi&limit=2&offset=2",
			body:   `{"eras": [{"era": 3}, {"era": 4}], "limit": 2, "offset": 2}`,
			want:   "https://archive/api/v1/era?limit=2&network=hoodi&offset=4",
		},
		"short page":        {target: other, body: `{"eras": [{"era": 3}], "limit": 2, "offset": 2}`},
		"empty page":        {target: other, body: `{"eras": [], "limit": 100, "offset": 200}`},
		"unpaged json":      {target: other, body: `{"eras": [{"era": 1}]}`},
		"every era":         {target: other, body: `{"eras": [{"era": 1}], "limit": null, "offset": 0}`},
		"html index":        {target: "https://archive/", body: `<html><a href="mainnet-00001-40cf2f3c.era"></a></html>`},
		"non-positive size": {target: other, body: `{"eras": [], "limit": 0, "offset": 0}`},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := nextPage(tc.target, []byte(tc.body))
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}
