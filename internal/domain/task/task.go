package task

import "time"

type Status string

const (
	StatusNew        Status = "new"
	StatusInProgress Status = "in_progress"
	StatusDone       Status = "done"
)

// RecurrenceType определяет тип повторяемости задачи.
type RecurrenceType string

const (
	// RecurrenceDaily — каждые N дней, начиная с scheduled_at.
	RecurrenceDaily RecurrenceType = "daily"
	// RecurrenceMonthly — каждое указанное число месяца (1–30).
	RecurrenceMonthly RecurrenceType = "monthly"
	// RecurrenceDates — задача только на конкретные даты.
	RecurrenceDates RecurrenceType = "dates"
	// RecurrenceEvenOdd — по чётным или нечётным числам месяца.
	RecurrenceEvenOdd RecurrenceType = "even_odd"
)

func (t RecurrenceType) Valid() bool {
	switch t {
	case RecurrenceDaily, RecurrenceMonthly, RecurrenceDates, RecurrenceEvenOdd:
		return true
	default:
		return false
	}
}

// RecurrenceParity используется для типа even_odd.
type RecurrenceParity string

const (
	ParityEven RecurrenceParity = "even"
	ParityOdd  RecurrenceParity = "odd"
)

func (p RecurrenceParity) Valid() bool {
	return p == ParityEven || p == ParityOdd
}

// Recurrence хранит настройки повторяемости задачи.
// Заполняются только поля, релевантные для выбранного Type.
type Recurrence struct {
	// Type — обязательный тип повторяемости.
	Type RecurrenceType `json:"type"`

	// Interval — интервал в днях для типа daily (>= 1).
	Interval int `json:"interval,omitempty"`

	// DayOfMonth — число месяца для типа monthly (1–30).
	DayOfMonth int `json:"day_of_month,omitempty"`

	// Dates — список конкретных дат (RFC3339 date, "2006-01-02") для типа dates.
	Dates []string `json:"dates,omitempty"`

	// Parity — чётность для типа even_odd: "even" или "odd".
	Parity RecurrenceParity `json:"parity,omitempty"`
}

type Task struct {
	ID          int64       `json:"id"`
	Title       string      `json:"title"`
	Description string      `json:"description"`
	Status      Status      `json:"status"`
	ScheduledAt *time.Time  `json:"scheduled_at"`
	Recurrence  *Recurrence `json:"recurrence"`
	CreatedAt   time.Time   `json:"created_at"`
	UpdatedAt   time.Time   `json:"updated_at"`
}

func (s Status) Valid() bool {
	switch s {
	case StatusNew, StatusInProgress, StatusDone:
		return true
	default:
		return false
	}
}
