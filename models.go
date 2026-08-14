package main

type User struct {
	ID           string `json:"id,omitempty"`
	Name         string `json:"name"`
	Username     string `json:"username"`
	Email        string `json:"email"`
	PasswordHash string `json:"password_hash"`
	Role         string `json:"role"`
	Active       bool   `json:"active"`
	CreatedAt    string `json:"created_at,omitempty"`
	LastLoginAt  string `json:"last_login_at,omitempty"`
}

type Session struct {
	ID        string `json:"id,omitempty"`
	UserID    string `json:"user_id"`
	TokenHash string `json:"token_hash"`
	CSRFToken string `json:"csrf_token"`
	ExpiresAt string `json:"expires_at"`
	CreatedAt string `json:"created_at,omitempty"`
}

type Invite struct {
	ID        string `json:"id,omitempty"`
	TokenHash string `json:"token_hash"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	Role      string `json:"role"`
	ExpiresAt string `json:"expires_at"`
	UsedAt    string `json:"used_at,omitempty"`
	CreatedBy string `json:"created_by,omitempty"`
	CreatedAt string `json:"created_at,omitempty"`
}

type Client struct {
	ID             string  `json:"id,omitempty"`
	Name           string  `json:"name"`
	Document       string  `json:"document"`
	Phone          string  `json:"phone"`
	Email          string  `json:"email"`
	RBA            float64 `json:"rba"`
	Classification string  `json:"classification"`
	Notes          string  `json:"notes"`
	CreatedAt      string  `json:"created_at,omitempty"`
}

type Property struct {
	ID                string  `json:"id,omitempty"`
	ClientID          string  `json:"client_id"`
	Name              string  `json:"name"`
	City              string  `json:"city"`
	State             string  `json:"state"`
	AreaHa            float64 `json:"area_ha"`
	Registry          string  `json:"registry"`
	CAR               string  `json:"car"`
	CCIR              string  `json:"ccir"`
	ITR               string  `json:"itr"`
	Tenure            string  `json:"tenure"`
	Notes             string  `json:"notes"`
	Latitude          float64 `json:"latitude,omitempty"`
	Longitude         float64 `json:"longitude,omitempty"`
	WaterInfo         string  `json:"water_info,omitempty"`
	EnvironmentalInfo string  `json:"environmental_info,omitempty"`
	AccessInfo        string  `json:"access_info,omitempty"`
	OwnerName         string  `json:"owner_name,omitempty"`
	RegistryDate      string  `json:"registry_date,omitempty"`
	CARStatus         string  `json:"car_status,omitempty"`
	CreatedAt         string  `json:"created_at,omitempty"`
}

type Project struct {
	ID               string  `json:"id,omitempty"`
	ClientID         string  `json:"client_id"`
	PropertyID       string  `json:"property_id,omitempty"`
	Title            string  `json:"title"`
	Modality         string  `json:"modality"`
	Activity         string  `json:"activity"`
	Bank             string  `json:"bank"`
	Program          string  `json:"program"`
	TotalValue       float64 `json:"total_value"`
	FinancedValue    float64 `json:"financed_value"`
	OwnResources     float64 `json:"own_resources,omitempty"`
	InterestRate     float64 `json:"interest_rate"`
	TermMonths       int     `json:"term_months"`
	GraceMonths      int     `json:"grace_months,omitempty"`
	PaymentFrequency string  `json:"payment_frequency,omitempty"`
	Status           string  `json:"status"`
	Phase            string  `json:"phase"`
	Responsible      string  `json:"responsible,omitempty"`
	SentAt           string  `json:"sent_at,omitempty"`
	ContractedAt     string  `json:"contracted_at,omitempty"`
	TechnicalSummary string  `json:"technical_summary"`
	Alerts           string  `json:"alerts"`
	SensitiveNotes   string  `json:"sensitive_notes,omitempty"`
	CreatedAt        string  `json:"created_at,omitempty"`
	UpdatedAt        string  `json:"updated_at,omitempty"`
	ClientName       string  `json:"-"`
	PropertyName     string  `json:"-"`
}

type ProjectFile struct {
	ID            string `json:"id,omitempty"`
	ProjectID     string `json:"project_id"`
	Name          string `json:"name"`
	StoragePath   string `json:"storage_path"`
	ContentType   string `json:"content_type"`
	SizeBytes     int64  `json:"size_bytes"`
	UploadedBy    string `json:"uploaded_by,omitempty"`
	DocumentType  string `json:"document_type,omitempty"`
	VersionLabel  string `json:"version_label,omitempty"`
	ValidUntil    string `json:"valid_until,omitempty"`
	IsCurrent     bool   `json:"is_current,omitempty"`
	CreatedAt     string `json:"created_at,omitempty"`
}

type DailyReport struct {
	ID             string  `json:"id,omitempty"`
	ProjectID      string  `json:"project_id,omitempty"`
	EventType      string  `json:"event_type,omitempty"`
	Number         int     `json:"number"`
	Date           string  `json:"date"`
	Value          float64 `json:"value"`
	Bank           string  `json:"bank"`
	Referral       string  `json:"referral"`
	Producer       string  `json:"producer"`
	ProjectType    string  `json:"project_type"`
	SentDate       string  `json:"sent_date,omitempty"`
	Situation      string  `json:"situation"`
	CommissionRate float64 `json:"commission_rate"`
	Commission     float64 `json:"commission"`
	PedroPercent   float64 `json:"pedro_percent"`
	PaidBy         string  `json:"paid_by"`
	Notes          string  `json:"notes"`
	CreatedAt      string  `json:"created_at,omitempty"`
}

type Rule struct {
	ID         string `json:"id,omitempty"`
	Category   string `json:"category"`
	Title      string `json:"title"`
	Reference  string `json:"reference"`
	Summary    string `json:"summary"`
	SourceURL  string `json:"source_url"`
	VerifiedAt string `json:"verified_at,omitempty"`
	Active     bool   `json:"active"`
}

type AuditLog struct {
	ID        string `json:"id,omitempty"`
	UserID    string `json:"user_id,omitempty"`
	Action    string `json:"action"`
	Entity    string `json:"entity"`
	EntityID  string `json:"entity_id"`
	Details   string `json:"details"`
	CreatedAt string `json:"created_at,omitempty"`
}
