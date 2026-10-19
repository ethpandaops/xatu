package blockarchive

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseListingDirectoryIndex(t *testing.T) {
	body := `<html><body><pre><a href="../">../</a>
<a href="mainnet-00000-4b363db9.era">mainnet-00000-4b363db9.era</a>   19-Apr-2023 08:01   1810141
<a href="mainnet-00001-40cf2f3c.era">mainnet-00001-40cf2f3c.era</a>   19-Apr-2023 08:01   18580262
<a href="mainnet-01886-653d1f74.era">mainnet-01886-653d1f74.era</a>   17-Oct-2026 10:19   754122718
<a href="mainnet-01886-653d1f74.envelopes.e2s">mainnet-01886-653d1f74.envelopes.e2s</a>
<a href="sepolia-01389-5a35096a.era">sepolia-01389-5a35096a.era</a>
<a href="xmainnet-00002-aaaaaaaa.era">xmainnet-00002-aaaaaaaa.era</a>
</pre></body></html>`

	assert.Equal(t, map[uint64]eraFile{
		0:    {name: "mainnet-00000-4b363db9.era"},
		1:    {name: "mainnet-00001-40cf2f3c.era"},
		1886: {name: "mainnet-01886-653d1f74.era", envelopes: "mainnet-01886-653d1f74.envelopes.e2s"},
	}, parseListing([]byte(body), "mainnet"))
}

func TestParseListingJSON(t *testing.T) {
	body := `{"files":[
		{"name":"hoodi-00010-0123abcd.era","size":1},
		{"name":"hoodi-00011-89abcdef.era","url":"https://archive/era/hoodi/hoodi-00011-89abcdef.era"},
		{"name":"hoodi-00011-89abcdef.era.envelopes.e2s"},
		{"name":"hoodi-00010-ffffffff.envelopes.e2s"}
	]}`

	assert.Equal(t, map[uint64]eraFile{
		10: {name: "hoodi-00010-0123abcd.era"},
		11: {name: "hoodi-00011-89abcdef.era", envelopes: "hoodi-00011-89abcdef.era.envelopes.e2s"},
	}, parseListing([]byte(body), "hoodi"), "a sidecar pairs only with the era file of the same root")
}
