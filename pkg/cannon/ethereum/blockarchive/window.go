package blockarchive

import "fmt"

// windowPlan is what one window fetch reads from an era file, and which slots of the window that proves.
type windowPlan struct {
	// readFirst and readLast widen the window to the nearest block on either side of it in the era file.
	readFirst, readLast uint64
	// lo..hi are the proven slots: blocks linked end to end by parent root and the missed slots between them.
	lo, hi uint64
	// anchor is the highest block read; its root must match the beacon node's.
	anchor uint64
	// succ is set when readLast is the first block after the window.
	succ bool
}

// planWindow plans reading slots first..last of an era file; ok is false when the file can prove none of them.
func planWindow(idx *slotIndex, first, last uint64) (p windowPlan, ok bool) {
	pred, hasPred := idx.prevEntry(first)
	succ, hasSucc := idx.nextEntry(last)

	lowest, hasBlocks := first, false
	if _, has := idx.entryAt(first); has {
		hasBlocks = true
	} else if next, has := idx.nextEntry(first); has && next <= last {
		lowest, hasBlocks = next, true
	}

	if !hasBlocks {
		if !hasPred || !hasSucc {
			return p, false
		}

		return windowPlan{readFirst: pred, readLast: succ, lo: first, hi: last, anchor: succ, succ: true}, true
	}

	highest := last
	if _, has := idx.entryAt(last); !has {
		highest, _ = idx.prevEntry(last)
	}

	p = windowPlan{readFirst: lowest, readLast: highest, lo: lowest, hi: highest, anchor: highest}

	if hasPred {
		p.readFirst, p.lo = pred, first
	}

	if hasSucc {
		p.readLast, p.hi, p.anchor, p.succ = succ, last, succ, true
	}

	return p, true
}

func windowKey(name string, first, last uint64) string {
	return fmt.Sprintf("%s/%d-%d", name, first, last)
}

// slotSet is a set of slots within first..last.
type slotSet struct {
	first, last uint64
	bits        []uint64
}

func newSlotSet(first, last uint64) slotSet {
	return slotSet{first: first, last: last, bits: make([]uint64, (last-first)/64+1)}
}

func (s *slotSet) add(slot uint64) {
	if slot >= s.first && slot <= s.last {
		s.bits[(slot-s.first)/64] |= 1 << ((slot - s.first) % 64)
	}
}

func (s *slotSet) has(slot uint64) bool {
	return slot >= s.first && slot <= s.last && s.bits[(slot-s.first)/64]&(1<<((slot-s.first)%64)) != 0
}

// windowMemo records what a fetched window answers for, so lookups the archive cannot answer skip the fetch.
type windowMemo struct {
	blocks    slotSet
	envelopes slotSet
}

func memoOf(slots Slots, first, last uint64) *windowMemo {
	m := &windowMemo{blocks: newSlotSet(first, last), envelopes: newSlotSet(first, last)}

	for s, c := range slots {
		m.blocks.add(uint64(s))

		if c.EnvelopeKnown {
			m.envelopes.add(uint64(s))
		}
	}

	return m
}
