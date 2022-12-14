// RLPx ping https://github.com/ethereum/devp2p/blob/master/rlpx.md#ping-0x02
package execution

const (
	PingCode = 0x02
)

type Ping struct{}

func (h *Ping) Code() int { return PingCode }

func (h *Ping) ReqID() uint64 { return 0 }
