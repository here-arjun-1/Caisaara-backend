package report

const (
	ReasonCheating = "cheating"
	ReasonAbuse    = "abuse"
	ReasonOther    = "other"
)

const StatusPending = "pending"

const MaxDetailsLength = 500

var validReasons = map[string]bool{
	ReasonCheating: true,
	ReasonAbuse:    true,
	ReasonOther:    true,
}
