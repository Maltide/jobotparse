package types

type Client struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	Ttl          int    `json:"ttl"`
}

type Vacancy struct {
	ID            int    `gorm:"primaryKey;autoIncrement" json:"-"` // локальный PK в БД (автоинкремент)
	ExternalID    int    `json:"id" gorm:"uniqueIndex"`             // ID вакансии из внешнего API
	Profession    string `gorm:"not null" json:"profession"`        // Название вакансии
	IDCompany     int    `json:"id_company"`                        // ID компании
	IDVacCreator  int    `json:"id_user"`                           // ID пользователя, создавшего вакансию
	ExternalURL   string `json:"external_url"`                      // URL сайта с вакансией
	DatePubTo     int    `json:"date_pub_to"`                       // Вакансия опубликована до (unixtime)
	DateArchived  int    `json:"date_archived"`                     // Дата архивации вакансии (unixtime)
	DatePublished int    `json:"date_published"`                    // Дата публикации вакансии (unixtime)
	Work          string `json:"work"`                              // Должностные обязанности
	Compensation  string `json:"compensation"`                      // Условия работы
	Address       string `json:"address"`                           // Адрес компании (если указан)
	Candidate     string `json:"candidate"`                         // Требования к кандидату
	Link          string `json:"link"`                              // Прямая ссылка на вакансию
	ViewsCount    int    `json:"views_count"`                       // Количество просмотров вакансии
	Moveable      bool   `json:"moveable"`                          // Рассматриваются соискатели из других городов
	FirmName      string `json:"firm_name"`                         // Название компании
	FirmActivity  string `json:"firm_activity"`                     // Описание деятельности компании
	Town          *Town  `json:"town" gorm:"-"`                     // Город вакансии
	TownName      string `json:"-" gorm:"column:town_name"`         // Название города
}

type VacanciesResponse struct {
	Objects []Vacancy `json:"objects"`
}

type Town struct {
	ID    int    `json:"id"`
	Title string `json:"title"`
}

type Filters struct {
	Profession string
	Town       string
	SalaryFrom string
	SalaryTo   string
	Skills     string
}
