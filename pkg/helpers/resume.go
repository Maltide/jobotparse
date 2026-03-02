package helpers

import (
	"strings"

	"github.com/Maltide/jobotparse/pkg/types"
)

// BuildResumeText constructs a compact, human-readable resume text
// from the structured payload. It does not validate emptiness; caller
// (frontend) must ensure required fields are present.
func BuildResumeText(in types.Resume) string {
	// Build in a stable order to help the model keep section ordering.
	lines := []string{}

	lines = append(lines, "ФИО: "+strings.TrimSpace(in.FullName))
	if strings.TrimSpace(in.DOB) != "" {
		lines = append(lines, "Дата рождения: "+strings.TrimSpace(in.DOB))
	}
	lines = append(lines, "Контакты: "+strings.TrimSpace(in.Contact))
	lines = append(lines, "Проживает: "+strings.TrimSpace(in.Location))
	lines = append(lines, "Гражданство: "+strings.TrimSpace(in.Citizenship))
	lines = append(lines, "Переезд: "+strings.TrimSpace(in.Relocate))
	lines = append(lines, "Командировки: "+strings.TrimSpace(in.Travel))
	lines = append(lines, "Желаемая должность: "+strings.TrimSpace(in.Position))
	if strings.TrimSpace(in.DesiredSalary) != "" {
		lines = append(lines, "Желаемая зарплата: "+strings.TrimSpace(in.DesiredSalary))
	}
	lines = append(lines, "Тип занятости: "+strings.TrimSpace(in.JobType))

	formats := make([]string, 0, len(in.WorkFormats))
	for _, f := range in.WorkFormats {
		formats = append(formats, f)
	}
	lines = append(lines, "Формат работы: "+strings.Join(formats, ", "))

	// Work entries: simple concatenation per entry
	for _, w := range in.Work {
		company := strings.TrimSpace(w.Company)
		position := strings.TrimSpace(w.Position)
		from := strings.TrimSpace(w.From)
		to := strings.TrimSpace(w.To)
		desc := strings.TrimSpace(w.Description)

		entry := "Компания: " + company + "; "
		entry += "Должность: " + position + "; "
		entry += "Период: " + from + " - " + to + "; "
		entry += "Описание: " + desc
		lines = append(lines, entry)
	}

	// Education entries
	for _, e := range in.Education {
		institution := strings.TrimSpace(e.Institution)
		degree := strings.TrimSpace(e.Degree)
		from := strings.TrimSpace(e.StartDate)
		to := strings.TrimSpace(e.EndDate)
		loc := strings.TrimSpace(e.Location)

		if institution == "" && degree == "" && from == "" && to == "" && loc == "" {
			continue
		}

		entry := "Образование: " + institution
		if degree != "" {
			entry += "; " + degree
		}
		if from != "" || to != "" {
			entry += "; Период: " + from + " - " + to
		}
		if loc != "" {
			entry += "; Локация: " + loc
		}
		lines = append(lines, entry)
	}

	languages := make([]string, 0, len(in.Languages))
	for _, l := range in.Languages {
		l = strings.TrimSpace(l)
		if l != "" {
			languages = append(languages, l)
		}
	}
	lines = append(lines, "Знание языков: "+strings.Join(languages, ", "))

	skills := make([]string, 0, len(in.Skills))
	for _, s := range in.Skills {
		s = strings.TrimSpace(s)
		if s != "" {
			skills = append(skills, s)
		}
	}
	lines = append(lines, "Навыки: "+strings.Join(skills, ", "))

	// Summary: simple label + trimmed content.
	lines = append(lines, "Обо мне: "+strings.TrimSpace(in.Summary))

	return strings.TrimSpace(strings.Join(lines, "\n"))
}
