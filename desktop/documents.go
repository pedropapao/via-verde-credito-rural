package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

const propertyDocumentMaxBytes int64 = 50 << 20

type PropertyDocument struct {
	ID            int64  `json:"id"`
	PropertyID    int64  `json:"property_id"`
	DocType       string `json:"doc_type"`
	Title         string `json:"title"`
	OriginalName  string `json:"original_name"`
	StoredPath    string `json:"stored_path"`
	Source        string `json:"source"`
	IssueDate     string `json:"issue_date"`
	ExpiryDate    string `json:"expiry_date"`
	ReferenceYear string `json:"reference_year"`
	Notes         string `json:"notes"`
	SHA256        string `json:"sha256"`
	SizeBytes     int64  `json:"size_bytes"`
	IsCurrent     bool   `json:"is_current"`
	CreatedAt     string `json:"created_at"`
	UpdatedAt     string `json:"updated_at"`
}

type DocumentChecklistItem struct {
	DocType       string            `json:"doc_type"`
	Label         string            `json:"label"`
	Group         string            `json:"group"`
	SourceLabel   string            `json:"source_label"`
	SourceURL     string            `json:"source_url"`
	Automatic     bool              `json:"automatic"`
	Conditional   bool              `json:"conditional"`
	Status        string            `json:"status"`
	StatusLabel   string            `json:"status_label"`
	Detail        string            `json:"detail"`
	Notes         string            `json:"notes"`
	VersionCount  int               `json:"version_count"`
	Current       *PropertyDocument `json:"current,omitempty"`
}

type PropertyDocumentSummary struct {
	Received      int `json:"received"`
	Pending       int `json:"pending"`
	Review        int `json:"review"`
	Expired       int `json:"expired"`
	NotApplicable int `json:"not_applicable"`
}

type PropertyDocumentCenter struct {
	PropertyID int64                   `json:"property_id"`
	Items      []DocumentChecklistItem `json:"items"`
	Summary    PropertyDocumentSummary `json:"summary"`
	UpdatedAt  string                  `json:"updated_at"`
}

type documentDefinition struct {
	Type        string
	Label       string
	Group       string
	SourceLabel string
	SourceURL   string
	Automatic   bool
	Conditional bool
	Default     string
}

var propertyDocumentDefinitions = []documentDefinition{
	{Type: "car", Label: "Cadastro Ambiental Rural (CAR)", Group: "Imóvel e cadastro", SourceLabel: "SICAR", Automatic: true, Default: "pending"},
	{Type: "kml_sicar", Label: "KML SICAR", Group: "Imóvel e cadastro", SourceLabel: "SICAR", Automatic: true, Default: "pending"},
	{Type: "kml_client", Label: "KML do cliente", Group: "Imóvel e cadastro", SourceLabel: "Arquivo do cliente", Automatic: true, Conditional: true, Default: "review"},
	{Type: "registry", Label: "Matrícula / certidão do imóvel", Group: "Imóvel e cadastro", SourceLabel: "RI Digital / Cartório", SourceURL: "https://www.ridigital.org.br/", Default: "pending"},
	{Type: "ccir", Label: "CCIR", Group: "Imóvel e cadastro", SourceLabel: "INCRA / SNCR", SourceURL: "https://www.gov.br/pt-br/servicos/emitir-o-certificado-de-cadastro-de-imovel-rural-ccir", Default: "pending"},
	{Type: "cafir_cib", Label: "CAFIR / CIB", Group: "Imóvel e cadastro", SourceLabel: "Receita Federal / CNIR", SourceURL: "https://www.gov.br/pt-br/servicos/consultar-cadastro-nacional-de-imovel-rural", Default: "review"},
	{Type: "itr", Label: "ITR / DITR / recibo", Group: "Imóvel e cadastro", SourceLabel: "Receita Federal", SourceURL: "https://www.gov.br/receitafederal/pt-br/assuntos/orientacao-tributaria/declaracoes-e-demonstrativos/ditr", Default: "pending"},
	{Type: "producer_id", Label: "Documentos do produtor", Group: "Cliente", SourceLabel: "Cliente", Default: "pending"},
	{Type: "lease", Label: "Arrendamento / cessão / comodato", Group: "Atividade / projeto", SourceLabel: "Cliente / instrumento contratual", Conditional: true, Default: "review"},
	{Type: "outorga", Label: "Outorga / regularização hídrica", Group: "Atividade / projeto", SourceLabel: "ANA / órgão estadual", SourceURL: "https://www.gov.br/ana/pt-br/assuntos/gestao-das-aguas/politica-nacional-de-recursos-hidricos/outorga-dos-direitos-de-uso-de-recursos-hidricos", Conditional: true, Default: "review"},
	{Type: "gta", Label: "GTA / trânsito animal", Group: "Atividade / projeto", SourceLabel: "MAPA / defesa sanitária estadual", SourceURL: "https://www.gov.br/agricultura/pt-br/assuntos/sanidade-animal-e-vegetal/saude-animal/cgtqa/t_nacional/gta", Conditional: true, Default: "review"},
	{Type: "budget", Label: "Orçamento", Group: "Atividade / projeto", SourceLabel: "Fornecedor", Conditional: true, Default: "review"},
	{Type: "invoice", Label: "Nota fiscal", Group: "Atividade / projeto", SourceLabel: "Fornecedor / produtor", Conditional: true, Default: "review"},
	{Type: "technical_report", Label: "Laudo / documento técnico", Group: "Atividade / projeto", SourceLabel: "Profissional responsável", Conditional: true, Default: "review"},
}

