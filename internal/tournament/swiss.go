package tournament

import (
	"errors"
	"fmt"
	"sort"
)

var ErrPairingFailed = errors.New("failed to generate valid Swiss pairings")

type SwissPlayer struct {
	UserID            int64
	Rating            int
	Score             float64
	Buchholz          float64
	Wins              int
	Draws             int
	Losses            int
	GamesPlayed       int
	PreviousOpponents map[int64]bool
	WhiteCount        int
	BlackCount        int
	LastColor         string
	ColorStreak       int
	ReceivedBye       bool
}

type SwissPairingResult struct {
	WhitePlayerID *int64
	BlackPlayerID *int64
	IsBye         bool
}

func GenerateSwissPairings(players []*SwissPlayer) ([]SwissPairingResult, error) {
	if len(players) < 2 {
		return nil, errors.New("at least 2 players required for pairings")
	}

	pool := make([]*SwissPlayer, len(players))
	for i, p := range players {
		cp := *p
		cp.PreviousOpponents = make(map[int64]bool)
		for k, v := range p.PreviousOpponents {
			cp.PreviousOpponents[k] = v
		}
		pool[i] = &cp
	}

	var pairings []SwissPairingResult

	if len(pool)%2 != 0 {
		byeIndex := selectByePlayer(pool)
		if byeIndex != -1 {
			byePlayer := pool[byeIndex]
			pairings = append(pairings, SwissPairingResult{
				WhitePlayerID: &byePlayer.UserID,
				BlackPlayerID: nil,
				IsBye:         true,
			})
			pool = append(pool[:byeIndex], pool[byeIndex+1:]...)
		}
	}

	sortPlayers(pool)

	matched, err := matchSwissPool(pool)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrPairingFailed, err)
	}

	pairings = append(pairings, matched...)
	return pairings, nil
}

func selectByePlayer(players []*SwissPlayer) int {
	candidates := make([]int, len(players))
	for i := range players {
		candidates[i] = i
	}

	sort.SliceStable(candidates, func(i, j int) bool {
		pi, pj := players[candidates[i]], players[candidates[j]]
		if pi.Score != pj.Score {
			return pi.Score < pj.Score
		}
		if pi.Buchholz != pj.Buchholz {
			return pi.Buchholz < pj.Buchholz
		}
		if pi.Rating != pj.Rating {
			return pi.Rating < pj.Rating
		}
		return pi.GamesPlayed < pj.GamesPlayed
	})

	for _, idx := range candidates {
		if !players[idx].ReceivedBye {
			return idx
		}
	}

	return candidates[0]
}

func sortPlayers(players []*SwissPlayer) {
	sort.SliceStable(players, func(i, j int) bool {
		if players[i].Score != players[j].Score {
			return players[i].Score > players[j].Score
		}
		if players[i].Buchholz != players[j].Buchholz {
			return players[i].Buchholz > players[j].Buchholz
		}
		if players[i].Rating != players[j].Rating {
			return players[i].Rating > players[j].Rating
		}
		return players[i].UserID < players[j].UserID
	})
}

func matchSwissPool(players []*SwissPlayer) ([]SwissPairingResult, error) {
	n := len(players)
	if n == 0 {
		return []SwissPairingResult{}, nil
	}

	var results []SwissPairingResult
	used := make([]bool, n)

	if solvePairings(players, used, 0, &results, true) {
		return results, nil
	}

	results = nil
	used = make([]bool, n)
	if solvePairings(players, used, 0, &results, false) {
		return results, nil
	}

	return nil, errors.New("unable to find valid non-repeat pairing")
}

func solvePairings(players []*SwissPlayer, used []bool, firstUnused int, results *[]SwissPairingResult, strictNoRepeat bool) bool {
	n := len(players)

	for firstUnused < n && used[firstUnused] {
		firstUnused++
	}

	if firstUnused >= n {
		return true
	}

	p1 := players[firstUnused]
	used[firstUnused] = true

	candidates := getPairingCandidates(players, used, firstUnused)

	for _, i := range candidates {
		p2 := players[i]

		if strictNoRepeat {
			if p1.PreviousOpponents[p2.UserID] || p2.PreviousOpponents[p1.UserID] {
				continue
			}
		}

		white, black := assignColors(p1, p2)

		used[i] = true
		*results = append(*results, SwissPairingResult{
			WhitePlayerID: &white.UserID,
			BlackPlayerID: &black.UserID,
			IsBye:         false,
		})

		if solvePairings(players, used, firstUnused+1, results, strictNoRepeat) {
			return true
		}

		*results = (*results)[:len(*results)-1]
		used[i] = false
	}

	used[firstUnused] = false
	return false
}

func getPairingCandidates(players []*SwissPlayer, used []bool, firstUnused int) []int {
	n := len(players)
	candidates := make([]int, 0, n)

	for i := firstUnused + 1; i < n; i++ {
		if !used[i] {
			candidates = append(candidates, i)
		}
	}

	half := len(candidates) / 2
	if half > 0 {
		dutchOrdered := make([]int, 0, len(candidates))
		for i := 0; i < half; i++ {
			dutchOrdered = append(dutchOrdered, candidates[i+half])
			dutchOrdered = append(dutchOrdered, candidates[i])
		}
		if len(candidates)%2 != 0 {
			dutchOrdered = append(dutchOrdered, candidates[len(candidates)-1])
		}
		return dutchOrdered
	}

	return candidates
}

func assignColors(p1, p2 *SwissPlayer) (white *SwissPlayer, black *SwissPlayer) {
	b1 := p1.WhiteCount - p1.BlackCount
	b2 := p2.WhiteCount - p2.BlackCount

	if p1.LastColor == "W" && p1.ColorStreak >= 2 {
		return p2, p1
	}
	if p2.LastColor == "W" && p2.ColorStreak >= 2 {
		return p1, p2
	}
	if p1.LastColor == "B" && p1.ColorStreak >= 2 {
		return p1, p2
	}
	if p2.LastColor == "B" && p2.ColorStreak >= 2 {
		return p2, p1
	}

	if b1 < b2 {
		return p1, p2
	} else if b2 < b1 {
		return p2, p1
	}

	if p1.LastColor == "B" && p2.LastColor != "B" {
		return p1, p2
	} else if p2.LastColor == "B" && p1.LastColor != "B" {
		return p2, p1
	}

	if p1.Rating > p2.Rating {
		return p1, p2
	}

	return p1, p2
}
