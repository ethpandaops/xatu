package blockarchive

import (
	"regexp"
	"strconv"
	"strings"
)

// eraFile is an era file listed by the archive, with its envelope sidecar when one is published.
type eraFile struct {
	name      string
	envelopes string
}

// parseListing finds network's era files and <era-name>.envelopes.e2s sidecars named anywhere in body.
func parseListing(body []byte, network string) map[uint64]eraFile {
	re := regexp.MustCompile(`(?:^|[^A-Za-z0-9_-])(` + regexp.QuoteMeta(network) +
		`-(\d{5,})-[0-9a-f]{8})(\.era\.envelopes\.e2s|\.envelopes\.e2s|\.era)`)

	eras := make(map[uint64]eraFile)
	sidecars := make(map[string]string)

	for _, m := range re.FindAllSubmatch(body, -1) {
		stem, suffix := string(m[1]), string(m[3])

		if strings.HasSuffix(suffix, ".e2s") {
			sidecars[stem] = stem + suffix

			continue
		}

		era, err := strconv.ParseUint(string(m[2]), 10, 64)
		if err != nil {
			continue
		}

		if _, seen := eras[era]; !seen {
			eras[era] = eraFile{name: stem + suffix}
		}
	}

	for era, f := range eras {
		f.envelopes = sidecars[strings.TrimSuffix(f.name, ".era")]
		eras[era] = f
	}

	return eras
}