func documentDefinitionByType(docType string) (documentDefinition, bool) {
	docType = strings.TrimSpace(docType)
	for _, d := range propertyDocumentDefinitions {
		if d.Type == docType {
			return d, true
		}
	}
	return documentDefinition{}, false
}

func validDocumentStatus(status string) bool {
	switch strings.TrimSpace(status) {
	case "received", "pending", "review", "expired", "not_applicable":
		return true
	default:
		return false
	}
}

func documentStatusLabel(status string) string {
	switch strings.TrimSpace(status) {
	case "received":
		return "Recebido"
	case "pending":
		return "Pendente"
	case "review":
		return "Conferir"
	case "expired":
		return "Vencido / desatualizado"
	case "not_applicable":
		return "Não se aplica"
	default:
		return "Conferir"
	}
}

func (a *App) AddPropertyDocument(propertyID int64, docType string) (PropertyDocument, error) {
	if a.ctx == nil {
		return PropertyDocument{}, errors.New("aplicativo ainda não inicializado")
	}
	if a.db == nil {
		return PropertyDocument{}, errors.New("banco local indisponível")
	}
	if propertyID <= 0 {
		return PropertyDocument{}, errors.New("salve ou selecione o imóvel antes de anexar documentos")
	}
	def, ok := documentDefinitionByType(docType)
	if !ok || def.Automatic {
		return PropertyDocument{}, errors.New("tipo de documento inválido para upload")
	}
	if _, err := a.GetProperty(propertyID); err != nil {
		return PropertyDocument{}, err
	}
	path, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Selecionar " + strings.ToLower(def.Label),
		Filters: []runtime.FileFilter{
			{DisplayName: "Documentos", Pattern: "*.pdf;*.png;*.jpg;*.jpeg;*.doc;*.docx;*.xls;*.xlsx;*.xml;*.txt"},
			{DisplayName: "Todos os arquivos", Pattern: "*.*"},
		},
	})
	if err != nil {
		return PropertyDocument{}, err
	}
	if strings.TrimSpace(path) == "" {
		return PropertyDocument{}, errors.New("importação cancelada")
	}
	info, err := os.Stat(path)
	if err != nil {
		return PropertyDocument{}, err
	}
	if info.IsDir() {
		return PropertyDocument{}, errors.New("selecione um arquivo")
	}
	if info.Size() <= 0 {
		return PropertyDocument{}, errors.New("o arquivo selecionado está vazio")
	}
	if info.Size() > propertyDocumentMaxBytes {
		return PropertyDocument{}, errors.New("arquivo maior que 50 MB")
	}

	now := time.Now()
	dir := filepath.Join(a.dataDir, "properties", strconv.FormatInt(propertyID, 10), "documents", docType)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return PropertyDocument{}, err
	}
	base := safePropertyDocumentFilename(filepath.Base(path))
	dest := filepath.Join(dir, now.Format("20060102_150405")+"_"+base)

	src, err := os.Open(path)
	if err != nil {
		return PropertyDocument{}, err
	}
	defer src.Close()
	dst, err := os.OpenFile(dest, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err != nil {
		return PropertyDocument{}, err
	}
	h := sha256.New()
	_, copyErr := io.Copy(io.MultiWriter(dst, h), src)
	closeErr := dst.Close()
	if copyErr != nil {
		_ = os.Remove(dest)
		return PropertyDocument{}, copyErr
	}
	if closeErr != nil {
		_ = os.Remove(dest)
		return PropertyDocument{}, closeErr
	}

	hash := hex.EncodeToString(h.Sum(nil))
	var existing PropertyDocument
	err = a.db.QueryRow(`SELECT id,property_id,doc_type,title,original_name,stored_path,source,issue_date,expiry_date,reference_year,notes,sha256,size_bytes,is_current,created_at,updated_at
		FROM property_documents WHERE property_id=? AND doc_type=? AND sha256=? ORDER BY id DESC LIMIT 1`, propertyID, docType, hash).
		Scan(&existing.ID, &existing.PropertyID, &existing.DocType, &existing.Title, &existing.OriginalName, &existing.StoredPath, &existing.Source,
			&existing.IssueDate, &existing.ExpiryDate, &existing.ReferenceYear, &existing.Notes, &existing.SHA256, &existing.SizeBytes, &existing.IsCurrent, &existing.CreatedAt, &existing.UpdatedAt)
	if err == nil {
		_ = os.Remove(dest)
		nowText := now.Format(time.RFC3339)
		tx, txErr := a.db.Begin()
		if txErr != nil {
			return PropertyDocument{}, txErr
		}
		defer tx.Rollback()
		if _, txErr = tx.Exec(`UPDATE property_documents SET is_current=0,updated_at=? WHERE property_id=? AND doc_type=? AND is_current=1`, nowText, propertyID, docType); txErr != nil {
			return PropertyDocument{}, txErr
		}
		if _, txErr = tx.Exec(`UPDATE property_documents SET is_current=1,updated_at=? WHERE id=?`, nowText, existing.ID); txErr != nil {
			return PropertyDocument{}, txErr
		}
		if _, txErr = tx.Exec(`INSERT INTO property_document_status(property_id,doc_type,status,notes,updated_at)
			VALUES(?,?,?,?,?) ON CONFLICT(property_id,doc_type) DO UPDATE SET status=excluded.status,updated_at=excluded.updated_at`,
			propertyID, docType, "received", "", nowText); txErr != nil {
			return PropertyDocument{}, txErr
		}
		if txErr = tx.Commit(); txErr != nil {
			return PropertyDocument{}, txErr
		}
		return a.getPropertyDocument(existing.ID)
	}

	tx, err := a.db.Begin()
	if err != nil {
		_ = os.Remove(dest)
		return PropertyDocument{}, err
	}
	defer tx.Rollback()

	// Mantém versões anteriores no banco e no disco, mas apenas uma versão
	// corrente por item do checklist.
	if _, err := tx.Exec(`UPDATE property_documents SET is_current=0,updated_at=? WHERE property_id=? AND doc_type=? AND is_current=1`, now.Format(time.RFC3339), propertyID, docType); err != nil {
		_ = os.Remove(dest)
		return PropertyDocument{}, err
	}
	res, err := tx.Exec(`INSERT INTO property_documents(
		property_id,doc_type,title,original_name,stored_path,source,issue_date,expiry_date,reference_year,notes,sha256,size_bytes,is_current,created_at,updated_at
	) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,1,?,?)`,
		propertyID, docType, def.Label, filepath.Base(path), dest, def.SourceLabel, "", "", "", "", hash, info.Size(), now.Format(time.RFC3339), now.Format(time.RFC3339))
	if err != nil {
		_ = os.Remove(dest)
		return PropertyDocument{}, err
	}
	id, _ := res.LastInsertId()
	if _, err := tx.Exec(`INSERT INTO property_document_status(property_id,doc_type,status,notes,updated_at)
		VALUES(?,?,?,?,?) ON CONFLICT(property_id,doc_type) DO UPDATE SET status=excluded.status,updated_at=excluded.updated_at`,
		propertyID, docType, "received", "", now.Format(time.RFC3339)); err != nil {
		_ = os.Remove(dest)
		return PropertyDocument{}, err
	}
	if err := tx.Commit(); err != nil {
		_ = os.Remove(dest)
		return PropertyDocument{}, err
	}
	return a.getPropertyDocument(id)
}

