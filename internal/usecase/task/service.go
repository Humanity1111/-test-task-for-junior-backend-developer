package task

import (
	"context"
	"fmt"
	"strings"
	"time"

	taskdomain "example.com/taskservice/internal/domain/task"
)

const (
	maxOccurrenceRange = 366 * 24 * time.Hour // максимум 1 год в запросе
	maxOccurrences     = 366                  // максимум дат в ответе
)

type Service struct {
	repo Repository
	now  func() time.Time
}

func NewService(repo Repository) *Service {
	return &Service{
		repo: repo,
		now:  func() time.Time { return time.Now().UTC() },
	}
}

func (s *Service) Create(ctx context.Context, input CreateInput) (*taskdomain.Task, error) {
	normalized, err := validateCreateInput(input)
	if err != nil {
		return nil, err
	}

	model := &taskdomain.Task{
		Title:       normalized.Title,
		Description: normalized.Description,
		Status:      normalized.Status,
		ScheduledAt: normalized.ScheduledAt,
		Recurrence:  normalized.Recurrence,
	}
	now := s.now()
	model.CreatedAt = now
	model.UpdatedAt = now

	created, err := s.repo.Create(ctx, model)
	if err != nil {
		return nil, err
	}

	return created, nil
}

func (s *Service) GetByID(ctx context.Context, id int64) (*taskdomain.Task, error) {
	if id <= 0 {
		return nil, fmt.Errorf("%w: id must be positive", ErrInvalidInput)
	}

	return s.repo.GetByID(ctx, id)
}

func (s *Service) Update(ctx context.Context, id int64, input UpdateInput) (*taskdomain.Task, error) {
	if id <= 0 {
		return nil, fmt.Errorf("%w: id must be positive", ErrInvalidInput)
	}

	normalized, err := validateUpdateInput(input)
	if err != nil {
		return nil, err
	}

	model := &taskdomain.Task{
		ID:          id,
		Title:       normalized.Title,
		Description: normalized.Description,
		Status:      normalized.Status,
		ScheduledAt: normalized.ScheduledAt,
		Recurrence:  normalized.Recurrence,
		UpdatedAt:   s.now(),
	}

	updated, err := s.repo.Update(ctx, model)
	if err != nil {
		return nil, err
	}

	return updated, nil
}

func (s *Service) Delete(ctx context.Context, id int64) error {
	if id <= 0 {
		return fmt.Errorf("%w: id must be positive", ErrInvalidInput)
	}

	return s.repo.Delete(ctx, id)
}

func (s *Service) List(ctx context.Context) ([]taskdomain.Task, error) {
	return s.repo.List(ctx)
}

// GetOccurrences возвращает даты выполнения задачи в диапазоне [from, to] включительно.
// Время нормализуется до начала дня в UTC.
// Для задач без recurrence возвращает scheduled_at, если он попадает в диапазон.
func (s *Service) GetOccurrences(ctx context.Context, id int64, from, to time.Time) ([]time.Time, error) {
	if id <= 0 {
		return nil, fmt.Errorf("%w: id must be positive", ErrInvalidInput)
	}

	from = truncateToDay(from.UTC())
	to = truncateToDay(to.UTC())

	if from.After(to) {
		return nil, fmt.Errorf("%w: from must be before or equal to to", ErrInvalidInput)
	}

	if to.Sub(from) > maxOccurrenceRange {
		return nil, fmt.Errorf("%w: date range must not exceed 1 year", ErrInvalidInput)
	}

	task, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Задача без настроек повторяемости: разовая дата scheduled_at.
	if task.Recurrence == nil {
		if task.ScheduledAt == nil {
			return []time.Time{}, nil
		}
		d := truncateToDay(task.ScheduledAt.UTC())
		if !d.Before(from) && !d.After(to) {
			return []time.Time{d}, nil
		}
		return []time.Time{}, nil
	}

	return computeOccurrences(task.Recurrence, task.ScheduledAt, from, to)
}

// computeOccurrences вычисляет даты повторяемости внутри [from, to].
func computeOccurrences(rec *taskdomain.Recurrence, scheduledAt *time.Time, from, to time.Time) ([]time.Time, error) {
	switch rec.Type {
	case taskdomain.RecurrenceDaily:
		return computeDaily(rec.Interval, scheduledAt, from, to)
	case taskdomain.RecurrenceMonthly:
		return computeMonthly(rec.DayOfMonth, from, to), nil
	case taskdomain.RecurrenceDates:
		return computeDates(rec.Dates, from, to)
	case taskdomain.RecurrenceEvenOdd:
		return computeEvenOdd(rec.Parity, from, to), nil
	default:
		return nil, fmt.Errorf("%w: unknown recurrence type", ErrInvalidInput)
	}
}

// computeDaily возвращает даты каждые interval дней начиная с якорной даты (scheduled_at).
func computeDaily(interval int, anchor *time.Time, from, to time.Time) ([]time.Time, error) {
	if anchor == nil {
		return nil, fmt.Errorf("%w: scheduled_at is required for daily recurrence", ErrInvalidInput)
	}

	start := truncateToDay(anchor.UTC())
	var result []time.Time

	// Найти первую дату >= from, выровненную по интервалу относительно anchor.
	diff := int(from.Sub(start).Hours() / 24)
	if diff < 0 {
		diff = 0
	}

	// Округлить diff до ближайшего кратного interval в бо́льшую сторону.
	remainder := diff % interval
	if remainder != 0 {
		diff += interval - remainder
	}

	d := start.AddDate(0, 0, diff)
	for !d.After(to) {
		result = append(result, d)
		if len(result) >= maxOccurrences {
			break
		}
		d = d.AddDate(0, 0, interval)
	}

	if result == nil {
		return []time.Time{}, nil
	}
	return result, nil
}

