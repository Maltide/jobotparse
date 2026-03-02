package types

// Client represents SuperJob OAuth tokens as returned by their API.
type Client struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	Ttl          int    `json:"ttl"`
}

// Vacancy is a normalized vacancy entity persisted into Postgres.
// The struct uses both JSON tags and GORM tags (for DB).
type Vacancy struct {
	ExternalID      int         `json:"id"`             // ID вакансии из внешнего API
	Profession      string      `json:"profession"`     // Название вакансии
	IDCompany       int         `json:"id_company"`     // ID компании
	IDVacCreator    int         `json:"id_user"`        // ID пользователя, создавшего вакансию
	ExternalURL     string      `json:"external_url"`   // URL сайта с вакансией
	DatePubTo       int         `json:"date_pub_to"`    // Вакансия опубликована до (unixtime)
	DateArchived    int         `json:"date_archived"`  // Дата архивации вакансии (unixtime)
	DatePublished   int         `json:"date_published"` // Дата публикации вакансии (unixtime)
	Work            string      `json:"work"`           // Должностные обязанности
	TypeOfWork      *TypeOfWork `json:"type_of_work"`   // Тип занятости (только для JSON; в БД не храним как relation)
	TypeOfWorkTitle string      `json:"-"`              // Тип занятости (текстом)
	Experience      *Experience `json:"experience"`     // Опыт работы (только для JSON; в БД не храним как relation)
	ExperienceTitle string      `json:"-"`              // Опыт работы (текстом)
	PaymentFrom     int         `json:"payment_from"`   // Зарплата от
	PaymentTo       int         `json:"payment_to"`     // Зарплата до
	Currency        string      `json:"currency"`       // Валюта зарплаты
	Compensation    string      `json:"compensation"`   // Условия работы
	Address         string      `json:"address"`        // Адрес компании (если указан)
	Link            string      `json:"link"`           // Прямая ссылка на вакансию
	ViewsCount      int         `json:"views_count"`    // Количество просмотров вакансии
	Moveable        bool        `json:"moveable"`       // Рассматриваются соискатели из других городов
	FirmName        string      `json:"firm_name"`      // Название компании
	FirmActivity    string      `json:"firm_activity"`  // Описание деятельности компании
	Town            *Town       `json:"town"`           // Город вакансии
	TownName        string      `json:"-"`              // Название города
}

// VacanciesResponse is the API response wrapper.
type VacanciesResponse struct {
	Objects []Vacancy `json:"objects"`
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

// Resume mirrors expected JSON fields for the resume endpoint.
type Resume struct {
	VacancyURL    string      `json:"vacancy_url"`
	FullName      string      `json:"full_name"`
	Gender        string      `json:"gender"`
	Age           string      `json:"age"`
	DOB           string      `json:"dob"`
	Position      string      `json:"position"`
	DesiredSalary string      `json:"desired_salary"`
	JobType       string      `json:"job_type"`
	WorkFormats   []string    `json:"work_formats"`
	Contact       string      `json:"contact"`
	Location      string      `json:"location"`
	Citizenship   string      `json:"citizenship"`
	Relocate      string      `json:"relocate"`
	Travel        string      `json:"travel"`
	Summary       string      `json:"summary"`
	Skills        []string    `json:"skills"`
	Languages     []string    `json:"languages"`
	Work          []Work      `json:"work"`
	Education     []Education `json:"education"`
}

// Education represents one education entry in a resume.
type Education struct {
	Institution string `json:"institution"`
	Degree      string `json:"degree"`
	StartDate   string `json:"startDate"`
	EndDate     string `json:"endDate"`
	Location    string `json:"location"`
}

// Work represents one workplace in a resume.
type Work struct {
	Company     string `json:"company"`
	Position    string `json:"position"`
	From        string `json:"from"`
	To          string `json:"to"`
	Description string `json:"description"`
}
