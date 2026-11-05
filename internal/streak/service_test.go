package streak

import (
	"context"
	"errors"
	"testing"
	"time"
)

type mockRepo struct {
	streaks map[int64]*UserStreak
	days    map[string]string
}

func newMockRepo() *mockRepo {
	return &mockRepo{
		streaks: make(map[int64]*UserStreak),
		days:    make(map[string]string),
	}
}

func (m *mockRepo) GetOrCreateStreak(ctx context.Context, userID int64, defaultTz string) (*UserStreak, error) {
	if s, ok := m.streaks[userID]; ok {
		return s, nil
	}
	s := &UserStreak{
		UserID:           userID,
		CurrentStreak:    0,
		BestStreak:       0,
		MaxStreak:        0,
		FreezesAvailable: 2,
		StreakStatus:     StreakActive,
		Timezone:         defaultTz,
		UpdatedAt:        time.Now(),
	}
	m.streaks[userID] = s
	return s, nil
}

func (m *mockRepo) GetStreakByUserID(ctx context.Context, userID int64) (*UserStreak, error) {
	if s, ok := m.streaks[userID]; ok {
		return s, nil
	}
	return nil, errors.New("not found")
}

func (m *mockRepo) UpdateStreak(ctx context.Context, s *UserStreak) error {
	m.streaks[s.UserID] = s
	return nil
}

func (m *mockRepo) RecordActivityDay(ctx context.Context, userID int64, date string, activityType string) error {
	m.days[date] = activityType
	return nil
}

func (m *mockRepo) GetActivityDaysInRange(ctx context.Context, userID int64, startDate, endDate string) (map[string]string, error) {
	res := make(map[string]string)
	for k, v := range m.days {
		if k >= startDate && k <= endDate {
			res[k] = v
		}
	}
	return res, nil
}

func TestStreakLifecycle(t *testing.T) {
	ctx := context.Background()
	repo := newMockRepo()
	svc := NewService(repo)

	up1, err := svc.OnGameFinish(ctx, 1)
	if err != nil {
		t.Fatalf("first finish: %v", err)
	}
	if up1.CurrentStreak != 1 || !up1.Incremented || !up1.TodayCompleted {
		t.Fatalf("unexpected up1: %+v", up1)
	}

	up2, err := svc.OnGameFinish(ctx, 1)
	if err != nil {
		t.Fatalf("second finish: %v", err)
	}
	if up2.CurrentStreak != 1 || up2.Incremented || !up2.TodayCompleted {
		t.Fatalf("unexpected up2: %+v", up2)
	}

	res, err := svc.GetStreak(ctx, 1, "")
	if err != nil {
		t.Fatalf("get streak: %v", err)
	}
	if res.CurrentStreak != 1 || !res.TodayCompleted || res.FreezesAvailable != 2 {
		t.Fatalf("unexpected res: %+v", res)
	}
	if len(res.Week) != 7 {
		t.Fatalf("week length: %d", len(res.Week))
	}
}

func TestStreakConsecutiveDays(t *testing.T) {
	ctx := context.Background()
	repo := newMockRepo()
	svc := NewService(repo)

	loc, _ := time.LoadLocation("Asia/Kolkata")
	yesterday := time.Now().In(loc).AddDate(0, 0, -1).Format("2006-01-02")
	repo.streaks[1] = &UserStreak{
		UserID:           1,
		CurrentStreak:    3,
		BestStreak:       3,
		MaxStreak:        3,
		FreezesAvailable: 2,
		LastActivityDate: yesterday,
		StreakStatus:     StreakActive,
		Timezone:         "Asia/Kolkata",
	}

	res, err := svc.GetStreak(ctx, 1, "")
	if err != nil {
		t.Fatalf("get streak: %v", err)
	}
	if res.CurrentStreak != 3 || res.TodayCompleted {
		t.Fatalf("consecutive check failed: %+v", res)
	}

	up, err := svc.RecordActivity(ctx, 1, "game")
	if err != nil {
		t.Fatalf("record activity: %v", err)
	}
	if up.CurrentStreak != 4 || !up.Incremented || !up.TodayCompleted {
		t.Fatalf("unexpected up: %+v", up)
	}
}

