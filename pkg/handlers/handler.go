package handlers

import (
	"bytes"
	"context"
	"io"
	"math"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/Maltide/jobotparse/pkg/helpers"
	"github.com/Maltide/jobotparse/pkg/interfaces"
	"github.com/Maltide/jobotparse/pkg/types"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func extractLatexBody(input string) string {
	trimmed := strings.TrimSpace(input)
	body := trimmed
	beginMarker := `\begin{document}`
	endMarker := `\end{document}`
	beginIdx := strings.Index(body, beginMarker)
	endIdx := strings.LastIndex(body, endMarker)
	if beginIdx != -1 && endIdx != -1 && endIdx > beginIdx {
		body = strings.TrimSpace(body[beginIdx+len(beginMarker) : endIdx])
	}
	return strings.TrimSpace(body)
}

func wrapLatexDocumentForPDFLaTeX(body string) string {
	return "\\documentclass[11pt,a4paper]{article}\n" +
		"\\usepackage[utf8]{inputenc}\n" +
		"\\usepackage[T2A]{fontenc}\n" +
		"\\usepackage[russian]{babel}\n" +
		"\\usepackage[margin=18mm]{geometry}\n" +
		"\\usepackage{xcolor}\n" +
		"\\usepackage{tabularx}\n" +
		"\\usepackage{array}\n" +
		"\\usepackage[protrusion=true,expansion=false]{microtype}\n" +
		"\\renewcommand{\\familydefault}{\\sfdefault}\n" +
		"\\pagestyle{empty}\n" +
		"\\raggedbottom\n" +
		"\\setlength{\\parindent}{0pt}\n" +
		"\\setlength{\\parskip}{3pt}\n" +
		"\\setlength{\\leftmargini}{16pt}\n" +
		"\\setlength{\\fboxsep}{2.5pt}\n" +
		"\\renewcommand{\\labelitemi}{--}\n" +
		"\\definecolor{cvMuted}{RGB}{115,115,115}\n" +
		"\\definecolor{cvRule}{RGB}{210,210,210}\n" +
		"\\definecolor{cvTagBg}{RGB}{235,235,235}\n" +
		"\\definecolor{graysection}{RGB}{115,115,115}\n" +
		"\\newcommand{\\cvdivider}{{\\color{cvRule}\\rule{\\linewidth}{0.4pt}}\\par}\n" +
		"\\newcommand{\\cvblock}[2]{\\begin{tabularx}{\\textwidth}{@{}p{3.3cm}X@{}}{\\color{cvMuted}\\small #1} & {#2}\\\\\\end{tabularx}\\par}\n" +
		"\\newcommand{\\cvname}[1]{{\\fontsize{26}{30}\\selectfont\\textbf{#1}}\\par}\n" +
		"\\newcommand{\\cvmeta}[1]{{\\color{cvMuted}\\normalsize #1}\\par}\n" +
		"\\newcommand{\\cvsection}[1]{\\vspace{10pt}{\\color{cvMuted}\\small #1}\\par{\\color{cvRule}\\rule{\\linewidth}{0.4pt}}\\vspace{4pt}}\n" +
		"\\newcommand{\\skilltag}[1]{\\colorbox{cvTagBg}{\\strut\\scriptsize #1}\\hspace{4pt}}\n" +
		"\\newenvironment{cvbullets}{\\begin{itemize}\\setlength{\\itemsep}{2pt}\\setlength{\\topsep}{2pt}\\setlength{\\parskip}{0pt}}{\\end{itemize}}\n" +
		"\\newcommand{\\IfNotEmpty}[2]{\\if\\relax\\detokenize{#1}\\relax\\else#2\\fi}\n" +
		"\\newcommand{\\cventry}[4]{%\n" +
		"\\begin{tabularx}{\\textwidth}{@{}p{3.3cm}X@{}}\n" +
		"{\\color{cvMuted}\\small #1} & {\\textbf{#2}}\\\\\n" +
		"\\IfNotEmpty{#3}{ & {#3}\\\\}\n" +
		"\\IfNotEmpty{#4}{ & {\\color{cvMuted}\\small #4}\\\\}\n" +
		"\\end{tabularx}\\par}\n" +
		"\\begin{document}\n" + body + "\n\\end{document}\n"
}

func sanitizeLatexBodyForPDFLaTeX(input string, telegramHandle string) string {
	body := extractLatexBody(input)

	// Частый источник падений pdfLaTeX: несбалансированные фигурные скобки
	// (например, модель недозакрыла аргумент макроса).
	// Мы пытаемся мягко восстановить баланс: лишние закрывающие удаляем,
	// недостающие закрывающие добавляем в конец.
	balanceLatexBraces := func(s string) string {
		var b strings.Builder
		b.Grow(len(s) + 8)
		depth := 0
		for i := 0; i < len(s); i++ {
			// Уважаем литералы \{ и \}.
			if s[i] == '\\' && i+1 < len(s) && (s[i+1] == '{' || s[i+1] == '}') {
				b.WriteByte(s[i])
				b.WriteByte(s[i+1])
				i++
				continue
			}
			switch s[i] {
			case '{':
				depth++
				b.WriteByte('{')
			case '}':
				if depth == 0 {
					// лишняя закрывающая скобка
					continue
				}
				depth--
				b.WriteByte('}')
			default:
				b.WriteByte(s[i])
			}
		}
		for depth > 0 {
			b.WriteByte('}')
			depth--
		}
		return b.String()
	}

	// Минимальная нормализация для стабильной сборки pdfLaTeX.
	body = strings.ReplaceAll(body, "itemitemize", "itemize")
	var out strings.Builder
	out.Grow(len(body))
	for _, r := range body {
		switch r {
		case '\u2022':
			out.WriteString("- ")
			continue
		case '\u2013':
			out.WriteString("--")
			continue
		case '\u2014':
			out.WriteString("---")
			continue
		case '\u00A0':
			out.WriteByte(' ')
			continue
		case '\uFEFF', '\uFFFD':
			continue
		}

		// ASCII оставляем как есть. Всё "латинское" 0x80..0xFF выкидываем:
		// это чаще всего кракозябры и может ломать pdfLaTeX (пример: \dh в T2A).
		if r <= 0x7F {
			out.WriteRune(r)
			continue
		}
		// Cyrillic (basic + extended)
		if (r >= 0x0400 && r <= 0x04FF) || (r >= 0x0500 && r <= 0x052F) {
			out.WriteRune(r)
			continue
		}
		// всё остальное выкидываем, чтобы pdfLaTeX не падал на Unicode
	}
	body = out.String()
	body = strings.ReplaceAll(body, "\\dh", "d")
	body = strings.ReplaceAll(body, "\\DH", "D")

	// pdflatex часто падает на "\\" в начале строки или на длинные цепочки "\\\\\\".
	// Делаем простую, но эффективную чистку без regex.
	lines := strings.Split(body, "\n")
	clean := make([]string, 0, len(lines))
	for _, ln := range lines {
		line := strings.TrimSpace(ln)
		if line == "" {
			clean = append(clean, "")
			continue
		}

		// Схлопываем повторяющиеся переносы: \\\\ -> \\.
		for strings.Contains(line, "\\\\\\\\") {
			line = strings.ReplaceAll(line, "\\\\\\\\", "\\\\")
		}

		// Убираем переносы в начале строки: "\\\\ ..." -> "...".
		for strings.HasPrefix(line, "\\\\") {
			line = strings.TrimSpace(strings.TrimPrefix(line, "\\\\"))
		}

		// Если строка стала состоять только из переносов — выкидываем.
		onlyBreaks := true
		tmp := line
		for strings.HasPrefix(tmp, "\\\\") {
			tmp = strings.TrimPrefix(tmp, "\\\\")
		}
		if strings.TrimSpace(tmp) != "" {
			onlyBreaks = false
		}
		if onlyBreaks {
			continue
		}

		clean = append(clean, line)
	}
	body = strings.TrimSpace(strings.Join(clean, "\n"))

	// Модель иногда всё равно вставляет запрещённые команды \url / \href.
	// Чтобы не подключать пакеты, просто превращаем их в обычный текст.
	stripBraceArg := func(s string, braceStart int) (content string, next int, ok bool) {
		if braceStart < 0 || braceStart >= len(s) || s[braceStart] != '{' {
			return "", braceStart, false
		}
		depth := 0
		for i := braceStart; i < len(s); i++ {
			if s[i] == '{' {
				depth++
				continue
			}
			if s[i] == '}' {
				depth--
				if depth == 0 {
					return s[braceStart+1 : i], i + 1, true
				}
			}
		}
		return "", len(s), false
	}

	sanitizeHypersetup := func(s string) string {
		var b strings.Builder
		b.Grow(len(s))
		for i := 0; i < len(s); {
			if strings.HasPrefix(s[i:], "\\hypersetup{") {
				_, next, ok := stripBraceArg(s, i+len("\\hypersetup"))
				if ok {
					i = next
					continue
				}
			}
			b.WriteByte(s[i])
			i++
		}
		return b.String()
	}

	// Модель иногда генерирует стандартные LaTeX-команды титула.
	// В нашем шаблоне (server-side preamble) они не нужны и часто ломают сборку:
	// например, \maketitle без \title даёт "No \\title given".
	sanitizeTitleCommands := func(s string) string {
		// Сначала простой маппинг секций на наш макрос.
		s = strings.ReplaceAll(s, "\\section*{", "\\cvsection{")
		s = strings.ReplaceAll(s, "\\section{", "\\cvsection{")
		s = strings.ReplaceAll(s, "\\subsection*{", "\\cvsection{")
		s = strings.ReplaceAll(s, "\\subsection{", "\\cvsection{")

		var b strings.Builder
		b.Grow(len(s))
		for i := 0; i < len(s); {
			switch {
			case strings.HasPrefix(s[i:], "\\maketitle"):
				i += len("\\maketitle")
				continue
			case strings.HasPrefix(s[i:], "\\title{"):
				_, next, ok := stripBraceArg(s, i+len("\\title"))
				if ok {
					i = next
					continue
				}
			case strings.HasPrefix(s[i:], "\\author{"):
				_, next, ok := stripBraceArg(s, i+len("\\author"))
				if ok {
					i = next
					continue
				}
			case strings.HasPrefix(s[i:], "\\date{"):
				_, next, ok := stripBraceArg(s, i+len("\\date"))
				if ok {
					i = next
					continue
				}
			}
			b.WriteByte(s[i])
			i++
		}
		return b.String()
	}

	isPunctOnly := func(s string) bool {
		trim := strings.TrimSpace(s)
		if trim == "" {
			return true
		}
		for _, r := range trim {
			switch r {
			case ',', '.', ';', ':', '-', '—', '–', '(', ')', '[', ']', '{', '}', '/':
				continue
			default:
				return false
			}
		}
		return true
	}

	isTinyLatinOnly := func(s string) bool {
		t := strings.TrimSpace(s)
		if t == "" {
			return false
		}
		if len(t) > 2 {
			return false
		}
		for _, r := range t {
			if r < 'A' || (r > 'Z' && r < 'a') || r > 'z' {
				return false
			}
		}
		return true
	}

	containsSalary := func(s string) bool {
		low := strings.ToLower(s)
		if strings.Contains(low, "зарп") || strings.Contains(low, "оклад") || strings.Contains(low, "вознаграж") {
			return true
		}
		if strings.Contains(low, "руб") || strings.Contains(s, "₽") || strings.Contains(low, "р.") || strings.Contains(low, "rur") || strings.Contains(low, "rub") {
			return true
		}
		return false
	}

	containsTelegramLine := func(s string) bool {
		low := strings.ToLower(s)
		return strings.Contains(low, "связаться со мной в телеграме:")
	}

	sanitizeSkilltagArg := func(arg string) string {
		a := strings.TrimSpace(arg)
		a = strings.ReplaceAll(a, "\r", " ")
		a = strings.ReplaceAll(a, "\n", " ")
		a = strings.ReplaceAll(a, "{", "")
		a = strings.ReplaceAll(a, "}", "")
		a = strings.Join(strings.Fields(a), " ")
		for strings.Contains(a, "()") {
			a = strings.ReplaceAll(a, "()", "")
		}
		for strings.Contains(a, "( )") {
			a = strings.ReplaceAll(a, "( )", "")
		}
		a = strings.Join(strings.Fields(a), " ")
		trimEdge := func(r rune) bool {
			switch r {
			case ',', '.', ';', ':', '-', '—', '–', '(', ')', '[', ']', '{', '}', '/', '\\':
				return true
			default:
				return false
			}
		}
		a = strings.TrimFunc(a, trimEdge)
		a = strings.TrimSpace(a)
		if isPunctOnly(a) {
			return ""
		}
		return a
	}

	sanitizeCventryArg := func(arg string, kind string) string {
		a := strings.TrimSpace(arg)
		a = strings.ReplaceAll(a, "\r", " ")
		a = strings.ReplaceAll(a, "\n", " ")
		a = strings.ReplaceAll(a, "{", "")
		a = strings.ReplaceAll(a, "}", "")
		a = strings.Join(strings.Fields(a), " ")
		for strings.Contains(a, "()") {
			a = strings.ReplaceAll(a, "()", "")
		}
		for strings.Contains(a, "( )") {
			a = strings.ReplaceAll(a, "( )", "")
		}
		a = strings.Join(strings.Fields(a), " ")
		if kind == "date" {
			a = strings.ReplaceAll(a, ",", " ")
			a = strings.Join(strings.Fields(a), " ")
			if isPunctOnly(a) {
				return ""
			}
		}
		if isPunctOnly(a) {
			return ""
		}
		return strings.TrimSpace(a)
	}

	sanitizeCventryCalls := func(s string) string {
		var b strings.Builder
		b.Grow(len(s))
		for i := 0; i < len(s); {
			idx := strings.Index(s[i:], "\\cventry{")
			if idx == -1 {
				b.WriteString(s[i:])
				break
			}
			idx += i
			b.WriteString(s[i:idx])
			j := idx + len("\\cventry")
			arg1, next1, ok1 := stripBraceArg(s, j)
			if !ok1 {
				b.WriteString("\\cventry")
				i = j
				continue
			}
			arg2, next2, ok2 := stripBraceArg(s, next1)
			arg3, next3, ok3 := stripBraceArg(s, next2)
			arg4, next4, ok4 := stripBraceArg(s, next3)
			if !(ok2 && ok3 && ok4) {
				// Модель часто ломает сигнатуру \cventry и даёт меньше 4 аргументов.
				// Оставлять это как есть нельзя: pdfLaTeX упадёт на «missing { }».
				// Поэтому деградируем вызов до обычного текста (первый аргумент).
				plain := sanitizeCventryArg(arg1, "meta")
				if plain != "" {
					b.WriteString(plain)
				}
				i = next1
				continue
			}

			arg1 = sanitizeCventryArg(arg1, "date")
			arg2 = sanitizeCventryArg(arg2, "org")
			arg3 = sanitizeCventryArg(arg3, "role")
			arg4 = sanitizeCventryArg(arg4, "meta")

			b.WriteString("\\cventry{")
			b.WriteString(arg1)
			b.WriteString("}{")
			b.WriteString(arg2)
			b.WriteString("}{")
			b.WriteString(arg3)
			b.WriteString("}{")
			b.WriteString(arg4)
			b.WriteString("}")

			// Ещё один частый артефакт: после корректного \cventry модель добавляет
			// лишние группы вида {..}{..} (как будто продолжает аргументы).
			// Если их не убрать, появляются ошибки вроде "Argument of \\cvsection has an extra }".
			i = next4
			for {
				k := i
				for k < len(s) {
					switch s[k] {
					case ' ', '\t', '\n', '\r':
						k++
						continue
					}
					break
				}
				if k >= len(s) || s[k] != '{' {
					break
				}
				content, next, ok := stripBraceArg(s, k)
				if !ok {
					break
				}
				trim := strings.TrimSpace(content)
				// Удаляем только «похожее на хвост аргументов»: пустое/пунктуация,
				// либо цепочка аргументов (следом снова '{').
				nextNonSpace := next
				for nextNonSpace < len(s) {
					switch s[nextNonSpace] {
					case ' ', '\t', '\n', '\r':
						nextNonSpace++
						continue
					}
					break
				}
				if trim == "" || isPunctOnly(trim) || (nextNonSpace < len(s) && s[nextNonSpace] == '{') {
					i = next
					continue
				}
				break
			}
		}
		return b.String()
	}

	sanitizeSkilltagCalls := func(s string) string {
		var b strings.Builder
		b.Grow(len(s))
		for i := 0; i < len(s); {
			idx := strings.Index(s[i:], "\\skilltag{")
			if idx == -1 {
				b.WriteString(s[i:])
				break
			}
			idx += i
			b.WriteString(s[i:idx])
			j := idx + len("\\skilltag")
			arg, next, ok := stripBraceArg(s, j)
			if !ok {
				b.WriteString("\\skilltag")
				i = j
				continue
			}
			arg = sanitizeSkilltagArg(arg)
			if arg != "" {
				b.WriteString("\\skilltag{")
				b.WriteString(arg)
				b.WriteString("}")
			}
			i = next
		}
		return b.String()
	}

	sanitizeCvmetaArg := func(arg string) string {
		escapeLatexText := func(s string) string {
			var b strings.Builder
			b.Grow(len(s) + 8)
			for i := 0; i < len(s); i++ {
				ch := s[i]
				switch ch {
				case '\\':
					// В мета-строках (контакты/статусы) обратный слэш часто является артефактом
					// и может превратиться в LaTeX-команды. Просто выкидываем.
					continue
				case '{':
					b.WriteString("\\{")
					continue
				case '}':
					b.WriteString("\\}")
					continue
				case '_', '%', '&', '#', '$':
					if i > 0 && s[i-1] == '\\' {
						b.WriteByte(ch)
						continue
					}
					b.WriteByte('\\')
					b.WriteByte(ch)
					continue
				}
				b.WriteByte(ch)
			}
			return b.String()
		}

		a := strings.TrimSpace(arg)
		a = strings.ReplaceAll(a, "\r", " ")
		a = strings.ReplaceAll(a, "\n", " ")
		a = strings.Join(strings.Fields(a), " ")
		for strings.Contains(a, "\\\\\\\\") {
			a = strings.ReplaceAll(a, "\\\\\\\\", "\\\\")
		}
		a = strings.TrimSpace(a)
		a = strings.TrimSpace(strings.TrimSuffix(a, "\\\\"))
		a = strings.TrimSpace(strings.TrimPrefix(a, "\\\\"))
		if isPunctOnly(a) {
			return ""
		}
		return escapeLatexText(a)
	}

	sanitizeCvmetaCalls := func(s string) string {
		var b strings.Builder
		b.Grow(len(s))
		for i := 0; i < len(s); {
			idx := strings.Index(s[i:], "\\cvmeta{")
			if idx == -1 {
				b.WriteString(s[i:])
				break
			}
			idx += i
			b.WriteString(s[i:idx])
			j := idx + len("\\cvmeta")
			arg, next, ok := stripBraceArg(s, j)
			if !ok {
				b.WriteString("\\cvmeta")
				i = j
				continue
			}
			arg = sanitizeCvmetaArg(arg)
			b.WriteString("\\cvmeta{")
			b.WriteString(arg)
			b.WriteString("}")
			i = next
		}
		return b.String()
	}

	sanitizeCvblockLeftArg := func(arg string) string {
		// Левая колонка — это «графа», ожидаем короткий текст без LaTeX-команд.
		a := strings.TrimSpace(arg)
		a = strings.ReplaceAll(a, "\r", " ")
		a = strings.ReplaceAll(a, "\n", " ")
		a = strings.Join(strings.Fields(a), " ")
		if isPunctOnly(a) {
			return ""
		}
		// Используем ту же экранировку, что и для cvmeta.
		return sanitizeCvmetaArg(a)
	}

	sanitizeCvblockCalls := func(s string) string {
		var b strings.Builder
		b.Grow(len(s))
		for i := 0; i < len(s); {
			idx := strings.Index(s[i:], "\\cvblock{")
			if idx == -1 {
				b.WriteString(s[i:])
				break
			}
			idx += i
			b.WriteString(s[i:idx])
			j := idx + len("\\cvblock")
			arg1, next1, ok1 := stripBraceArg(s, j)
			arg2, next2, ok2 := stripBraceArg(s, next1)
			if !(ok1 && ok2) {
				// Если cvblock сломан (нет закрывающей скобки), проще выкинуть этот фрагмент,
				// иначе pdfLaTeX упадёт на несбалансированных скобках.
				nl := strings.IndexByte(s[idx:], '\n')
				if nl == -1 {
					break
				}
				i = idx + nl + 1
				continue
			}

			arg1 = sanitizeCvblockLeftArg(arg1)
			arg2 = strings.TrimSpace(arg2)
			if arg1 != "" {
				b.WriteString("\\cvblock{")
				b.WriteString(arg1)
				b.WriteString("}{")
				b.WriteString(arg2)
				b.WriteString("}")
			}
			i = next2
		}
		return b.String()
	}

	sanitizeLinks := func(s string) string {
		var b strings.Builder
		b.Grow(len(s))
		for i := 0; i < len(s); {
			if strings.HasPrefix(s[i:], "\\url{") {
				content, next, ok := stripBraceArg(s, i+4)
				if ok {
					b.WriteString(content)
					i = next
					continue
				}
			}
			if strings.HasPrefix(s[i:], "\\href{") {
				_, next1, ok1 := stripBraceArg(s, i+5)
				if ok1 && next1 < len(s) && s[next1] == '{' {
					text, next2, ok2 := stripBraceArg(s, next1)
					if ok2 {
						b.WriteString(text)
						i = next2
						continue
					}
				}
			}
			b.WriteByte(s[i])
			i++
		}
		return b.String()
	}

	body = sanitizeHypersetup(body)
	body = sanitizeTitleCommands(body)
	body = sanitizeLinks(body)
	body = sanitizeCventryCalls(body)
	body = sanitizeSkilltagCalls(body)
	body = sanitizeCvmetaCalls(body)
	body = sanitizeCvblockCalls(body)

	// Модель иногда вставляет "\\item" вне окружений itemize/cvbullets.
	sanitizeItemsOutsideLists := func(s string) string {
		lines := strings.Split(s, "\n")
		out := make([]string, 0, len(lines))
		listDepth := 0
		for _, ln := range lines {
			trim := strings.TrimSpace(ln)
			if strings.Contains(trim, "\\begin{itemize}") || strings.Contains(trim, "\\begin{enumerate}") || strings.Contains(trim, "\\begin{cvbullets}") {
				listDepth++
				out = append(out, ln)
				continue
			}
			if strings.Contains(trim, "\\end{itemize}") || strings.Contains(trim, "\\end{enumerate}") || strings.Contains(trim, "\\end{cvbullets}") {
				if listDepth > 0 {
					listDepth--
				}
				out = append(out, ln)
				continue
			}
			if listDepth == 0 {
				if strings.HasPrefix(trim, "\\item") {
					rest := strings.TrimSpace(strings.TrimPrefix(trim, "\\item"))
					if rest == "" {
						continue
					}
					out = append(out, rest)
					continue
				}
			}
			out = append(out, ln)
		}
		return strings.Join(out, "\n")
	}
	body = sanitizeItemsOutsideLists(body)

	// Запрещаем указание зарплаты: выкидываем строки, которые выглядят как "зарплата ...".
	// Важно: не трогаем строки с фигурными скобками, чтобы не ломать баланс.
	if body != "" {
		lines3 := strings.Split(body, "\n")
		clean3 := make([]string, 0, len(lines3))
		for _, ln := range lines3 {
			trim := strings.TrimSpace(ln)
			if trim == "" {
				clean3 = append(clean3, "")
				continue
			}
			if !strings.ContainsAny(trim, "{}") && containsSalary(trim) {
				continue
			}
			clean3 = append(clean3, ln)
		}
		body = strings.TrimSpace(strings.Join(clean3, "\n"))
	}

	// Страховка: если телеграм-строки нет, добавим её в шапку.
	telegramHandle = strings.TrimSpace(telegramHandle)
	if telegramHandle != "" && !strings.HasPrefix(telegramHandle, "@") {
		telegramHandle = "@" + telegramHandle
	}
	if telegramHandle != "" && !containsTelegramLine(body) {
		telegramSafe := telegramHandle
		// В строку контактов не должны попадать backslash-артефакты: они легко превращаются в команды LaTeX.
		telegramSafe = strings.ReplaceAll(telegramSafe, "\\", "")
		telegramSafe = strings.ReplaceAll(telegramSafe, "_", "\\_")
		telegramSafe = strings.ReplaceAll(telegramSafe, "%", "\\%")
		telegramSafe = strings.ReplaceAll(telegramSafe, "&", "\\&")
		telegramSafe = strings.ReplaceAll(telegramSafe, "#", "\\#")
		telegramSafe = strings.ReplaceAll(telegramSafe, "$", "\\$")
		insert := "\\cvmeta{связаться со мной в телеграме: " + telegramSafe + "}"
		if idx := strings.Index(body, "\\cvname{"); idx != -1 {
			start := idx + len("\\cvname")
			_, next, ok := stripBraceArg(body, start)
			if ok {
				post := next
				if post < len(body) && body[post] == '\n' {
					post++
				}
				body = body[:post] + insert + "\n" + body[post:]
			} else {
				body = insert + "\n" + body
			}
		} else {
			body = insert + "\n" + body
		}
	}

	// Убираем "мусорные" строки вида ", ," / "()" / одиночные латинские буквы.
	lines2 := strings.Split(body, "\n")
	clean2 := make([]string, 0, len(lines2))
	for _, ln := range lines2 {
		line := strings.TrimSpace(ln)
		if line == "" {
			clean2 = append(clean2, "")
			continue
		}
		if !strings.ContainsAny(line, "{}") {
			if isPunctOnly(line) || isTinyLatinOnly(line) {
				continue
			}
		}
		if !strings.ContainsAny(line, "{}") {
			if isPunctOnly(strings.ReplaceAll(line, "\\\\", "")) {
				continue
			}
		}
		clean2 = append(clean2, ln)
	}
	body = strings.TrimSpace(strings.Join(clean2, "\n"))

	repl := strings.NewReplacer(
		"Microservices architecture", "Микросервисная архитектура",
		"microservices architecture", "Микросервисная архитектура",
		"Database indexing and optimization", "Индексация и оптимизация БД",
		"database indexing and optimization", "Индексация и оптимизация БД",
		"Backend system design", "Проектирование backend-систем",
		"backend system design", "Проектирование backend-систем",
		"Docker container orchestration", "Оркестрация контейнеров Docker",
		"docker container orchestration", "Оркестрация контейнеров Docker",
		"HTTP API development", "Разработка HTTP API",
		"Unit testing", "Юнит-тестирование",
		"CRUD operations", "CRUD-операции",
	)
	body = repl.Replace(body)

	body = strings.ReplaceAll(body, "\\\\begin{center}", "")
	body = strings.ReplaceAll(body, "\\\\end{center}", "")
	body = strings.ReplaceAll(body, "\\\\centering", "")

	body = strings.TrimSpace(body)
	body = balanceLatexBraces(body)
	return strings.TrimSpace(body)
}

func normalizeLatexForPDFLaTeX(input string, telegramHandle string) string {
	body := sanitizeLatexBodyForPDFLaTeX(input, telegramHandle)
	return wrapLatexDocumentForPDFLaTeX(body)
}

func extractTelegramHandle(s string) string {
	// 1) t.me/<user>
	reTme := regexp.MustCompile(`(?i)t\.me/([a-z0-9_]{4,32})`)
	if m := reTme.FindStringSubmatch(s); len(m) == 2 {
		return "@" + m[1]
	}
	// 2) @username (но избегаем email-частей вроде name@domain)
	reAt := regexp.MustCompile(`(?m)(^|\s)@([a-z0-9_]{4,32})\b`)
	if m := reAt.FindStringSubmatch(s); len(m) == 3 {
		return "@" + m[2]
	}
	return ""
}

func extractEmailLocalPart(s string) string {
	re := regexp.MustCompile(`(?i)\b([a-z0-9._%+-]{2,64})@([a-z0-9.-]+\.[a-z]{2,})\b`)
	if m := re.FindStringSubmatch(s); len(m) == 3 {
		local := m[1]
		local = strings.Trim(local, "._%+-")
		if local == "" {
			return ""
		}
		return local
	}
	return ""
}

func telegramFromEmailLocalPart(local string) string {
	// Telegram username: 4..32 chars, latin letters/digits/underscore.
	// Email local-part часто содержит '.', '+', '-', которые в Telegram недопустимы.
	s := strings.ToLower(strings.TrimSpace(local))
	if s == "" {
		return ""
	}
	// Типовые разделители превращаем в underscore.
	s = strings.NewReplacer(".", "_", "-", "_", "+", "_").Replace(s)
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' {
			b.WriteRune(r)
		}
	}
	out := b.String()
	for strings.Contains(out, "__") {
		out = strings.ReplaceAll(out, "__", "_")
	}
	out = strings.Trim(out, "_")
	if len(out) > 32 {
		out = out[:32]
	}
	if len(out) < 4 {
		return ""
	}
	return out
}

