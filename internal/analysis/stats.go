package analysis

import (
	"fmt"
	"math"
	"strings"
)

type SummaryStatistics struct {
	AccuracyWhite           float64 `json:"accuracy_white"`
	AccuracyBlack           float64 `json:"accuracy_black"`
	RatingWhite             int     `json:"rating_white"`
	RatingBlack             int     `json:"rating_black"`
	BrilliantWhite          int     `json:"brilliant_white"`
	GreatWhite              int     `json:"great_white"`
	BestWhite               int     `json:"best_white"`
	GoodWhite               int     `json:"good_white"`
	InaccuraciesWhite       int     `json:"inaccuracies_white"`
	MistakesWhite           int     `json:"mistakes_white"`
	MissesWhite             int     `json:"misses_white"`
	BlundersWhite           int     `json:"blunders_white"`
	BrilliantBlack          int     `json:"brilliant_black"`
	GreatBlack              int     `json:"great_black"`
	BestBlack               int     `json:"best_black"`
	GoodBlack               int     `json:"good_black"`
	InaccuraciesBlack       int     `json:"inaccuracies_black"`
	MistakesBlack           int     `json:"mistakes_black"`
	MissesBlack             int     `json:"misses_black"`
	BlundersBlack           int     `json:"blunders_black"`
	OpeningAccuracyWhite    float64 `json:"opening_accuracy_white"`
	OpeningAccuracyBlack    float64 `json:"opening_accuracy_black"`
	MiddlegameAccuracyWhite float64 `json:"middlegame_accuracy_white"`
	MiddlegameAccuracyBlack float64 `json:"middlegame_accuracy_black"`
	EndgameAccuracyWhite    float64 `json:"endgame_accuracy_white"`
	EndgameAccuracyBlack    float64 `json:"endgame_accuracy_black"`
}

type EvalHistoryPoint struct {
	MoveNumber  int    `json:"move_number"`
	PlayerColor string `json:"player_color"`
	EvalWhite   int    `json:"eval_white"`
}

type StatisticsService struct{}

func NewStatisticsService() *StatisticsService {
	return &StatisticsService{}
}

func (s *StatisticsService) CalculateStatistics(moves []GameMoveAnalysis) (SummaryStatistics, []EvalHistoryPoint) {
	var stats SummaryStatistics
	evalHistory := make([]EvalHistoryPoint, 0, len(moves))

	whiteLosses := []float64{}
	blackLosses := []float64{}

	whiteOpeningLosses := []float64{}
	blackOpeningLosses := []float64{}
	whiteMiddleLosses := []float64{}
	blackMiddleLosses := []float64{}
	whiteEndLosses := []float64{}
	blackEndLosses := []float64{}

	for _, m := range moves {
		evalVal := 0
		if m.EvalAfter != nil {
			evalVal = *m.EvalAfter
		}

		evalHistory = append(evalHistory, EvalHistoryPoint{
			MoveNumber:  m.MoveNumber,
			PlayerColor: m.PlayerColor,
			EvalWhite:   evalVal,
		})

		cpl := float64(m.CentipawnLoss)

		if m.PlayerColor == "white" {
			whiteLosses = append(whiteLosses, cpl)
			switch m.Phase {
			case PhaseOpening:
				whiteOpeningLosses = append(whiteOpeningLosses, cpl)
			case PhaseMiddlegame:
				whiteMiddleLosses = append(whiteMiddleLosses, cpl)
			case PhaseEndgame:
				whiteEndLosses = append(whiteEndLosses, cpl)
			}

			switch m.Classification {
			case ClassificationBrilliant:
				stats.BrilliantWhite++
			case ClassificationGreat:
				stats.GreatWhite++
			case ClassificationBest:
				stats.BestWhite++
			case ClassificationGood:
				stats.GoodWhite++
			case ClassificationInaccuracy:
				stats.InaccuraciesWhite++
			case ClassificationMistake:
				stats.MistakesWhite++
			case ClassificationMiss:
				stats.MissesWhite++
			case ClassificationBlunder:
				stats.BlundersWhite++
			}
		} else {
			blackLosses = append(blackLosses, cpl)
			switch m.Phase {
			case PhaseOpening:
				blackOpeningLosses = append(blackOpeningLosses, cpl)
			case PhaseMiddlegame:
				blackMiddleLosses = append(blackMiddleLosses, cpl)
			case PhaseEndgame:
				blackEndLosses = append(blackEndLosses, cpl)
			}

			switch m.Classification {
			case ClassificationBrilliant:
				stats.BrilliantBlack++
			case ClassificationGreat:
				stats.GreatBlack++
			case ClassificationBest:
				stats.BestBlack++
			case ClassificationGood:
				stats.GoodBlack++
			case ClassificationInaccuracy:
				stats.InaccuraciesBlack++
			case ClassificationMistake:
				stats.MistakesBlack++
			case ClassificationMiss:
				stats.MissesBlack++
			case ClassificationBlunder:
				stats.BlundersBlack++
			}
		}
	}

	stats.AccuracyWhite = CalculateAccuracyFormula(whiteLosses)
	stats.AccuracyBlack = CalculateAccuracyFormula(blackLosses)

	stats.OpeningAccuracyWhite = CalculateAccuracyFormula(whiteOpeningLosses)
	stats.OpeningAccuracyBlack = CalculateAccuracyFormula(blackOpeningLosses)
	stats.MiddlegameAccuracyWhite = CalculateAccuracyFormula(whiteMiddleLosses)
	stats.MiddlegameAccuracyBlack = CalculateAccuracyFormula(blackMiddleLosses)
	stats.EndgameAccuracyWhite = CalculateAccuracyFormula(whiteEndLosses)
	stats.EndgameAccuracyBlack = CalculateAccuracyFormula(blackEndLosses)

	stats.RatingWhite = CalculatePerformanceRating(stats.AccuracyWhite)
	stats.RatingBlack = CalculatePerformanceRating(stats.AccuracyBlack)

	return stats, evalHistory
}

