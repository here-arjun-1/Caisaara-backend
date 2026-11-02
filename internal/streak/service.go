package streak

import (
	"context"
	"time"
)

type StreakRepository interface {
	GetOrCreateStreak(ctx context.Context, userID int64, defaultTz string) (*UserStreak, error)
	GetStreakByUserID(ctx context.Context, userID int64) (*UserStreak, error)
	UpdateStreak(ctx context.Context, s *UserStreak) error
	RecordActivityDay(ctx context.Context, userID int64, date string, activityType string) error
	GetActivityDaysInRange(ctx context.Context, userID int64, startDate, endDate string) (map[string]string, error)
}

type Service struct {
	repo StreakRepository
}

func NewService(repo StreakRepository) *Service {
	return &Service{repo: repo}
}

func (s *Service) resolveTime(tz string) (*time.Location, string, string, string) {
	loc, err := time.LoadLocation(tz)
	if err != nil || tz == "" {
		loc, _ = time.LoadLocation("Asia/Kolkata")
		tz = "Asia/Kolkata"
	}
	now := time.Now().In(loc)
	return loc, tz, now.Format("2006-01-02"), now.AddDate(0, 0, -1).Format("2006-01-02")
}

func daysDiff(from, to string) int {
	t1, err1 := time.Parse("2006-01-02", from)
	t2, err2 := time.Parse("2006-01-02", to)
	if err1 != nil || err2 != nil {
		return 0
	}
	return int(t2.Sub(t1).Hours() / 24)
}

func (s *Service) applyLazyEvaluation(ctx context.Context, streak *UserStreak, today string) {
	if streak.LastActivityDate == "" {
		return
	}
	days := daysDiff(streak.LastActivityDate, today)
	switch {
	case days <= 1:
		if streak.StreakStatus != StreakActive {
			streak.StreakStatus = StreakActive
			_ = s.repo.UpdateStreak(ctx, streak)
		}
	case days <= 3:
		if streak.StreakStatus != StreakPaused {
			streak.StreakStatus = StreakPaused
			_ = s.repo.UpdateStreak(ctx, streak)
		}
	default:
		if streak.CurrentStreak != 0 || streak.StreakStatus != StreakBroken {
			streak.CurrentStreak = 0
			streak.StreakStatus = StreakBroken
			_ = s.repo.UpdateStreak(ctx, streak)
		}
	}
}

func (s *Service) RecordActivity(ctx context.Context, userID int64, activityType string) (*StreakUpdate, error) {
	if userID <= 0 {
		return nil, nil
	}
	if activityType == "" {
		activityType = "game"
	}

	streak, err := s.repo.GetOrCreateStreak(ctx, userID, "Asia/Kolkata")
	if err != nil {
		return nil, err
	}

	_, _, today, _ := s.resolveTime(streak.Timezone)

	if streak.LastActivityDate == today {
		_ = s.repo.RecordActivityDay(ctx, userID, today, activityType)
		return &StreakUpdate{
			CurrentStreak:  streak.CurrentStreak,
			BestStreak:     streak.BestStreak,
			StreakStatus:   streak.StreakStatus,
			Incremented:    false,
			TodayCompleted: true,
		}, nil
	}

	days := daysDiff(streak.LastActivityDate, today)
	incremented := false

	switch {
	case streak.LastActivityDate == "":
		streak.CurrentStreak = 1
		streak.StreakStatus = StreakActive
		incremented = true
	case days == 1:
		streak.CurrentStreak++
		streak.StreakStatus = StreakActive
		incremented = true
	case days <= 3:
		streak.CurrentStreak++
		streak.StreakStatus = StreakActive
		incremented = true
		loc, _ := time.LoadLocation(streak.Timezone)
		if loc == nil {
			loc = time.UTC
		}
		tLast, parseErr := time.ParseInLocation("2006-01-02", streak.LastActivityDate, loc)
		if parseErr == nil {
			for d := 1; d < days; d++ {
				gapDate := tLast.AddDate(0, 0, d).Format("2006-01-02")
				_ = s.repo.RecordActivityDay(ctx, userID, gapDate, "freeze")
			}
		}
		missedDays := days - 1
		if streak.FreezesAvailable > 0 {
			streak.FreezesAvailable -= missedDays
			if streak.FreezesAvailable < 0 {
				streak.FreezesAvailable = 0
			}
		}
	default:
		streak.CurrentStreak = 1
		streak.StreakStatus = StreakActive
		incremented = true
	}

	if streak.CurrentStreak > streak.BestStreak {
		streak.BestStreak = streak.CurrentStreak
	}
	streak.MaxStreak = streak.BestStreak
	streak.LastActivityDate = today

	_ = s.repo.RecordActivityDay(ctx, userID, today, activityType)
	_ = s.repo.UpdateStreak(ctx, streak)

	return &StreakUpdate{
		CurrentStreak:  streak.CurrentStreak,
		BestStreak:     streak.BestStreak,
		StreakStatus:   streak.StreakStatus,
		Incremented:    incremented,
		TodayCompleted: true,
	}, nil
}

