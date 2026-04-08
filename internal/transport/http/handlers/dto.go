package handlers

import (
	"time"

	taskdomain "example.com/taskservice/internal/domain/task"
)

// recurrenceDTO — настройки повторяемости в HTTP-запросах и ответах.
type recurrenceDTO struct {
	Type       taskdomain.RecurrenceType   `json:"type"`
	Interval   int                         `json:"interval,omitempty"`
	DayOfMonth int                         `json:"day_of_month,omitempty"`
	Dates      []string                    `json:"dates,omitempty"`
	Parity     taskdomain.RecurrenceParity `json:"parity,omitempty"`
}

func recurrenceDTOFromDomain(rec *taskdomain.Recurrence) *recurrenceDTO {
	if rec == nil {
		return nil
	}
	return &recurrenceDTO{
		Type:       rec.Type,
		Interval:   rec.Interval,
		DayOfMonth: rec.DayOfMonth,
		Dates:      rec.Dates,
		Parity:     rec.Parity,
	}
}

func recurrenceDTOToDomain(dto *recurrenceDTO) *taskdomain.Recurrence {
	if dto == nil {
		return nil
	}
	return &taskdomain.Recurrence{
		Type:       dto.Type,
		Interval:   dto.Interval,
		DayOfMonth: dto.DayOfMonth,
		Dates:      dto.Dates,
		Parity:     dto.Parity,
	}
}

type taskMutationDTO struct {
	Title       string            `json:"title"`
	Description string            `json:"description"`
	Status      taskdomain.Status `json:"status"`
	ScheduledAt *time.Time        `json:"scheduled_at"`
	Recurrence  *recurrenceDTO    `json:"recurrence"`
}

type taskDTO struct {
	ID          int64             `json:"id"`
	Title       string            `json:"title"`
	Description string            `json:"description"`
	Status      taskdomain.Status `json:"status"`
	ScheduledAt *time.Time        `json:"scheduled_at"`
	Recurrence  *recurrenceDTO    `json:"recurrence"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
}

func newTaskDTO(task *taskdomain.Task) taskDTO {
	return taskDTO{
		ID:          task.ID,
		Title:       task.Title,
		Description: task.Description,
		Status:      task.Status,
		ScheduledAt: task.ScheduledAt,
		Recurrence:  recurrenceDTOFromDomain(task.Recurrence),
		CreatedAt:   task.CreatedAt,
		UpdatedAt:   task.UpdatedAt,
	}
}