func CalculateAccuracyFormula(losses []float64) float64 {
	if len(losses) == 0 {
		return 100.0
	}
	var total float64
	for _, l := range losses {
		total += l
	}
	avgLoss := total / float64(len(losses))

	accuracy := math.Max(0.0, math.Min(100.0, 100.0-(avgLoss*0.4)))
	return math.Round(accuracy*100) / 100
}

func CalculatePerformanceRating(accuracy float64) int {
	rating := 100.0 + (accuracy * 27.0)
	if rating < 100 {
		rating = 100
	}
	if rating > 2800 {
		rating = 2800
	}
	return int(math.Round(rating))
}

func DeterminePhase(fen string, moveNumber int) string {
	pieceCount := 0
	parts := strings.Split(fen, " ")
	if len(parts) > 0 {
		for _, char := range parts[0] {
			if (char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') {
				pieceCount++
			}
		}
	}
	if moveNumber <= 16 && pieceCount > 24 {
		return PhaseOpening
	}
	if pieceCount <= 12 {
		return PhaseEndgame
	}
	return PhaseMiddlegame
}

func GenerateMoveExplanation(classification string, playedMove string, bestMove string, cpl int, evalBefore int, evalAfter int) string {
	if playedMove != "" && bestMove != "" && playedMove == bestMove {
		return fmt.Sprintf("Best move! Played %s which maintains the optimal engine recommendation.", playedMove)
	}

	switch classification {
	case ClassificationBrilliant:
		return fmt.Sprintf("Brilliant move! %s executes a high-level tactical sequence or piece sacrifice.", playedMove)
	case ClassificationGreat:
		return fmt.Sprintf("Great move! %s is the only move that maintains your advantage.", playedMove)
	case ClassificationBest:
		return fmt.Sprintf("Best move! %s keeps your position strong.", playedMove)
	case ClassificationGood:
		return fmt.Sprintf("Good move. %s is solid, though %s was slightly better.", playedMove, bestMove)
	case ClassificationInaccuracy:
		return fmt.Sprintf("Inaccuracy. %s gives away a slight edge (%d centipawns). Best was %s.", playedMove, cpl, bestMove)
	case ClassificationMistake:
		return fmt.Sprintf("Mistake. %s allows your opponent an opportunity (%d centipawn loss). Best was %s.", playedMove, cpl, bestMove)
	case ClassificationMiss:
		return fmt.Sprintf("Missed opportunity! %s missed a key tactical win or win of material. %s was best.", playedMove, bestMove)
	case ClassificationBlunder:
		return fmt.Sprintf("Blunder! %s drastically changes the position evaluation (%d centipawn loss). Best move was %s.", playedMove, cpl, bestMove)
	default:
		return fmt.Sprintf("Played %s.", playedMove)
	}
}
