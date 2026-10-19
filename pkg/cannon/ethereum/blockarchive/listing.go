package blockarchive

import (
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strconv"
)

const (
	eraSuffix     = ".era"
	sidecarSuffix = ".envelopes.e2s"
)

// eraFile is an era file listed by the archive, with its envelope sidecar when one is published.
type eraFile struct {
	name      string
	envelopes string
}

// listing collects a network's era files and <era basename>.envelopes.e2s sidecars named anywhere in listing pages.
type listing struct {
	re       *regexp.Regexp
	eras     map[uint64]string
	sidecars map[string]bool
}

func newListing(network string) *listing {
	return &listing{
		re: regexp.MustCompile(`(?:^|[^A-Za-z0-9_-])(` + regexp.QuoteMeta(network) +
			`-(\d{5,})-[0-9a-f]{8})(` + regexp.QuoteMeta(sidecarSuffix) + `|` + regexp.QuoteMeta(eraSuffix) + `)`),
		eras:     make(map[uint64]string),
		sidecars: make(map[string]bool),
	}
}

func (l *listing) add(body []byte) {
	for _, m := range l.re.FindAllSubmatchIndex(body, -1) {
		// A name must end at its suffix, so .era1 or .era.tmp files are not era files.
		if m[1] < len(body) && isNameByte(body[m[1]]) {
			continue
		}

		basename, suffix := string(body[m[2]:m[3]]), string(body[m[6]:m[7]])

		if suffix == sidecarSuffix {
			l.sidecars[basename] = true

			continue
		}

		era, err := strconv.ParseUint(string(body[m[4]:m[5]]), 10, 64)
		if err != nil {
			continue
		}

		if _, seen := l.eras[era]; !seen {
			l.eras[era] = basename
		}
	}
}

func isNameByte(b byte) bool {
	return b >= 'a' && b <= 'z' || b >= 'A' && b <= 'Z' || b >= '0' && b <= '9' || b == '.' || b == '_' || b == '-'
}

// files pairs each listed era file with its sidecar, which must share the era file's basename.
func (l *listing) files() map[uint64]eraFile {
	out := make(map[uint64]eraFile, len(l.eras))

	for era, basename := range l.eras {
		f := eraFile{name: basename + eraSuffix}
		if l.sidecars[basename] {
			f.envelopes = basename + sidecarSuffix
		}

		out[era] = f
	}

	return out
}

// pagedListing is the paging part of a JSON listing such as block-archiver's /api/v1/era?network=.
type pagedListing struct {
	Eras   []json.RawMessage `json:"eras"`
	Limit  *int64            `json:"limit"`
	Offset *int64            `json:"offset"`
}

// nextPage returns the listing page after body, fetched from target; "" when body is not a full page of a paged listing.
func nextPage(target string, body []byte) (string, error) {
	var page pagedListing
	if err := json.Unmarshal(body, &page); err != nil || page.Limit == nil || page.Offset == nil ||
		*page.Limit < 1 || int64(len(page.Eras)) < *page.Limit {
		return "", nil //nolint:nilerr // a body that is not a paged JSON listing is a whole listing.
	}

	u, err := url.Parse(target)
	if err != nil {
		return "", fmt.Errorf("invalid listing url: %w", err)
	}

	q := u.Query()
	q.Set("offset", strconv.FormatInt(*page.Offset+int64(len(page.Eras)), 10))
	q.Set("limit", strconv.FormatInt(*page.Limit, 10))
	u.RawQuery = q.Encode()

	return u.String(), nil
}
