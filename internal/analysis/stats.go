package analysis

import (
	"fmt"
	"math"
	"strings"
)

type SummaryStatistics struct {
	AccuracyWhite     float64 `json:"accuracy_white"`
	AccuracyBlack     float64 `json:"accuracy_black"`
	RatingWhite       int     `json:"rating_white"`
	RatingBlack       int     `json:"rating_black"`
	BrilliantWhite    int     `json:"brilliant_white"`
	GreatWhite        int     `json:"great_white"`
	BestWhite         int     `json:"best_white"`
	GoodWhite         int     `json:"good_white"`
	InaccuraciesWhite int     `json:"inaccuracies_white"`
	MistakesWhite     int     `json:"mistakes_white"`
	MissesWhite       int     `json:"misses_white"`
	BlundersWhite     int     `json:"blunders_white"`
	BrilliantBlack    int     `json:"brilliant_black"`
	GreatBlack        int     `json:"great_black"`
	BestBlack         int     `json:"best_black"`
	GoodBlack         int     `json:"good_black"`
	InaccuraciesBlack int     `json:"inaccuracies_black"`
	MistakesBlack     int     `json:"mistakes_black"`
	MissesBlack       int     `json:"misses_black"`
	BlundersBlack     int     `json:"blunders_black"`
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

func (s *StatisticsService) CalculateStatistics(moves []GameMoveAnalysis) (SummaryStatistics, []PhaseReviewItem, []EvalHistoryPoint) {
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

	whiteOpeningInacc, whiteOpeningMistake, whiteOpeningMiss, whiteOpeningBlunder := 0, 0, 0, 0
	whiteMiddleInacc, whiteMiddleMistake, whiteMiddleMiss, whiteMiddleBlunder := 0, 0, 0, 0
	whiteEndInacc, whiteEndMistake, whiteEndMiss, whiteEndBlunder := 0, 0, 0, 0

	blackOpeningInacc, blackOpeningMistake, blackOpeningMiss, blackOpeningBlunder := 0, 0, 0, 0
	blackMiddleInacc, blackMiddleMistake, blackMiddleMiss, blackMiddleBlunder := 0, 0, 0, 0
	blackEndInacc, blackEndMistake, blackEndMiss, blackEndBlunder := 0, 0, 0, 0

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
				switch m.Classification {
				case ClassificationInaccuracy:
					whiteOpeningInacc++
				case ClassificationMistake:
					whiteOpeningMistake++
				case ClassificationMiss:
					whiteOpeningMiss++
				case ClassificationBlunder:
					whiteOpeningBlunder++
				}
			case PhaseMiddlegame:
				whiteMiddleLosses = append(whiteMiddleLosses, cpl)
				switch m.Classification {
				case ClassificationInaccuracy:
					whiteMiddleInacc++
				case ClassificationMistake:
					whiteMiddleMistake++
				case ClassificationMiss:
					whiteMiddleMiss++
				case ClassificationBlunder:
					whiteMiddleBlunder++
				}
			case PhaseEndgame:
				whiteEndLosses = append(whiteEndLosses, cpl)
				switch m.Classification {
				case ClassificationInaccuracy:
					whiteEndInacc++
				case ClassificationMistake:
					whiteEndMistake++
				case ClassificationMiss:
					whiteEndMiss++
				case ClassificationBlunder:
					whiteEndBlunder++
				}
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
				switch m.Classification {
				case ClassificationInaccuracy:
					blackOpeningInacc++
				case ClassificationMistake:
					blackOpeningMistake++
				case ClassificationMiss:
					blackOpeningMiss++
				case ClassificationBlunder:
					blackOpeningBlunder++
				}
			case PhaseMiddlegame:
				blackMiddleLosses = append(blackMiddleLosses, cpl)
				switch m.Classification {
				case ClassificationInaccuracy:
					blackMiddleInacc++
				case ClassificationMistake:
					blackMiddleMistake++
				case ClassificationMiss:
					blackMiddleMiss++
				case ClassificationBlunder:
					blackMiddleBlunder++
				}
			case PhaseEndgame:
				blackEndLosses = append(blackEndLosses, cpl)
				switch m.Classification {
				case ClassificationInaccuracy:
					blackEndInacc++
				case ClassificationMistake:
					blackEndMistake++
				case ClassificationMiss:
					blackEndMiss++
				case ClassificationBlunder:
					blackEndBlunder++
				}
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

	stats.RatingWhite = CalculatePerformanceRating(stats.AccuracyWhite)
	stats.RatingBlack = CalculatePerformanceRating(stats.AccuracyBlack)

	openAccWhite := CalculateAccuracyFormula(whiteOpeningLosses)
	openAccBlack := CalculateAccuracyFormula(blackOpeningLosses)
	midAccWhite := CalculateAccuracyFormula(whiteMiddleLosses)
	midAccBlack := CalculateAccuracyFormula(blackMiddleLosses)
	endAccWhite := CalculateAccuracyFormula(whiteEndLosses)
	endAccBlack := CalculateAccuracyFormula(blackEndLosses)

	openClassWhite, openSymWhite := ClassifyPhase(whiteOpeningLosses, whiteOpeningInacc, whiteOpeningMistake, whiteOpeningMiss, whiteOpeningBlunder)
	openClassBlack, openSymBlack := ClassifyPhase(blackOpeningLosses, blackOpeningInacc, blackOpeningMistake, blackOpeningMiss, blackOpeningBlunder)
	midClassWhite, midSymWhite := ClassifyPhase(whiteMiddleLosses, whiteMiddleInacc, whiteMiddleMistake, whiteMiddleMiss, whiteMiddleBlunder)
	midClassBlack, midSymBlack := ClassifyPhase(blackMiddleLosses, blackMiddleInacc, blackMiddleMistake, blackMiddleMiss, blackMiddleBlunder)
	endClassWhite, endSymWhite := ClassifyPhase(whiteEndLosses, whiteEndInacc, whiteEndMistake, whiteEndMiss, whiteEndBlunder)
	endClassBlack, endSymBlack := ClassifyPhase(blackEndLosses, blackEndInacc, blackEndMistake, blackEndMiss, blackEndBlunder)

	var pOpenAccW, pOpenAccB *float64
	if len(whiteOpeningLosses) > 0 {
		pOpenAccW = &openAccWhite
	}
	if len(blackOpeningLosses) > 0 {
		pOpenAccB = &openAccBlack
	}

	var pMidAccW, pMidAccB *float64
	if len(whiteMiddleLosses) > 0 {
		pMidAccW = &midAccWhite
	}
	if len(blackMiddleLosses) > 0 {
		pMidAccB = &midAccBlack
	}

	var pEndAccW, pEndAccB *float64
	if len(whiteEndLosses) > 0 {
		pEndAccW = &endAccWhite
	}
	if len(blackEndLosses) > 0 {
		pEndAccB = &endAccBlack
	}

	phases := []PhaseReviewItem{
		{
			Phase:               PhaseOpening,
			AccuracyWhite:       pOpenAccW,
			AccuracyBlack:       pOpenAccB,
			ClassificationWhite: openClassWhite,
			ClassificationBlack: openClassBlack,
			SymbolWhite:         openSymWhite,
			SymbolBlack:         openSymBlack,
		},
		{
			Phase:               PhaseMiddlegame,
			AccuracyWhite:       pMidAccW,
			AccuracyBlack:       pMidAccB,
			ClassificationWhite: midClassWhite,
			ClassificationBlack: midClassBlack,
			SymbolWhite:         midSymWhite,
			SymbolBlack:         midSymBlack,
		},
		{
			Phase:               PhaseEndgame,
			AccuracyWhite:       pEndAccW,
			AccuracyBlack:       pEndAccB,
			ClassificationWhite: endClassWhite,
			ClassificationBlack: endClassBlack,
			SymbolWhite:         endSymWhite,
			SymbolBlack:         endSymBlack,
		},
	}

	return stats, phases, evalHistory
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

func ClassifyPhase(losses []float64, inaccuracies, mistakes, misses, blunders int) (string, string) {
	if len(losses) == 0 {
		return "", "-"
	}
	acc := CalculateAccuracyFormula(losses)
	if blunders > 0 && acc < 60.0 {
		return ClassificationBlunder, "??"
	}
	if (mistakes > 0 || misses > 0) && acc < 70.0 {
		return ClassificationMistake, "?"
	}
	if (inaccuracies > 0 || acc < 75.0) && acc < 80.0 {
		return ClassificationInaccuracy, "?!"
	}
	if acc < 60.0 {
		return ClassificationInaccuracy, "?!"
	}
	return ClassificationGood, "✔"
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
