package main

type User struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Username     string `json:"username"`
	Email        string `json:"email"`
	PasswordHash string `json:"password_hash"`
	Role         string `json:"role"`
	Active       bool   `json:"active"`
	CreatedAt    string `json:"created_at"`
	LastLoginAt  string `json:"last_login_at"`
}

type Session struct {
	ID        string `json:"id"`
	UserID    string `json:"user_id"`
	TokenHash string `json:"token_hash"`
	CSRFToken string `json:"csrf_token"`
	ExpiresAt string `json:"expires_at"`
	CreatedAt string `json:"created_at"`
}

type Invite struct {
	ID        string `json:"id"`
	TokenHash string `json:"token_hash"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	Role      string `json:"role"`
	ExpiresAt string `json:"expires_at"`
	UsedAt    string `json:"used_at"`
	CreatedBy string `json:"created_by"`
	CreatedAt string `json:"created_at"`
}

type Client struct {
	ID             string  `json:"id"`
	Name           string  `json:"name"`
	Document       string  `json:"document"`
	Phone          string  `json:"phone"`
	Email          string  `json:"email"`
	RBA            float64 `json:"rba"`
	Classification string  `json:"classification"`
	Notes          string  `json:"notes"`
	CreatedAt      string  `json:"created_at"`
}

type Property struct {
	ID        string  `json:"id"`
	ClientID  string  `json:"client_id"`
	Name      string  `json:"name"`
	City      string  `json:"city"`
	State     string  `json:"state"`
	AreaHa    float64 `json:"area_ha"`
	Registry  string  `json:"registry"`
	CAR       string  `json:"car"`
	CCIR      string  `json:"ccir"`
	ITR       string  `json:"itr"`
	Tenure    string  `json:"tenure"`
	Notes     string  `json:"notes"`
	CreatedAt string  `json:"created_at"`
}

type Project struct {
	ID               string  `json:"id"`
	ClientID         string  `json:"client_id"`
	PropertyID       string  `json:"property_id"`
	Title            string  `json:"title"`
	Modality         string  `json:"modality"`
	Activity         string  `json:"activity"`
	Bank             string  `json:"bank"`
	Program          string  `json:"program"`
	TotalValue       float64 `json:"total_value"`
	FinancedValue    float64 `json:"financed_value"`
	InterestRate     float64 `json:"interest_rate"`
	TermMonths       int     `json:"term_months"`
	Status           string  `json:"status"`
	Phase            string  `json:"phase"`
	TechnicalSummary string  `json:"technical_summary"`
	Alerts           string  `json:"alerts"`
	CreatedAt        string  `json:"created_at"`
	UpdatedAt        string  `json:"updated_at"`
	ClientName       string  `json:"-"`
	PropertyName     string  `json:"-"`
}

type ProjectFile struct {
	ID          string `json:"id"`
	ProjectID   string `json:"project_id"`
	Name        string `json:"name"`
	StoragePath string `json:"storage_path"`
	ContentType string `json:"content_type"`
	SizeBytes   int64  `json:"size_bytes"`
	UploadedBy  string `json:"uploaded_by"`
	CreatedAt   string `json:"created_at"`
}

type DailyReport struct {
	ID             string  `json:"id"`
	Number         int     `json:"number"`
	Date           string  `json:"date"`
	Value          float64 `json:"value"`
	Bank           string  `json:"bank"`
	Referral       string  `json:"referral"`
	Producer       string  `json:"producer"`
	ProjectType    string  `json:"project_type"`
	SentDate       string  `json:"sent_date"`
	Situation      string  `json:"situation"`
	CommissionRate float64 `json:"commission_rate"`
	Commission     float64 `json:"commission"`
	PedroPercent   float64 `json:"pedro_percent"`
	PaidBy         string  `json:"paid_by"`
	Notes          string  `json:"notes"`
	CreatedAt      string  `json:"created_at"`
}

type Rule struct {
	ID         string `json:"id"`
	Category   string `json:"category"`
	Title      string `json:"title"`
	Reference  string `json:"reference"`
	Summary    string `json:"summary"`
	SourceURL  string `json:"source_url"`
	VerifiedAt string `json:"verified_at"`
	Active     bool   `json:"active"`
}

type AuditLog struct {
	ID        string `json:"id"`
	UserID    string `json:"user_id"`
	Action    string `json:"action"`
	Entity    string `json:"entity"`
	EntityID  string `json:"entity_id"`
	Details   string `json:"details"`
	CreatedAt string `json:"created_at"`
}