func (a *App) UpdatePropertyDocumentMeta(doc PropertyDocument) (PropertyDocument, error) {
	if a.db == nil {
		return PropertyDocument{}, errors.New("banco local indisponível")
	}
	if doc.ID <= 0 {
		return PropertyDocument{}, errors.New("documento inválido")
	}
	current, err := a.getPropertyDocument(doc.ID)
	if err != nil {
		return PropertyDocument{}, err
	}
	now := time.Now().Format(time.RFC3339)
	_, err = a.db.Exec(`UPDATE property_documents SET issue_date=?,expiry_date=?,reference_year=?,notes=?,updated_at=? WHERE id=?`,
		strings.TrimSpace(doc.IssueDate), strings.TrimSpace(doc.ExpiryDate), strings.TrimSpace(doc.ReferenceYear), strings.TrimSpace(doc.Notes), now, doc.ID)
	if err != nil {
		return PropertyDocument{}, err
	}
	current.IssueDate = strings.TrimSpace(doc.IssueDate)
	current.ExpiryDate = strings.TrimSpace(doc.ExpiryDate)
	current.ReferenceYear = strings.TrimSpace(doc.ReferenceYear)
	current.Notes = strings.TrimSpace(doc.Notes)
	current.UpdatedAt = now
	return current, nil
}

