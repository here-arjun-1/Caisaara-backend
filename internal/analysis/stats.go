package analysis

import (
	"math"
)

type SummaryStatistics struct {
	AccuracyWhite     float64 `json:"accuracy_white"`
	AccuracyBlack     float64 `json:"accuracy_black"`
	BestWhite         int     `json:"best_white"`
	GoodWhite         int     `json:"good_white"`
	InaccuraciesWhite int     `json:"inaccuracies_white"`
	MistakesWhite     int     `json:"mistakes_white"`
	BlundersWhite     int     `json:"blunders_white"`
	BestBlack         int     `json:"best_black"`
	GoodBlack         int     `json:"good_black"`
	InaccuraciesBlack int     `json:"inaccuracies_black"`
	MistakesBlack     int     `json:"mistakes_black"`
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

func (s *StatisticsService) CalculateStatistics(moves []GameMoveAnalysis) (SummaryStatistics, []EvalHistoryPoint) {
	var stats SummaryStatistics
	evalHistory := make([]EvalHistoryPoint, 0, len(moves))

	whiteLosses := []float64{}
	blackLosses := []float64{}

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

		if m.PlayerColor == "white" {
			whiteLosses = append(whiteLosses, float64(m.CentipawnLoss))
			switch m.Classification {
			case ClassificationBest:
				stats.BestWhite++
			case ClassificationGood:
				stats.GoodWhite++
			case ClassificationInaccuracy:
				stats.InaccuraciesWhite++
			case ClassificationMistake:
				stats.MistakesWhite++
			case ClassificationBlunder:
				stats.BlundersWhite++
			}
		} else {
			blackLosses = append(blackLosses, float64(m.CentipawnLoss))
			switch m.Classification {
			case ClassificationBest:
				stats.BestBlack++
			case ClassificationGood:
				stats.GoodBlack++
			case ClassificationInaccuracy:
				stats.InaccuraciesBlack++
			case ClassificationMistake:
				stats.MistakesBlack++
			case ClassificationBlunder:
				stats.BlundersBlack++
			}
		}
	}

	stats.AccuracyWhite = CalculateAccuracyFormula(whiteLosses)
	stats.AccuracyBlack = CalculateAccuracyFormula(blackLosses)

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