func extractVacancySkillHints(vac types.VacancySJ) []string {
	src := strings.ToLower(strings.Join([]string{vac.Profession, vac.Work, vac.Compensation, vac.ExperienceTitle, vac.TypeOfWorkTitle}, "\n"))
	// Небольшой словарь тех-ключей: цель — подсказка модели, а не идеальный NER.
	keywords := []struct {
		needle string
		label  string
	}{
		{" golang ", "Go"},
		{" go ", "Go"},
		{"golang 1.22", "Go 1.22"},
		{"go 1.22", "Go 1.22"},
		{"postgres", "PostgreSQL"},
		{"postgresql", "PostgreSQL"},
		{"sql", "SQL"},
		{"gorm", "GORM"},
		{"docker", "Docker"},
		{"kubernetes", "Kubernetes"},
		{"k8s", "Kubernetes"},
		{"git", "Git"},
		{"gitlab", "GitLab"},
		{"teamcity", "TeamCity"},
		{"linux", "Linux"},
		{"rest", "REST"},
		{"http", "HTTP"},
		{"grpc", "gRPC"},
		{"json", "JSON"},
		{"redis", "Redis"},
		{"kafka", "Kafka"},
		{"rabbit", "RabbitMQ"},
		{"очеред", "Очереди сообщений"},
		{"async", "Асинхронное взаимодействие"},
		{"асинхрон", "Асинхронное взаимодействие"},
		{"ci/cd", "CI/CD"},
		{"ci", "CI/CD"},
		{"cd", "CI/CD"},
		{"prometheus", "Prometheus"},
		{"grafana", "Grafana"},
		{"sentry", "Sentry"},
		{"temporal", "Temporal"},
		{"тест", "Тестирование"},
		{"unit", "Юнит-тестирование"},
		{"микросервис", "Микросервисы"},
		{"микро сервис", "Микросервисы"},
		{"микро-сервис", "Микросервисы"},
	}
	found := make([]string, 0, 12)
	seen := make(map[string]struct{}, 32)
	pad := " " + src + " "
	for _, kw := range keywords {
		if _, ok := seen[kw.label]; ok {
			continue
		}
		if strings.Contains(pad, kw.needle) || strings.Contains(src, strings.TrimSpace(kw.needle)) {
			seen[kw.label] = struct{}{}
			found = append(found, kw.label)
		}
	}
	return found
}

