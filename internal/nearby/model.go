package nearby

const (
	searchRadiusMeters  = 50000
	maxNearbyPlayers    = 50
	activeWindowMinutes = 10
)

type NearbyPlayer struct {
	UserID     int64  `json:"user_id"`
	Username   string `json:"username"`
	Rating     int    `json:"rating"`
	DistanceKm int    `json:"distance_km"`
	Active     bool   `json:"active"`
}
