package tournament

import (
	"errors"
	"math"
)

func NextPowerOfTwo(n int) int {
	if n <= 1 {
		return 1
	}
	p := 1
	for p < n {
		p <<= 1
	}
	return p
}

func GenerateKnockoutRound1(players []*SwissPlayer) ([]SwissPairingResult, int, error) {
	n := len(players)
	if n < 2 {
		return nil, 0, errors.New("at least 2 players required for Knockout tournament")
	}

	bracketSize := NextPowerOfTwo(n)
	totalRounds := int(math.Log2(float64(bracketSize)))
	byesCount := bracketSize - n

	var pairings []SwissPairingResult

	for i := 0; i < byesCount; i++ {
		wID := players[i].UserID
		pairings = append(pairings, SwissPairingResult{
			WhitePlayerID: &wID,
			BlackPlayerID: nil,
			IsBye:         true,
		})
	}

	remaining := players[byesCount:]
	for i := 0; i < len(remaining); i += 2 {
		if i+1 < len(remaining) {
			wID := remaining[i].UserID
			bID := remaining[i+1].UserID
			pairings = append(pairings, SwissPairingResult{
				WhitePlayerID: &wID,
				BlackPlayerID: &bID,
				IsBye:         false,
			})
		}
	}

	return pairings, totalRounds, nil
}

func GenerateKnockoutNextRound(winnerIDs []int64) ([]SwissPairingResult, error) {
	if len(winnerIDs) < 2 {
		return nil, errors.New("at least 2 winners required for next Knockout round")
	}

	var pairings []SwissPairingResult
	for i := 0; i < len(winnerIDs); i += 2 {
		if i+1 < len(winnerIDs) {
			wID := winnerIDs[i]
			bID := winnerIDs[i+1]
			pairings = append(pairings, SwissPairingResult{
				WhitePlayerID: &wID,
				BlackPlayerID: &bID,
				IsBye:         false,
			})
		} else {
			wID := winnerIDs[i]
			pairings = append(pairings, SwissPairingResult{
				WhitePlayerID: &wID,
				BlackPlayerID: nil,
				IsBye:         true,
			})
		}
	}

	return pairings, nil
}