// Authorize serves the /auth endpoint.
// GET returns an HTML login form; POST validates admin and redirects the client to SuperJob OAuth authorization.
func Authorize(w http.ResponseWriter, r *http.Request, log *zap.SugaredLogger) error {
	if r.Method == http.MethodGet {
		http.ServeFile(w, r, "./static/auth.html")
		return nil
	}

	if r.Method == http.MethodPost {
		// Validate admin fields from env before redirecting to SuperJob OAuth.
		if err := r.ParseForm(); err != nil {
			log.Errorf("handlers: error parsing form: %v", err)
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return err
		}
		user := r.PostFormValue("username")
		pass := r.PostFormValue("password")

		if user == "" || pass == "" {
			log.Errorf("handlers: username or password is empty")
			http.Error(w, "Username and password are required", http.StatusBadRequest)
			return nil
		}
		if user != os.Getenv("ADMIN_USER") || pass != os.Getenv("ADMIN_PASS") {
			log.Errorf("handlers: invalid username or password")
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return nil
		}
	} else {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return nil
	}

	// Not BeforeRequest func because it uses RefreshFunc like if tokens already expired 100%, but we need to check tokens are still valid
	tokensinfo, rerr := helpers.ReadTokens(log)
	if rerr == nil {
		ok, ierr := helpers.IsValidToken(&tokensinfo, log)
		if ierr == nil && ok {
			// If a valid token already exists, avoid forcing OAuth.
			log.Infof("handlers: tokens are present, no need to authorize")
			return nil
		}
	}

	url, err := helpers.Authstr()
	if err != nil {
		log.Errorf("handlers: error getting auth URL: %v", err)
		return err
	}

	http.Redirect(w, r, url, http.StatusFound)

	return nil
}

