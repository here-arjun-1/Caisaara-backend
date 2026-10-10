package nearby

const (
	searchRadiusMeters = 50000
	maxNearbyPlayers   = 50
)

type NearbyPlayer struct {
	UserID     int64  `json:"user_id"`
	Username   string `json:"username"`
	Rating     int    `json:"rating"`
	DistanceKm int    `json:"distance_km"`
}
