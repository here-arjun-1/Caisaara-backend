package tournament

import (
	"errors"
)

func GenerateRoundRobinSchedule(playerIDs []int64) (map[int][]SwissPairingResult, int, error) {
	n := len(playerIDs)
	if n < 2 {
		return nil, 0, errors.New("at least 2 players required for Round Robin")
	}

	players := make([]int64, n)
	copy(players, playerIDs)

	if len(players)%2 != 0 {
		players = append(players, -1)
	}

	totalPlayers := len(players)
	totalRounds := totalPlayers - 1

	schedule := make(map[int][]SwissPairingResult)

	for r := 1; r <= totalRounds; r++ {
		var roundPairings []SwissPairingResult

		for i := 0; i < totalPlayers/2; i++ {
			p1 := players[i]
			p2 := players[totalPlayers-1-i]

			if p1 == -1 || p2 == -1 {
				realPlayer := p1
				if p1 == -1 {
					realPlayer = p2
				}
				wID := realPlayer
				roundPairings = append(roundPairings, SwissPairingResult{
					WhitePlayerID: &wID,
					BlackPlayerID: nil,
					IsBye:         true,
				})
			} else {
				var wID, bID int64
				if (i+r)%2 == 0 {
					wID = p1
					bID = p2
				} else {
					wID = p2
					bID = p1
				}
				roundPairings = append(roundPairings, SwissPairingResult{
					WhitePlayerID: &wID,
					BlackPlayerID: &bID,
					IsBye:         false,
				})
			}
		}

		schedule[r] = roundPairings

		last := players[totalPlayers-1]
		for k := totalPlayers - 1; k > 1; k-- {
			players[k] = players[k-1]
		}
		players[1] = last
	}

	return schedule, totalRounds, nil
}