func (s *Service) OnGameFinish(ctx context.Context, userID int64) (*StreakUpdate, error) {
	return s.RecordActivity(ctx, userID, "game")
}

func (s *Service) GetStreak(ctx context.Context, userID int64, tz string) (*StreakResponse, error) {
	streak, err := s.repo.GetOrCreateStreak(ctx, userID, "Asia/Kolkata")
	if err != nil {
		return nil, err
	}
	if tz != "" && tz != streak.Timezone {
		if _, err := time.LoadLocation(tz); err == nil {
			streak.Timezone = tz
			_ = s.repo.UpdateStreak(ctx, streak)
		}
	}

	loc, tzName, today, _ := s.resolveTime(streak.Timezone)
	s.applyLazyEvaluation(ctx, streak, today)

	todayCompleted := streak.LastActivityDate == today
	week, err := s.buildWeek(ctx, streak, loc, today, todayCompleted)
	if err != nil {
		return nil, err
	}

	freezes := streak.FreezesAvailable
	if streak.LastActivityDate != "" {
		days := daysDiff(streak.LastActivityDate, today)
		if days > 1 && days <= 3 {
			used := days - 1
			if used <= freezes {
				freezes -= used
			} else {
				freezes = 0
			}
		}
	}

	return &StreakResponse{
		CurrentStreak:    streak.CurrentStreak,
		MaxStreak:        streak.BestStreak,
		BestStreak:       streak.BestStreak,
		TodayCompleted:   todayCompleted,
		StreakStatus:     streak.StreakStatus,
		Timezone:         tzName,
		FreezesAvailable: freezes,
		Week:             week,
	}, nil
}

func (s *Service) GetStreakUpdate(ctx context.Context, userID int64) (*StreakUpdate, error) {
	streak, err := s.repo.GetStreakByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	_, _, today, _ := s.resolveTime(streak.Timezone)
	return &StreakUpdate{
		CurrentStreak:  streak.CurrentStreak,
		BestStreak:     streak.BestStreak,
		StreakStatus:   streak.StreakStatus,
		Incremented:    false,
		TodayCompleted: streak.LastActivityDate == today,
	}, nil
}

func (s *Service) buildWeek(ctx context.Context, streak *UserStreak, loc *time.Location, today string, todayCompleted bool) ([]WeekDayItem, error) {
	todayTime, _ := time.ParseInLocation("2006-01-02", today, loc)
	start := todayTime.AddDate(0, 0, -3).Format("2006-01-02")
	end := todayTime.AddDate(0, 0, 3).Format("2006-01-02")

	activities, _ := s.repo.GetActivityDaysInRange(ctx, streak.UserID, start, end)

	weekdays := []string{"S", "M", "T", "W", "T", "F", "S"}
	week := make([]WeekDayItem, 0, 7)
	for i := -3; i <= 3; i++ {
		cur := todayTime.AddDate(0, 0, i)
		dStr := cur.Format("2006-01-02")

		var status string
		switch {
		case dStr > today:
			status = StatusEmpty
		case dStr == today:
			if todayCompleted {
				status = StatusPlayed
			} else {
				status = StatusEmpty
			}
		default:
			if act, ok := activities[dStr]; ok {
				if act == "freeze" {
					status = StatusFrozen
				} else {
					status = StatusPlayed
				}
			} else if streak.CurrentStreak > 0 && streak.LastActivityDate != "" && dStr > streak.LastActivityDate && daysDiff(streak.LastActivityDate, dStr) <= 2 {
				status = StatusFrozen
			} else {
				status = StatusMissed
			}
		}

		week = append(week, WeekDayItem{
			Date:    dStr,
			Weekday: weekdays[cur.Weekday()],
			Status:  status,
		})
	}
	return week, nil
}