// AllVacancies fetches vacancies from all configured providers and merges results.
func AllVacancies(apis []interfaces.VacanciesProvider, filters types.Filters, log *zap.SugaredLogger) (types.VacanciesResponse, error) {
	var allVacs types.VacanciesResponse

	for _, api := range apis {
		vacs, err := api.Fetch(filters, log)
		if err != nil {
			log.Errorf("handlers: error fetching vacancies from API: %v", err)
			continue
		}
		allVacs.Objects = append(allVacs.Objects, vacs.Objects...)
	}

	log.Infof("handlers: total vacancies fetched from all APIs: %d", len(allVacs.Objects))

	return allVacs, nil
}

func AdaptResume(w http.ResponseWriter, r *http.Request, db *gorm.DB, log *zap.SugaredLogger) error {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return nil
	}

	// УБРАЛИ раннюю проверку только query:
	// vacancy_url может прийти из multipart/form-data как поле формы.

	refinePasses := 3
	if v := strings.TrimSpace(os.Getenv("ADAPT_REFINE_PASSES")); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			refinePasses = n
		}
	}
	if refinePasses < 0 {
		refinePasses = 0
	}
	if refinePasses > 5 {
		refinePasses = 5
	}

	// общий таймаут на всю операцию (PDF->text + LLM + LaTeX->PDF)
	// Refinement-проходы дают больше качества, но требуют больше времени.
	timeout := 2*time.Minute + time.Duration(refinePasses)*time.Minute
	if timeout > 7*time.Minute {
		timeout = 7 * time.Minute
	}
	ctx, cancel := context.WithTimeout(r.Context(), timeout)
	defer cancel()

	if err := r.ParseMultipartForm(10 << 20); err != nil { // 10 MB
		log.Errorf("handlers: AdaptResume: error parsing multipart form: %v", err)
		http.Error(w, "file is too large(10 MB max)", http.StatusBadRequest)
		return nil
	}

	file, header, err := r.FormFile("resume_pdf")
	if err != nil {
		log.Errorf("handlers: AdaptResume: error retrieving the file: %v", err)
		http.Error(w, "Invalid file upload", http.StatusBadRequest)
		return nil
	}
	defer file.Close()

	if header == nil || header.Size == 0 {
		log.Error("handlers: AdaptResume: empty file uploaded")
		http.Error(w, "file is empty", http.StatusBadRequest)
		return nil
	}

	// Явная валидация vacancy_url после ParseMultipartForm:
	vacURL := strings.TrimSpace(r.URL.Query().Get("vacancy_url"))
	if vacURL == "" {
		vacURL = strings.TrimSpace(r.FormValue("vacancy_url"))
	}
	if vacURL == "" {
		log.Error("handlers: AdaptResume: empty vacancy_url")
		http.Error(w, "vacancy_url parameter is required", http.StatusBadRequest)
		return nil
	}

	// 1) вакансия (у вас уже сделано)
	vac, err := helpers.CheckVacInDB(r, db, log)
	if err != nil {
		log.Errorf("handlers: AdaptResume: error checking vacancy in DB: %v", err)
		http.Error(w, "vacancy not found", http.StatusNotFound)
		return nil
	}

	// 2) сохраняем PDF во временную папку
	tmpDir, err := os.MkdirTemp("", "jobotparse-adapt-*")
	if err != nil {
		log.Errorf("handlers: AdaptResume: MkdirTemp error: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return nil
	}
	defer os.RemoveAll(tmpDir)

	pdfPath := filepath.Join(tmpDir, "resume.pdf")
	outPDF, err := os.Create(pdfPath)
	if err != nil {
		log.Errorf("handlers: AdaptResume: create temp pdf error: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return nil
	}
	if _, err := io.Copy(outPDF, file); err != nil {
		outPDF.Close()
		log.Errorf("handlers: AdaptResume: saving pdf error: %v", err)
		http.Error(w, "Bad resume file", http.StatusBadRequest)
		return nil
	}
	outPDF.Close()

	// 3) PDF -> text (вынесли в helpers)
	resumeText, err := helpers.PDFToText(ctx, pdfPath, log)
	if err != nil {
		http.Error(w, "Failed to read PDF text (pdftotext)", http.StatusInternalServerError)
		return nil
	}

	tele := extractTelegramHandle(resumeText)
	if tele == "" {
		if local := extractEmailLocalPart(resumeText); local != "" {
			if tg := telegramFromEmailLocalPart(local); tg != "" {
				tele = "@" + tg
			}
		}
	}
	if tele == "" {
		tele = "@candidate"
	}
	vacSkills := extractVacancySkillHints(vac)
	minCoverage := 0
	if len(vacSkills) > 0 {
		minCoverage = int(math.Ceil(0.75 * float64(len(vacSkills))))
	}

	// 4) заполняем шаблон вакансии (aireq.txt)
	aireq, err := template.ParseFiles("static/aireq.txt")
	if err != nil {
		log.Errorf("handlers: AdaptResume: error parsing AI request template: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return nil
	}
	data := struct {
		Vac            types.VacancySJ
		VacancySkills  []string
		MinSkillsCover int
		TelegramHandle string
	}{
		Vac:            vac,
		VacancySkills:  vacSkills,
		MinSkillsCover: minCoverage,
		TelegramHandle: tele,
	}

	var vacBuf bytes.Buffer
	if err := aireq.Execute(&vacBuf, data); err != nil {
		log.Errorf("handlers: AdaptResume: error executing AI request template: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return nil
	}

	// 5) prompt
	fullPrompt := vacBuf.String() +
		"\n\n[ТЕКСТ РЕЗЮМЕ КАНДИДАТА]\n" + resumeText + "\n"

	// 6) Ollama (вынесли в helpers)
	latex, err := helpers.OllamaGenerate(ctx, fullPrompt, log)
	if err != nil {
		log.Errorf("handlers: AdaptResume: ollama error: %v", err)
		http.Error(w, "Ollama is not available", http.StatusBadGateway)
		return nil
	}

	// Держим LaTeX как "тело" документа: так удобнее делать refinement passes.
	body := sanitizeLatexBodyForPDFLaTeX(latex, tele)

	if refinePasses > 0 {
		refineTpl, err := template.ParseFiles("static/aireq_refine.txt")
		if err != nil {
			log.Errorf("handlers: AdaptResume: error parsing refine template: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return nil
		}

		for pass := 0; pass < refinePasses; pass++ {
			var refineBuf bytes.Buffer
			refineData := struct {
				Vac            types.VacancySJ
				VacancySkills  []string
				MinSkillsCover int
				TelegramHandle string
				CurrentLatex   string
			}{
				Vac:            vac,
				VacancySkills:  vacSkills,
				MinSkillsCover: minCoverage,
				TelegramHandle: tele,
				CurrentLatex:   body,
			}
			if err := refineTpl.Execute(&refineBuf, refineData); err != nil {
				log.Errorf("handlers: AdaptResume: error executing refine template: %v", err)
				http.Error(w, "Internal server error", http.StatusInternalServerError)
				return nil
			}

			log.Infof("handlers: AdaptResume: refinement pass %d/%d", pass+1, refinePasses)
			refined, err := helpers.OllamaGenerate(ctx, refineBuf.String(), log)
			if err != nil {
				log.Errorf("handlers: AdaptResume: ollama refine error: %v", err)
				http.Error(w, "Ollama is not available", http.StatusBadGateway)
				return nil
			}

			body = sanitizeLatexBodyForPDFLaTeX(refined, tele)
		}
	}

	// Всегда приводим LaTeX к pdfLaTeX-совместимому варианту (кириллица).
	latex = wrapLatexDocumentForPDFLaTeX(body)

	// 7) LaTeX -> PDF
	// По умолчанию пытаемся tectonic (если доступен), иначе pdflatex.
	// Можно переопределить бинарник через LATEX_BIN или (для обратной совместимости) TECTONIC_BIN.
	latexBin := strings.TrimSpace(os.Getenv("LATEX_BIN"))
	if latexBin == "" {
		latexBin = strings.TrimSpace(os.Getenv("TECTONIC_BIN"))
	}
	if latexBin == "" {
		if _, err := exec.LookPath("tectonic"); err == nil {
			latexBin = "tectonic"
		} else {
			latexBin = "pdflatex"
		}
	}

	texPath := filepath.Join(tmpDir, "adapted_resume.tex")
	if err := os.WriteFile(texPath, []byte(latex), 0644); err != nil {
		log.Errorf("handlers: AdaptResume: write tex error: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return nil
	}

	var compileCmd *exec.Cmd
	if filepath.Base(latexBin) == "tectonic" {
		compileCmd = exec.CommandContext(ctx, latexBin, "-X", "compile", "--outdir", tmpDir, texPath)
	} else {
		compileCmd = exec.CommandContext(ctx, latexBin, "-interaction=nonstopmode", "-halt-on-error", "-output-directory", tmpDir, texPath)
	}
	if out, err := compileCmd.CombinedOutput(); err != nil {
		outStr := string(out)
		// Пытаемся дополнительно показать контекст из .tex вокруг строки ошибки (l.<n>).
		lineRe := regexp.MustCompile(`(?m)^l\.(\d+)\b`)
		if m := lineRe.FindStringSubmatch(outStr); len(m) == 2 {
			if n, convErr := strconv.Atoi(m[1]); convErr == nil && n > 0 {
				if texBytes, readErr := os.ReadFile(texPath); readErr == nil {
					texLines := strings.Split(string(texBytes), "\n")
					from := n - 3
					if from < 1 {
						from = 1
					}
					to := n + 3
					if to > len(texLines) {
						to = len(texLines)
					}
					var snip strings.Builder
					for i := from; i <= to; i++ {
						snip.WriteString(strconv.Itoa(i))
						snip.WriteString(": ")
						snip.WriteString(texLines[i-1])
						snip.WriteByte('\n')
					}
					log.Errorf("handlers: AdaptResume: latex compiler error context (tex around line %d):\n%s", n, snip.String())
				}
			}
		}

		// Если есть .log, выведем хвост: он иногда точнее, чем stdout.
		logPath := filepath.Join(tmpDir, "adapted_resume.log")
		if lb, readErr := os.ReadFile(logPath); readErr == nil {
			logLines := strings.Split(string(lb), "\n")
			start := len(logLines) - 80
			if start < 0 {
				start = 0
			}
			log.Errorf("handlers: AdaptResume: latex compiler error log tail:\n%s", strings.Join(logLines[start:], "\n"))
		}

		log.Errorf("handlers: AdaptResume: latex compiler error: %v, out: %s", err, outStr)
		http.Error(w, "Failed to compile LaTeX to PDF", http.StatusInternalServerError)
		return nil
	}

	pdfOutPath := filepath.Join(tmpDir, "adapted_resume.pdf")
	pdfBytes, err := os.ReadFile(pdfOutPath)
	if err != nil {
		log.Errorf("handlers: AdaptResume: read compiled pdf error: %v", err)
		http.Error(w, "Failed to read generated PDF", http.StatusInternalServerError)
		return nil
	}

	// 8) отдаём PDF пользователю
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", `attachment; filename="adapted_resume.pdf"`)
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write(pdfBytes); err != nil {
		log.Errorf("handlers: AdaptResume: write response error: %v", err)
		return err
	}

	return nil
}

// (остальные функции без изменений)
var _ = interfaces.VacanciesProvider(nil)
var _ = types.Filters{}
