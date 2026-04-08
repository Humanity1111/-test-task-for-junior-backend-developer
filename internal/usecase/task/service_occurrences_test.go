package task

import (
	"context"
	"testing"
	"time"

	taskdomain "example.com/taskservice/internal/domain/task"
)

// --- stub repository ---

type stubRepo struct {
	task *taskdomain.Task
	err  error
}

func (s *stubRepo) Create(_ context.Context, t *taskdomain.Task) (*taskdomain.Task, error) {
	return t, s.err
}
func (s *stubRepo) GetByID(_ context.Context, _ int64) (*taskdomain.Task, error) {
	return s.task, s.err
}
func (s *stubRepo) Update(_ context.Context, t *taskdomain.Task) (*taskdomain.Task, error) {
	return t, s.err
}
func (s *stubRepo) Delete(_ context.Context, _ int64) error { return s.err }
func (s *stubRepo) List(_ context.Context) ([]taskdomain.Task, error) {
	if s.task != nil {
		return []taskdomain.Task{*s.task}, s.err
	}
	return nil, s.err
}

// --- helpers ---

func mustTime(s string) time.Time {
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		panic(err)
	}
	return t.UTC()
}

func ptr[T any](v T) *T { return &v }

func newServiceWithTask(t *taskdomain.Task) *Service {
	return NewService(&stubRepo{task: t})
}

// --- tests: daily ---

func TestGetOccurrences_Daily_EveryDay(t *testing.T) {
	anchor := mustTime("2026-04-01")
	svc := newServiceWithTask(&taskdomain.Task{
		ID:          1,
		ScheduledAt: &anchor,
		Recurrence:  &taskdomain.Recurrence{Type: taskdomain.RecurrenceDaily, Interval: 1},
	})

	from := mustTime("2026-04-01")
	to := mustTime("2026-04-05")
	got, err := svc.GetOccurrences(context.Background(), 1, from, to)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"2026-04-01", "2026-04-02", "2026-04-03", "2026-04-04", "2026-04-05"}
	assertDates(t, want, got)
}

func TestGetOccurrences_Daily_EveryOtherDay(t *testing.T) {
	anchor := mustTime("2026-04-01")
	svc := newServiceWithTask(&taskdomain.Task{
		ID:          1,
		ScheduledAt: &anchor,
		Recurrence:  &taskdomain.Recurrence{Type: taskdomain.RecurrenceDaily, Interval: 2},
	})

	from := mustTime("2026-04-01")
	to := mustTime("2026-04-10")
	got, err := svc.GetOccurrences(context.Background(), 1, from, to)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"2026-04-01", "2026-04-03", "2026-04-05", "2026-04-07", "2026-04-09"}
	assertDates(t, want, got)
}

func TestGetOccurrences_Daily_FromAfterAnchor(t *testing.T) {
	// Якорь до диапазона — должен правильно выровнять первую дату.
	anchor := mustTime("2026-01-01")
	svc := newServiceWithTask(&taskdomain.Task{
		ID:          1,
		ScheduledAt: &anchor,
		Recurrence:  &taskdomain.Recurrence{Type: taskdomain.RecurrenceDaily, Interval: 7},
	})

	from := mustTime("2026-04-01")
	to := mustTime("2026-04-30")
	got, err := svc.GetOccurrences(context.Background(), 1, from, to)
	if err != nil {
		t.Fatal(err)
	}
	// 2026-01-01 + 7*n: ищем первый >= 2026-04-01
	// 91 дней от 01-01 до 04-01 => 91/7=13 => 13*7=91 => 2026-04-02 (91 days), затем 2026-04-09, 2026-04-16, 2026-04-23, 2026-04-30
	want := []string{"2026-04-02", "2026-04-09", "2026-04-16", "2026-04-23", "2026-04-30"}
	assertDates(t, want, got)
}

func TestGetOccurrences_Daily_EmptyRange(t *testing.T) {
	anchor := mustTime("2026-06-01")
	svc := newServiceWithTask(&taskdomain.Task{
		ID:          1,
		ScheduledAt: &anchor,
		Recurrence:  &taskdomain.Recurrence{Type: taskdomain.RecurrenceDaily, Interval: 1},
	})

	from := mustTime("2026-04-01")
	to := mustTime("2026-04-30")
	got, err := svc.GetOccurrences(context.Background(), 1, from, to)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Errorf("expected empty slice, got %v", got)
	}
}

// --- tests: monthly ---

func TestGetOccurrences_Monthly_NormalMonth(t *testing.T) {
	svc := newServiceWithTask(&taskdomain.Task{
		ID:         1,
		Recurrence: &taskdomain.Recurrence{Type: taskdomain.RecurrenceMonthly, DayOfMonth: 15},
	})

	from := mustTime("2026-01-01")
	to := mustTime("2026-04-30")
	got, err := svc.GetOccurrences(context.Background(), 1, from, to)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"2026-01-15", "2026-02-15", "2026-03-15", "2026-04-15"}
	assertDates(t, want, got)
}

