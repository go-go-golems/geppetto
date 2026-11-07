package decision

// GateAction is what a caller should do with a probability.
type GateAction string

const (
	// Act: the probability clears the act threshold.
	Act GateAction = "act"
	// Confirm: between the confirm and act thresholds; ask for confirmation
	// (a human, a slower model, a second check).
	Confirm GateAction = "confirm"
	// Reject: below the confirm threshold.
	Reject GateAction = "reject"
)

// Gate applies caller thresholds to a probability: p >= act → Act,
// p >= confirm → Confirm, otherwise Reject. The package sets no default
// thresholds; they are policy, and belong to the caller.
func Gate(p, act, confirm float64) GateAction {
	switch {
	case p >= act:
		return Act
	case p >= confirm:
		return Confirm
	default:
		return Reject
	}
}
