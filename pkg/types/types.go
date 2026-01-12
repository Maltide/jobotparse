package types

type Client struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	Ttl          int    `json:"ttl"`
}

type Vacancy struct {
	ID            int      `json:"id"`             // ID вакансии
	Profession    string   `json:"profession"`     // Название вакансии
	IDCompany     int      `json:"id_company"`     // ID компании
	IDVacCreator  int      `json:"id_user"`        // ID пользователя, создавшего вакансию
	ExternalURL   string   `json:"external_url"`   // URL сайта с вакансией
	DatePubTo     int      `json:"date_pub_to"`    // Вакансия опубликована до (unixtime)
	DateArchived  int      `json:"date_archived"`  // Дата архивации вакансии (unixtime)
	DatePublished int      `json:"date_published"` // Дата публикации вакансии (unixtime)
	Work          string   `json:"work"`           // Должностные обязанности
	Compensation  string   `json:"compensation"`   // Условия работы
	Address       string   `json:"address"`        // Адрес компании (если указан)
	Candidate     string   `json:"candidate"`      // Требования к кандидату
	Town          Object   `json:"town"`           // Город
	TypeOfWork    Object   `json:"type_of_work"`   // Тип занятости
	PlaceOfWork   Object   `json:"place_of_work"`  // Место работы
	Education     Object   `json:"education"`      // Образование
	Experience    Object   `json:"experience"`     // Опыт работы
	Languages     []Object `json:"languages"`      // Иностранные языки
	Catalogues    []Object `json:"catalogues"`     // Категории вакансии
	Agency        Object   `json:"agency"`         // Тип работодателя
	Link          string   `json:"link"`           // Прямая ссылка на вакансию
	ViewsCount    int      `json:"views_count"`    // Количество просмотров вакансии
	Moveable      bool     `json:"moveable"`       // Рассматриваются соискатели из других городов
	FirmName      string   `json:"firm_name"`      // Название компании
	FirmActivity  string   `json:"firm_activity"`  // Описание деятельности компании
}

type VacanciesResponse struct {
	Objects []Vacancy `json:"objects"`
}

// Универсальная структура для вложенных объектов
type Object struct {
	ID         int      `json:"id"`                   // ID объекта
	Title      string   `json:"title"`                // Название объекта
	Declension string   `json:"declension,omitempty"` // Склонение (только для Town)
	Genitive   string   `json:"genitive,omitempty"`   // Родительный падеж (только для Town)
	Positions  []Object `json:"positions,omitempty"`  // Должности (только для Catalogues)
}

/*
Пояснения по использованию Object:

Town: ID — ID города, Title — название города, Declension — склонение, Genitive — родительный падеж
TypeOfWork: ID — тип занятости, Title — название. Значения ID: 6 — полный день, 10 — неполный день, 12 — сменный график, 13 — частичная занятость, 7 — временная работа, 9 — вахтовым методом
PlaceOfWork: ID — ID места, Title — название
Education: ID — ID образования, Title — название
Experience: ID — ID опыта, Title — название
Languages: ID — ID языка, Title — название
Catalogues: ID — ID категории, Title — название, Positions — []Object (ID — ID должности, Title — название)
Agency: ID — ID типа работодателя, Title — название
*/

type Filters struct {
	Profession string
	Town       string
	SalaryFrom string
	SalaryTo   string
	Skills     string
}