func TestGetOccurrences_Monthly_Day30_SkipsFebruary(t *testing.T) {
	svc := newServiceWithTask(&taskdomain.Task{
		ID:         1,
		Recurrence: &taskdomain.Recurrence{Type: taskdomain.RecurrenceMonthly, DayOfMonth: 30},
	})

	from := mustTime("2026-01-01")
	to := mustTime("2026-04-30")
	got, err := svc.GetOccurrences(context.Background(), 1, from, to)
	if err != nil {
		t.Fatal(err)
	}
	// Февраль 2026 не имеет 30-го числа — пропускается.
	want := []string{"2026-01-30", "2026-03-30", "2026-04-30"}
	assertDates(t, want, got)
}

func TestGetOccurrences_Monthly_FirstDay(t *testing.T) {
	svc := newServiceWithTask(&taskdomain.Task{
		ID:         1,
		Recurrence: &taskdomain.Recurrence{Type: taskdomain.RecurrenceMonthly, DayOfMonth: 1},
	})

	from := mustTime("2026-03-01")
	to := mustTime("2026-05-01")
	got, err := svc.GetOccurrences(context.Background(), 1, from, to)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"2026-03-01", "2026-04-01", "2026-05-01"}
	assertDates(t, want, got)
}

// --- tests: dates ---

func TestGetOccurrences_Dates_InRange(t *testing.T) {
	svc := newServiceWithTask(&taskdomain.Task{
		ID: 1,
		Recurrence: &taskdomain.Recurrence{
			Type:  taskdomain.RecurrenceDates,
			Dates: []string{"2026-04-05", "2026-04-15", "2026-05-01", "2026-03-01"},
		},
	})

	from := mustTime("2026-04-01")
	to := mustTime("2026-04-30")
	got, err := svc.GetOccurrences(context.Background(), 1, from, to)
	if err != nil {
		t.Fatal(err)
	}
	// Только даты внутри диапазона.
	want := []string{"2026-04-05", "2026-04-15"}
	assertDates(t, want, got)
}

func TestGetOccurrences_Dates_NoneInRange(t *testing.T) {
	svc := newServiceWithTask(&taskdomain.Task{
		ID: 1,
		Recurrence: &taskdomain.Recurrence{
			Type:  taskdomain.RecurrenceDates,
			Dates: []string{"2026-01-01", "2026-12-31"},
		},
	})

	from := mustTime("2026-04-01")
	to := mustTime("2026-04-30")
	got, err := svc.GetOccurrences(context.Background(), 1, from, to)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Errorf("expected empty, got %v", got)
	}
}

func TestGetOccurrences_Dates_InvalidFormat(t *testing.T) {
	svc := newServiceWithTask(&taskdomain.Task{
		ID: 1,
		Recurrence: &taskdomain.Recurrence{
			Type:  taskdomain.RecurrenceDates,
			Dates: []string{"not-a-date"},
		},
	})

	_, err := svc.GetOccurrences(context.Background(), 1, mustTime("2026-04-01"), mustTime("2026-04-30"))
	if err == nil {
		t.Fatal("expected error for invalid date format")
	}
}

// --- tests: even_odd ---

func TestGetOccurrences_Even_April(t *testing.T) {
	svc := newServiceWithTask(&taskdomain.Task{
		ID:         1,
		Recurrence: &taskdomain.Recurrence{Type: taskdomain.RecurrenceEvenOdd, Parity: taskdomain.ParityEven},
	})

	from := mustTime("2026-04-01")
	to := mustTime("2026-04-07")
	got, err := svc.GetOccurrences(context.Background(), 1, from, to)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"2026-04-02", "2026-04-04", "2026-04-06"}
	assertDates(t, want, got)
}

func TestGetOccurrences_Odd_April(t *testing.T) {
	svc := newServiceWithTask(&taskdomain.Task{
		ID:         1,
		Recurrence: &taskdomain.Recurrence{Type: taskdomain.RecurrenceEvenOdd, Parity: taskdomain.ParityOdd},
	})

	from := mustTime("2026-04-01")
	to := mustTime("2026-04-07")
	got, err := svc.GetOccurrences(context.Background(), 1, from, to)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"2026-04-01", "2026-04-03", "2026-04-05", "2026-04-07"}
	assertDates(t, want, got)
}

// --- tests: no recurrence ---

func TestGetOccurrences_NoRecurrence_ScheduledAtInRange(t *testing.T) {
	scheduled := mustTime("2026-04-15")
	svc := newServiceWithTask(&taskdomain.Task{
		ID:          1,
		ScheduledAt: &scheduled,
	})

	from := mustTime("2026-04-01")
	to := mustTime("2026-04-30")
	got, err := svc.GetOccurrences(context.Background(), 1, from, to)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"2026-04-15"}
	assertDates(t, want, got)
}

