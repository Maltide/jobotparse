package types

// Client represents SuperJob OAuth tokens as returned by their API.
type Client struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	Ttl          int    `json:"ttl"`
}

// VacancySJ is a normalized vacancy entity persisted into Postgres.
// The struct uses both JSON tags and GORM tags (for DB).
type VacancySJ struct {
	ID              int         `gorm:"primaryKey;autoIncrement" json:"-"`                               // локальный PK в БД (автоинкремент)
	ExternalID      int         `json:"id" gorm:"uniqueIndex:uniq_vacancy_source_external"`              // ID вакансии из внешнего API
	Profession      string      `gorm:"not null" json:"profession"`                                      // Название вакансии
	IDCompany       int         `json:"id_company"`                                                      // ID компании
	IDVacCreator    int         `json:"id_user"`                                                         // ID пользователя, создавшего вакансию
	ExternalURL     string      `json:"external_url"`                                                    // URL сайта с вакансией
	DatePubTo       int         `json:"date_pub_to"`                                                     // Вакансия опубликована до (unixtime)
	DateArchived    int         `json:"date_archived"`                                                   // Дата архивации вакансии (unixtime)
	DatePublished   int         `json:"date_published"`                                                  // Дата публикации вакансии (unixtime)
	Work            string      `json:"work"`                                                            // Должностные обязанности
	TypeOfWork      *TypeOfWork `json:"type_of_work" gorm:"-"`                                           // Тип занятости (только для JSON; в БД не храним как relation)
	TypeOfWorkTitle string      `json:"-" gorm:"column:type_of_work_title"`                              // Тип занятости (текстом)
	Experience      *Experience `json:"experience" gorm:"-"`                                             // Опыт работы (только для JSON; в БД не храним как relation)
	ExperienceTitle string      `json:"-" gorm:"column:experience_title"`                                // Опыт работы (текстом)
	PaymentFrom     int         `json:"payment_from"`                                                    // Зарплата от
	PaymentTo       int         `json:"payment_to"`                                                      // Зарплата до
	Currency        string      `json:"currency"`                                                        // Валюта зарплаты
	Compensation    string      `json:"compensation"`                                                    // Условия работы
	Address         string      `json:"address"`                                                         // Адрес компании (если указан)
	Link            string      `json:"link"`                                                            // Прямая ссылка на вакансию
	ViewsCount      int         `json:"views_count"`                                                     // Количество просмотров вакансии
	Moveable        bool        `json:"moveable"`                                                        // Рассматриваются соискатели из других городов
	FirmName        string      `json:"firm_name"`                                                       // Название компании
	FirmActivity    string      `json:"firm_activity"`                                                   // Описание деятельности компании
	Town            *Town       `json:"town" gorm:"-"`                                                   // Город вакансии
	TownName        string      `json:"-" gorm:"column:town_name"`                                       // Название города
	Source          string      `json:"-" gorm:"column:source;uniqueIndex:uniq_vacancy_source_external"` // Источник вакансии (например, "SuperJob")
}

// VacanciesResponse is the API response wrapper.
type VacanciesResponse struct {
	Objects []VacancySJ `json:"objects"`
}

type TypeOfWork struct {
	ID    int    `json:"id"`
	Title string `json:"title"`
}

// Town is a nested object inside a vacancy.
type Town struct {
	ID    int    `json:"id"`
	Title string `json:"title"`
}

type Experience struct {
	ID    int    `json:"id"`
	Title string `json:"title"`
}

// Filters contains user-provided search parameters.
type Filters struct {
	Profession string
	Town       string
	SalaryFrom string
	SalaryTo   string
	Skills     string
}
