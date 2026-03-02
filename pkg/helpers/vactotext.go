package helpers

import (
	"fmt"
	"strings"

	"github.com/Maltide/jobotparse/pkg/types"
)

func VacancyToText(v types.Vacancy) string {
	var b strings.Builder

	b.WriteString("Позиция: ")
	b.WriteString(v.Profession)
	b.WriteString("\n")

	b.WriteString("Компания: ")
	b.WriteString(v.FirmName)
	b.WriteString("\n")

	b.WriteString("Город: ")
	b.WriteString(v.TownName)
	b.WriteString("\n")

	if v.TypeOfWorkTitle != "" {
		b.WriteString("Тип занятости: ")
		b.WriteString(v.TypeOfWorkTitle)
		b.WriteString("\n")
	}
	if v.ExperienceTitle != "" {
		b.WriteString("Опыт: ")
		b.WriteString(v.ExperienceTitle)
		b.WriteString("\n")
	}
	if v.PaymentFrom != 0 || v.PaymentTo != 0 {
		b.WriteString("Зарплата: ")
		if v.PaymentFrom != 0 {
			b.WriteString(fmt.Sprintf("от %d ", v.PaymentFrom))
		}
		if v.PaymentTo != 0 {
			b.WriteString(fmt.Sprintf("до %d ", v.PaymentTo))
		}
		if v.Currency != "" {
			b.WriteString(v.Currency)
		}
		b.WriteString("\n")
	}
	if v.Work != "" {
		b.WriteString("Обязанности: \n")
		b.WriteString(strings.TrimSpace(v.Work))
		b.WriteString("\n")
	}
	if v.Compensation != "" {
		b.WriteString("Условия работы: \n")
		b.WriteString(strings.TrimSpace(v.Compensation))
		b.WriteString("\n")
	}
	return strings.TrimSpace(b.String())
}
