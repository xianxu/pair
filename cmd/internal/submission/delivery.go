package submission

import "fmt"

type Delivery string

const (
	Prepared   Delivery = "prepared"
	Delivering Delivery = "delivering"
	Dispatched Delivery = "dispatched"
	Resolved   Delivery = "resolved"
)

type Event string

const (
	Begin    Event = "begin"
	Ack      Event = "ack"
	Complete Event = "complete"
)

func validDelivery(s Delivery) bool {
	return s == Prepared || s == Delivering || s == Dispatched || s == Resolved
}

// ApplyDelivery never infers a send from silence. Events acknowledge observed work.
func ApplyDelivery(s Delivery, e Event) (Delivery, error) {
	switch e {
	case Begin:
		if s == Prepared || s == Delivering {
			return Delivering, nil
		}
	case Ack:
		if s == Delivering || s == Dispatched {
			return Dispatched, nil
		}
		if s == Resolved {
			return Resolved, nil
		}
	case Complete:
		if s == Dispatched || s == Resolved {
			return Resolved, nil
		}
	}
	return s, fmt.Errorf("refused delivery transition %s + %s", s, e)
}
