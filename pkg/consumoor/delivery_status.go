package consumoor

// DeliveryStatus represents the final processing outcome for one consumed
// Kafka message in the current attempt.
type DeliveryStatus uint8

const (
	// DeliveryStatusDelivered means processing completed successfully.
	DeliveryStatusDelivered DeliveryStatus = iota
	// DeliveryStatusErrored means processing failed transiently and should be
	// retried by replaying the message.
	DeliveryStatusErrored
	// DeliveryStatusRejected means processing failed permanently and the
	// message can be committed (dropped) without retry.
	DeliveryStatusRejected
)

func (s DeliveryStatus) String() string {
	switch s {
	case DeliveryStatusDelivered:
		return "delivered"
	case DeliveryStatusErrored:
		return "errored"
	case DeliveryStatusRejected:
		return "rejected"
	default:
		return "unknown"
	}
}

// commitEligible reports whether offsets can advance past this message.
func (s DeliveryStatus) commitEligible() bool {
	switch s {
	case DeliveryStatusDelivered, DeliveryStatusRejected:
		return true
	default:
		return false
	}
}