func (a *App) SetPropertyDocumentStatus(propertyID int64, docType, status, notes string) error {
	if a.db == nil {
		return errors.New("banco local indisponível")
	}
	if propertyID <= 0 {
		return errors.New("imóvel inválido")
	}
	if _, ok := documentDefinitionByType(docType); !ok {
		return errors.New("tipo de documento inválido")
	}
	status = strings.TrimSpace(status)
	if !validDocumentStatus(status) {
		return errors.New("status de documento inválido")
	}
	_, err := a.db.Exec(`INSERT INTO property_document_status(property_id,doc_type,status,notes,updated_at)
		VALUES(?,?,?,?,?) ON CONFLICT(property_id,doc_type) DO UPDATE SET status=excluded.status,notes=excluded.notes,updated_at=excluded.updated_at`,
		propertyID, docType, status, strings.TrimSpace(notes), time.Now().Format(time.RFC3339))
	return err
}

func (a *App) ArchivePropertyDocument(documentID int64) error {
	if a.db == nil {
		return errors.New("banco local indisponível")
	}
	doc, err := a.getPropertyDocument(documentID)
	if err != nil {
		return err
	}
	now := time.Now().Format(time.RFC3339)
	tx, err := a.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`UPDATE property_documents SET is_current=0,updated_at=? WHERE id=?`, now, documentID); err != nil {
		return err
	}
	if _, err := tx.Exec(`INSERT INTO property_document_status(property_id,doc_type,status,notes,updated_at)
		VALUES(?,?,?,?,?) ON CONFLICT(property_id,doc_type) DO UPDATE SET status=excluded.status,updated_at=excluded.updated_at`,
		doc.PropertyID, doc.DocType, "pending", "", now); err != nil {
		return err
	}
	return tx.Commit()
}

func (a *App) OpenPropertyDocument(documentID int64) error {
	if a.ctx == nil {
		return errors.New("aplicativo ainda não inicializado")
	}
	doc, err := a.getPropertyDocument(documentID)
	if err != nil {
		return err
	}
	if err := a.ensureManagedDocumentPath(doc.StoredPath); err != nil {
		return err
	}
	if _, err := os.Stat(doc.StoredPath); err != nil {
		return errors.New("arquivo do documento não foi localizado")
	}
	runtime.BrowserOpenURL(a.ctx, "file:///"+filepath.ToSlash(doc.StoredPath))
	return nil
}