// computeMonthly возвращает даты на указанное число каждого месяца в диапазоне.
func computeMonthly(dayOfMonth int, from, to time.Time) []time.Time {
	var result []time.Time

	year, month, _ := from.Date()
	// Перебираем каждый месяц в диапазоне.
	cur := time.Date(year, month, 1, 0, 0, 0, 0, time.UTC)
	for !cur.After(to) {
		// Проверяем, что такое число есть в этом месяце.
		d := time.Date(cur.Year(), cur.Month(), dayOfMonth, 0, 0, 0, 0, time.UTC)
		// Если день "перелился" в следующий месяц, значит такого числа нет — пропускаем.
		if d.Month() == cur.Month() && !d.Before(from) && !d.After(to) {
			result = append(result, d)
		}
		cur = cur.AddDate(0, 1, 0)
	}

	if result == nil {
		return []time.Time{}
	}
	return result
}

// computeDates возвращает из явного списка дат те, что попадают в диапазон.
func computeDates(dates []string, from, to time.Time) ([]time.Time, error) {
	var result []time.Time
	for _, raw := range dates {
		d, err := time.Parse("2006-01-02", raw)
		if err != nil {
			return nil, fmt.Errorf("%w: invalid date %q in dates list", ErrInvalidInput, raw)
		}
		d = d.UTC()
		if !d.Before(from) && !d.After(to) {
			result = append(result, d)
		}
	}
	if result == nil {
		return []time.Time{}, nil
	}
	return result, nil
}

// computeEvenOdd возвращает все чётные или нечётные числа месяца в диапазоне.
func computeEvenOdd(parity taskdomain.RecurrenceParity, from, to time.Time) []time.Time {
	var result []time.Time
	for d := from; !d.After(to); d = d.AddDate(0, 0, 1) {
		dayNum := d.Day()
		isEven := dayNum%2 == 0
		if (parity == taskdomain.ParityEven && isEven) || (parity == taskdomain.ParityOdd && !isEven) {
			result = append(result, d)
			if len(result) >= maxOccurrences {
				break
			}
		}
	}
	if result == nil {
		return []time.Time{}
	}
	return result
}

// truncateToDay возвращает дату с обнулённым временем (00:00:00 UTC).
func truncateToDay(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

func validateCreateInput(input CreateInput) (CreateInput, error) {
	input.Title = strings.TrimSpace(input.Title)
	input.Description = strings.TrimSpace(input.Description)

	if input.Title == "" {
		return CreateInput{}, fmt.Errorf("%w: title is required", ErrInvalidInput)
	}

	if input.Status == "" {
		input.Status = taskdomain.StatusNew
	}

	if !input.Status.Valid() {
		return CreateInput{}, fmt.Errorf("%w: invalid status", ErrInvalidInput)
	}

	if err := validateRecurrence(input.Recurrence, input.ScheduledAt); err != nil {
		return CreateInput{}, err
	}

	return input, nil
}

func validateUpdateInput(input UpdateInput) (UpdateInput, error) {
	input.Title = strings.TrimSpace(input.Title)
	input.Description = strings.TrimSpace(input.Description)

	if input.Title == "" {
		return UpdateInput{}, fmt.Errorf("%w: title is required", ErrInvalidInput)
	}

	if !input.Status.Valid() {
		return UpdateInput{}, fmt.Errorf("%w: invalid status", ErrInvalidInput)
	}

	if err := validateRecurrence(input.Recurrence, input.ScheduledAt); err != nil {
		return UpdateInput{}, err
	}

	return input, nil
}

// validateRecurrence проверяет корректность настроек повторяемости.
func validateRecurrence(rec *taskdomain.Recurrence, scheduledAt *time.Time) error {
	if rec == nil {
		return nil
	}

	if !rec.Type.Valid() {
		return fmt.Errorf("%w: invalid recurrence type %q", ErrInvalidInput, rec.Type)
	}

	switch rec.Type {
	case taskdomain.RecurrenceDaily:
		if scheduledAt == nil {
			return fmt.Errorf("%w: scheduled_at is required for daily recurrence", ErrInvalidInput)
		}
		if rec.Interval < 1 {
			return fmt.Errorf("%w: recurrence interval must be >= 1", ErrInvalidInput)
		}

	case taskdomain.RecurrenceMonthly:
		if rec.DayOfMonth < 1 || rec.DayOfMonth > 30 {
			return fmt.Errorf("%w: day_of_month must be between 1 and 30", ErrInvalidInput)
		}

	case taskdomain.RecurrenceDates:
		if len(rec.Dates) == 0 {
			return fmt.Errorf("%w: dates list must not be empty for dates recurrence", ErrInvalidInput)
		}
		for _, raw := range rec.Dates {
			if _, err := time.Parse("2006-01-02", raw); err != nil {
				return fmt.Errorf("%w: invalid date %q in dates list (expected YYYY-MM-DD)", ErrInvalidInput, raw)
			}
		}

	case taskdomain.RecurrenceEvenOdd:
		if !rec.Parity.Valid() {
			return fmt.Errorf("%w: parity must be \"even\" or \"odd\"", ErrInvalidInput)
		}
	}

	return nil
}