func TestStreakGapGracePeriod(t *testing.T) {
	ctx := context.Background()
	repo := newMockRepo()
	svc := NewService(repo)

	loc, _ := time.LoadLocation("Asia/Kolkata")
	twoDaysAgo := time.Now().In(loc).AddDate(0, 0, -2).Format("2006-01-02")
	repo.streaks[1] = &UserStreak{
		UserID:           1,
		CurrentStreak:    5,
		BestStreak:       5,
		MaxStreak:        5,
		FreezesAvailable: 2,
		LastActivityDate: twoDaysAgo,
		StreakStatus:     StreakActive,
		Timezone:         "Asia/Kolkata",
	}

	res, err := svc.GetStreak(ctx, 1, "")
	if err != nil {
		t.Fatalf("get streak: %v", err)
	}
	if res.CurrentStreak != 5 || res.StreakStatus != StreakPaused || res.FreezesAvailable != 1 {
		t.Fatalf("grace period check failed: %+v", res)
	}

	up, err := svc.RecordActivity(ctx, 1, "game")
	if err != nil {
		t.Fatalf("record activity: %v", err)
	}
	if up.CurrentStreak != 6 || !up.Incremented || !up.TodayCompleted {
		t.Fatalf("unexpected up after grace period: %+v", up)
	}

	resAfter, err := svc.GetStreak(ctx, 1, "")
	if err != nil {
		t.Fatalf("get streak after: %v", err)
	}
	if resAfter.CurrentStreak != 6 || resAfter.StreakStatus != StreakActive {
		t.Fatalf("streak should be active after move: %+v", resAfter)
	}
}

func TestStreakGap3DaysGracePeriod(t *testing.T) {
	ctx := context.Background()
	repo := newMockRepo()
	svc := NewService(repo)

	loc, _ := time.LoadLocation("Asia/Kolkata")
	threeDaysAgo := time.Now().In(loc).AddDate(0, 0, -3).Format("2006-01-02")
	repo.streaks[1] = &UserStreak{
		UserID:           1,
		CurrentStreak:    4,
		BestStreak:       4,
		MaxStreak:        4,
		FreezesAvailable: 2,
		LastActivityDate: threeDaysAgo,
		StreakStatus:     StreakActive,
		Timezone:         "Asia/Kolkata",
	}

	res, err := svc.GetStreak(ctx, 1, "")
	if err != nil {
		t.Fatalf("get streak: %v", err)
	}
	if res.CurrentStreak != 4 || res.StreakStatus != StreakPaused {
		t.Fatalf("should be paused on 3rd day: %+v", res)
	}

	up, err := svc.RecordActivity(ctx, 1, "game")
	if err != nil {
		t.Fatalf("activity record: %v", err)
	}
	if up.CurrentStreak != 5 || !up.Incremented {
		t.Fatalf("streak should continue on 3rd day: %+v", up)
	}
}

func TestStreakGapBeyondGracePeriod(t *testing.T) {
	ctx := context.Background()
	repo := newMockRepo()
	svc := NewService(repo)

	loc, _ := time.LoadLocation("Asia/Kolkata")
	fourDaysAgo := time.Now().In(loc).AddDate(0, 0, -4).Format("2006-01-02")
	repo.streaks[1] = &UserStreak{
		UserID:           1,
		CurrentStreak:    5,
		BestStreak:       5,
		MaxStreak:        5,
		FreezesAvailable: 0,
		LastActivityDate: fourDaysAgo,
		StreakStatus:     StreakActive,
		Timezone:         "Asia/Kolkata",
	}

	res, err := svc.GetStreak(ctx, 1, "")
	if err != nil {
		t.Fatalf("get streak: %v", err)
	}
	if res.CurrentStreak != 0 || res.StreakStatus != StreakBroken {
		t.Fatalf("streak should have reset to 0: %+v", res)
	}

	up, err := svc.RecordActivity(ctx, 1, "game")
	if err != nil {
		t.Fatalf("activity record: %v", err)
	}
	if up.CurrentStreak != 1 || !up.Incremented || !up.TodayCompleted {
		t.Fatalf("streak should be reset to 1 on activity: %+v", up)
	}
}
