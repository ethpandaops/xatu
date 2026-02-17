package transform

// Status represents the final processing outcome for one consumed
// Kafka message in the current attempt.
type Status uint8

const (
	// StatusDelivered means processing completed successfully.
	StatusDelivered Status = iota
	// StatusErrored means processing failed transiently and should be
	// retried by replaying the message.
	StatusErrored
	// StatusRejected means processing failed permanently and the
	// message can be committed (dropped) without retry.
	StatusRejected
)

func (s Status) String() string {
	switch s {
	case StatusDelivered:
		return "delivered"
	case StatusErrored:
		return "errored"
	case StatusRejected:
		return "rejected"
	default:
		return "unknown"
	}
}