func TestGetOccurrences_NoRecurrence_ScheduledAtOutOfRange(t *testing.T) {
	scheduled := mustTime("2026-05-15")
	svc := newServiceWithTask(&taskdomain.Task{
		ID:          1,
		ScheduledAt: &scheduled,
	})

	from := mustTime("2026-04-01")
	to := mustTime("2026-04-30")
	got, err := svc.GetOccurrences(context.Background(), 1, from, to)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Errorf("expected empty, got %v", got)
	}
}

func TestGetOccurrences_NoRecurrence_NoScheduledAt(t *testing.T) {
	svc := newServiceWithTask(&taskdomain.Task{ID: 1})

	from := mustTime("2026-04-01")
	to := mustTime("2026-04-30")
	got, err := svc.GetOccurrences(context.Background(), 1, from, to)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Errorf("expected empty, got %v", got)
	}
}

// --- tests: validation ---

func TestGetOccurrences_InvalidRange_FromAfterTo(t *testing.T) {
	svc := newServiceWithTask(&taskdomain.Task{ID: 1})

	_, err := svc.GetOccurrences(context.Background(), 1, mustTime("2026-04-30"), mustTime("2026-04-01"))
	if err == nil {
		t.Fatal("expected error when from > to")
	}
}

func TestGetOccurrences_InvalidRange_TooLong(t *testing.T) {
	svc := newServiceWithTask(&taskdomain.Task{ID: 1})

	_, err := svc.GetOccurrences(context.Background(), 1, mustTime("2026-01-01"), mustTime("2027-02-01"))
	if err == nil {
		t.Fatal("expected error when range > 1 year")
	}
}

func TestGetOccurrences_InvalidID(t *testing.T) {
	svc := newServiceWithTask(nil)

	_, err := svc.GetOccurrences(context.Background(), 0, mustTime("2026-04-01"), mustTime("2026-04-30"))
	if err == nil {
		t.Fatal("expected error for id=0")
	}
}

// --- tests: validateRecurrence ---

func TestValidateRecurrence_Daily_MissingScheduledAt(t *testing.T) {
	rec := &taskdomain.Recurrence{Type: taskdomain.RecurrenceDaily, Interval: 1}
	if err := validateRecurrence(rec, nil); err == nil {
		t.Fatal("expected error: daily requires scheduled_at")
	}
}

func TestValidateRecurrence_Daily_ZeroInterval(t *testing.T) {
	rec := &taskdomain.Recurrence{Type: taskdomain.RecurrenceDaily, Interval: 0}
	anchor := mustTime("2026-04-01")
	if err := validateRecurrence(rec, &anchor); err == nil {
		t.Fatal("expected error: interval must be >= 1")
	}
}

func TestValidateRecurrence_Monthly_InvalidDay(t *testing.T) {
	for _, day := range []int{0, 31, -1} {
		rec := &taskdomain.Recurrence{Type: taskdomain.RecurrenceMonthly, DayOfMonth: day}
		if err := validateRecurrence(rec, nil); err == nil {
			t.Fatalf("expected error for day_of_month=%d", day)
		}
	}
}

func TestValidateRecurrence_Dates_Empty(t *testing.T) {
	rec := &taskdomain.Recurrence{Type: taskdomain.RecurrenceDates, Dates: nil}
	if err := validateRecurrence(rec, nil); err == nil {
		t.Fatal("expected error: dates must not be empty")
	}
}

func TestValidateRecurrence_Dates_InvalidFormat(t *testing.T) {
	rec := &taskdomain.Recurrence{Type: taskdomain.RecurrenceDates, Dates: []string{"01-04-2026"}}
	if err := validateRecurrence(rec, nil); err == nil {
		t.Fatal("expected error: invalid date format")
	}
}

func TestValidateRecurrence_EvenOdd_MissingParity(t *testing.T) {
	rec := &taskdomain.Recurrence{Type: taskdomain.RecurrenceEvenOdd}
	if err := validateRecurrence(rec, nil); err == nil {
		t.Fatal("expected error: parity required")
	}
}

func TestValidateRecurrence_Nil(t *testing.T) {
	if err := validateRecurrence(nil, nil); err != nil {
		t.Fatalf("unexpected error for nil recurrence: %v", err)
	}
}

// --- helper ---

func assertDates(t *testing.T, want []string, got []time.Time) {
	t.Helper()
	if len(want) != len(got) {
		t.Fatalf("length mismatch: want %d (%v), got %d (%v)", len(want), want, len(got), formatDates(got))
	}
	for i, w := range want {
		if got[i].Format("2006-01-02") != w {
			t.Errorf("[%d] want %s, got %s", i, w, got[i].Format("2006-01-02"))
		}
	}
}

func formatDates(dates []time.Time) []string {
	result := make([]string, len(dates))
	for i, d := range dates {
		result[i] = d.Format("2006-01-02")
	}
	return result
}