func (a *App) ListPropertyDocumentHistory(propertyID int64, docType string) ([]PropertyDocument, error) {
	if a.db == nil {
		return nil, errors.New("banco local indisponível")
	}
	if propertyID <= 0 {
		return nil, errors.New("imóvel inválido")
	}
	if _, ok := documentDefinitionByType(docType); !ok {
		return nil, errors.New("tipo de documento inválido")
	}
	rows, err := a.db.Query(`SELECT id,property_id,doc_type,title,original_name,stored_path,source,issue_date,expiry_date,reference_year,notes,sha256,size_bytes,is_current,created_at,updated_at
		FROM property_documents WHERE property_id=? AND doc_type=? ORDER BY created_at DESC,id DESC`, propertyID, docType)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []PropertyDocument
	for rows.Next() {
		var d PropertyDocument
		if err := rows.Scan(&d.ID, &d.PropertyID, &d.DocType, &d.Title, &d.OriginalName, &d.StoredPath, &d.Source, &d.IssueDate, &d.ExpiryDate, &d.ReferenceYear, &d.Notes, &d.SHA256, &d.SizeBytes, &d.IsCurrent, &d.CreatedAt, &d.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

func (a *App) GetPropertyDocumentCenter(propertyID int64) (PropertyDocumentCenter, error) {
	if a.db == nil {
		return PropertyDocumentCenter{}, errors.New("banco local indisponível")
	}
	if propertyID <= 0 {
		return PropertyDocumentCenter{}, errors.New("salve ou selecione um imóvel para usar a central de documentos")
	}
	p, err := a.GetProperty(propertyID)
	if err != nil {
		return PropertyDocumentCenter{}, err
	}
	var lastCARJSON string
	_ = a.db.QueryRow(`SELECT last_car_json FROM properties WHERE id=?`, propertyID).Scan(&lastCARJSON)
	var car CARResult
	if strings.TrimSpace(lastCARJSON) != "" {
		_ = json.Unmarshal([]byte(lastCARJSON), &car)
	}

	currentDocs := map[string]PropertyDocument{}
	versionCounts := map[string]int{}
	rows, err := a.db.Query(`SELECT id,property_id,doc_type,title,original_name,stored_path,source,issue_date,expiry_date,reference_year,notes,sha256,size_bytes,is_current,created_at,updated_at
		FROM property_documents WHERE property_id=? ORDER BY created_at DESC,id DESC`, propertyID)
	if err != nil {
		return PropertyDocumentCenter{}, err
	}
	for rows.Next() {
		var d PropertyDocument
		if err := rows.Scan(&d.ID, &d.PropertyID, &d.DocType, &d.Title, &d.OriginalName, &d.StoredPath, &d.Source, &d.IssueDate, &d.ExpiryDate, &d.ReferenceYear, &d.Notes, &d.SHA256, &d.SizeBytes, &d.IsCurrent, &d.CreatedAt, &d.UpdatedAt); err != nil {
			rows.Close()
			return PropertyDocumentCenter{}, err
		}
		versionCounts[d.DocType]++
		if d.IsCurrent {
			if _, exists := currentDocs[d.DocType]; !exists {
				currentDocs[d.DocType] = d
			}
		}
	}
	if err := rows.Close(); err != nil {
		return PropertyDocumentCenter{}, err
	}

	statuses := map[string]string{}
	statusNotes := map[string]string{}
	statusRows, err := a.db.Query(`SELECT doc_type,status,notes FROM property_document_status WHERE property_id=?`, propertyID)
	if err != nil {
		return PropertyDocumentCenter{}, err
	}
	for statusRows.Next() {
		var typ, status, notes string
		if err := statusRows.Scan(&typ, &status, &notes); err != nil {
			statusRows.Close()
			return PropertyDocumentCenter{}, err
		}
		statuses[typ], statusNotes[typ] = status, notes
	}
	_ = statusRows.Close()

	out := PropertyDocumentCenter{PropertyID: propertyID, UpdatedAt: time.Now().Format(time.RFC3339)}
	for _, def := range propertyDocumentDefinitions {
		item := DocumentChecklistItem{
			DocType: def.Type, Label: def.Label, Group: def.Group, SourceLabel: def.SourceLabel, SourceURL: def.SourceURL,
			Automatic: def.Automatic, Conditional: def.Conditional, Status: def.Default, Notes: statusNotes[def.Type], VersionCount: versionCounts[def.Type],
		}

		switch def.Type {
		case "car":
			if strings.TrimSpace(p.CARNumber) != "" {
				item.Status = "received"
				item.Detail = p.CARNumber
			} else {
				item.Detail = "CAR ainda não vinculado ao imóvel."
			}
		case "kml_sicar":
			if strings.TrimSpace(car.AutoKMLPath) != "" {
				if _, statErr := os.Stat(car.AutoKMLPath); statErr == nil {
					item.Status = "received"
					item.Detail = "Perímetro público SICAR armazenado automaticamente."
				} else {
					item.Status = "review"
					item.Detail = "Geometria pública registrada; arquivo KML automático precisa ser regenerado."
				}
			} else if car.HasGeometry {
				item.Status = "review"
				item.Detail = "Geometria pública disponível; KML automático ainda não confirmado no disco."
			} else {
				item.Status = "pending"
				item.Detail = "Analise o CAR para gerar o KML SICAR."
			}
		case "kml_client":
			if strings.TrimSpace(p.KMLPath) != "" {
				if _, statErr := os.Stat(p.KMLPath); statErr == nil {
					item.Status = "received"
					item.Detail = filepath.Base(p.KMLPath)
				} else {
					item.Status = "review"
					item.Detail = "O cadastro aponta um KML externo, mas o arquivo não foi localizado."
				}
			} else {
				item.Status = "review"
				item.Detail = "Opcional para conferência independente CAR x levantamento externo."
			}
		default:
			if d, ok := currentDocs[def.Type]; ok {
				doc := d
				item.Current = &doc
				item.Status = "received"
				item.Detail = d.OriginalName
			}
			if explicit := strings.TrimSpace(statuses[def.Type]); explicit != "" {
				item.Status = explicit
			}
			if d, ok := currentDocs[def.Type]; ok && isExpiredDocument(d.ExpiryDate) && item.Status != "not_applicable" {
				item.Status = "expired"
				item.Detail = d.OriginalName + " • validade expirada"
			}
		}
		item.StatusLabel = documentStatusLabel(item.Status)
		out.Items = append(out.Items, item)
		switch item.Status {
		case "received":
			out.Summary.Received++
		case "pending":
			out.Summary.Pending++
		case "review":
			out.Summary.Review++
		case "expired":
			out.Summary.Expired++
		case "not_applicable":
			out.Summary.NotApplicable++
		}
	}
	return out, nil
}

func (a *App) getPropertyDocument(id int64) (PropertyDocument, error) {
	if a.db == nil {
		return PropertyDocument{}, errors.New("banco local indisponível")
	}
	var d PropertyDocument
	err := a.db.QueryRow(`SELECT id,property_id,doc_type,title,original_name,stored_path,source,issue_date,expiry_date,reference_year,notes,sha256,size_bytes,is_current,created_at,updated_at
		FROM property_documents WHERE id=?`, id).
		Scan(&d.ID, &d.PropertyID, &d.DocType, &d.Title, &d.OriginalName, &d.StoredPath, &d.Source, &d.IssueDate, &d.ExpiryDate, &d.ReferenceYear, &d.Notes, &d.SHA256, &d.SizeBytes, &d.IsCurrent, &d.CreatedAt, &d.UpdatedAt)
	return d, err
}

func (a *App) ensureManagedDocumentPath(path string) error {
	root, err := filepath.Abs(filepath.Join(a.dataDir, "properties"))
	if err != nil {
		return err
	}
	target, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	rel, err := filepath.Rel(root, target)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return errors.New("caminho de documento fora da área gerenciada pelo ViaVerdeCAR")
	}
	return nil
}

func isExpiredDocument(expiry string) bool {
	expiry = strings.TrimSpace(expiry)
	if expiry == "" {
		return false
	}
	t, err := time.Parse("2006-01-02", expiry)
	if err != nil {
		return false
	}
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	return t.Before(today)
}

func safePropertyDocumentFilename(name string) string {
	name = strings.TrimSpace(filepath.Base(name))
	if name == "" || name == "." {
		return "documento"
	}
	var b strings.Builder
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '.', r == '-', r == '_':
			b.WriteRune(r)
		case r == ' ', r == 'á', r == 'à', r == 'ã', r == 'â', r == 'é', r == 'ê', r == 'í', r == 'ó', r == 'ô', r == 'õ', r == 'ú', r == 'ç',
			r == 'Á', r == 'À', r == 'Ã', r == 'Â', r == 'É', r == 'Ê', r == 'Í', r == 'Ó', r == 'Ô', r == 'Õ', r == 'Ú', r == 'Ç':
			b.WriteRune('_')
		default:
			b.WriteRune('_')
		}
	}
	out := strings.Trim(b.String(), "._ ")
	if out == "" {
		return "documento"
	}
	if len(out) > 120 {
		ext := filepath.Ext(out)
		base := strings.TrimSuffix(out, ext)
		limit := 120 - len(ext)
		if limit < 20 {
			limit = 20
		}
		if len(base) > limit {
			base = base[:limit]
		}
		out = base + ext
	}
	return out
}
