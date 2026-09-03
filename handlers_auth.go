package main

import (
	"crypto/subtle"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"
)

func (a *App) loginGet(w http.ResponseWriter, r *http.Request) {
	if u, _, _ := a.loadAuth(r); u != nil {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	data := ViewData{Title: "Entrar", Data: map[string]any{"Username": a.cfg.AdminUsername, "Next": r.URL.Query().Get("next")}}
	a.mu.RLock()
	startupErr := a.startupErr
	a.mu.RUnlock()
	if startupErr != nil {
		data.Error = startupErr.Error()
	}
	a.render(w, r, "login", data)
}

func configuredAdminMatches(cfg Config, username, password string) bool {
	username = strings.ToLower(strings.TrimSpace(username))
	admin := strings.ToLower(strings.TrimSpace(cfg.AdminUsername))
	if username == "" || admin == "" || username != admin || cfg.AdminPassword == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(password), []byte(cfg.AdminPassword)) == 1
}

func (a *App) loginPost(w http.ResponseWriter, r *http.Request) {
	if a.sb == nil {
		a.render(w, r, "login", ViewData{Title: "Entrar", Error: "Banco de dados ainda não configurado. Veja o guia de implantação."})
		return
	}
	username := strings.ToLower(strings.TrimSpace(r.FormValue("username")))
	password := r.FormValue("password")
	var users []User
	if err := a.sb.Select(r.Context(), "users", eq("username", username)+"&limit=1", &users); err != nil {
		log.Printf("login: falha ao consultar usuario %q: %v", username, err)
		a.render(w, r, "login", ViewData{Title: "Entrar", Error: "Não foi possível acessar o banco de dados agora. Tente novamente em instantes.", Data: map[string]any{"Username": username}})
		return
	}

	// Uma consulta bem-sucedida confirma que eventual erro de bootstrap foi transitório.
	a.mu.Lock()
	a.startupErr = nil
	a.mu.Unlock()

	adminRecovery := configuredAdminMatches(a.cfg, username, password)
	if len(users) == 0 {
		if !adminRecovery {
			a.render(w, r, "login", ViewData{Title: "Entrar", Error: "Usuário ou senha inválidos.", Data: map[string]any{"Username": username}})
			return
		}
		hash, err := HashPassword(password)
		if err != nil {
			a.render(w, r, "login", ViewData{Title: "Entrar", Error: "Não foi possível recuperar o acesso do proprietário."})
			return
		}
		var created []User
		row := User{Name: a.cfg.AdminName, Username: username, PasswordHash: hash, Role: "owner", Active: true}
		if err := a.sb.Insert(r.Context(), "users", row, &created); err != nil || len(created) == 0 {
			log.Printf("login: falha ao recriar proprietario %q: %v", username, err)
			a.render(w, r, "login", ViewData{Title: "Entrar", Error: "Não foi possível recuperar o acesso do proprietário agora."})
			return
		}
		users = created
		log.Printf("login: acesso do proprietario recriado pelo segredo de ambiente")
	}

	user := users[0]
	passwordOK := user.Active && VerifyPassword(password, user.PasswordHash)
	if !passwordOK && adminRecovery {
		hash, err := HashPassword(password)
		if err != nil {
			a.render(w, r, "login", ViewData{Title: "Entrar", Error: "Não foi possível recuperar o acesso do proprietário."})
			return
		}
		var repaired []User
		vals := map[string]any{"password_hash": hash, "active": true, "role": "owner"}
		if strings.TrimSpace(user.Name) == "" && strings.TrimSpace(a.cfg.AdminName) != "" {
			vals["name"] = a.cfg.AdminName
		}
		if err := a.sb.Update(r.Context(), "users", eq("id", user.ID), vals, &repaired); err != nil || len(repaired) == 0 {
			log.Printf("login: falha ao reparar proprietario %q: %v", username, err)
			a.render(w, r, "login", ViewData{Title: "Entrar", Error: "Não foi possível recuperar o acesso do proprietário agora."})
			return
		}
		user = repaired[0]
		passwordOK = true
		log.Printf("login: credencial do proprietario sincronizada com o segredo de ambiente")
	}
	if !passwordOK {
		a.render(w, r, "login", ViewData{Title: "Entrar", Error: "Usuário ou senha inválidos.", Data: map[string]any{"Username": username}})
		return
	}

	// Mantém a tabela de sessões limpa sem afetar sessões ainda válidas.
	expired := eq("user_id", user.ID) + "&expires_at=lt." + url.QueryEscape(time.Now().UTC().Format(time.RFC3339))
	_ = a.sb.Delete(r.Context(), "sessions", expired)

	raw, _, err := a.createSession(r.Context(), user.ID)
	if err != nil {
		log.Printf("login: falha ao criar sessao para %q: %v", username, err)
		a.render(w, r, "login", ViewData{Title: "Entrar", Error: "A senha foi conferida, mas não foi possível iniciar a sessão. Tente novamente em instantes.", Data: map[string]any{"Username": username}})
		return
	}
	a.setSessionCookie(w, r, raw)
	_ = a.sb.Update(r.Context(), "users", eq("id", user.ID), map[string]any{"last_login_at": time.Now().UTC().Format(time.RFC3339)}, nil)
	next := r.URL.Query().Get("next")
	if next == "" || !strings.HasPrefix(next, "/") {
		next = "/"
	}
	http.Redirect(w, r, next, http.StatusSeeOther)
}

func (a *App) logout(w http.ResponseWriter, r *http.Request) {
	if !a.verifyCSRF(r) {
		http.Error(w, "Sessão inválida.", http.StatusForbidden)
		return
	}
	if c, err := r.Cookie("vv_session"); err == nil {
		_ = a.sb.Delete(r.Context(), "sessions", eq("token_hash", sha256Hex(c.Value)))
	}
	clearSessionCookie(w)
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func (a *App) inviteGet(w http.ResponseWriter, r *http.Request) {
	token := r.PathValue("token")
	inv, err := a.findInvite(r, token)
	if err != nil {
		a.render(w, r, "invite", ViewData{Title: "Convite", Error: err.Error()})
		return
	}
	a.render(w, r, "invite", ViewData{Title: "Criar acesso", Data: map[string]any{"Invite": inv, "Token": token}})
}

func (a *App) invitePost(w http.ResponseWriter, r *http.Request) {
	token := r.PathValue("token")
	inv, err := a.findInvite(r, token)
	if err != nil {
		a.render(w, r, "invite", ViewData{Title: "Convite", Error: err.Error()})
		return
	}
	username := strings.ToLower(strings.TrimSpace(r.FormValue("username")))
	name := strings.TrimSpace(r.FormValue("name"))
	if name == "" {
		name = inv.Name
	}
	password := r.FormValue("password")
	confirm := r.FormValue("confirm")
	if username == "" || strings.Contains(username, " ") {
		a.render(w, r, "invite", ViewData{Title: "Criar acesso", Error: "Use um nome de usuário sem espaços.", Data: map[string]any{"Invite": inv, "Token": token}})
		return
	}
	if password != confirm {
		a.render(w, r, "invite", ViewData{Title: "Criar acesso", Error: "As senhas não coincidem.", Data: map[string]any{"Invite": inv, "Token": token}})
		return
	}
	hash, err := HashPassword(password)
	if err != nil {
		a.render(w, r, "invite", ViewData{Title: "Criar acesso", Error: err.Error(), Data: map[string]any{"Invite": inv, "Token": token}})
		return
	}
	var existing []User
	_ = a.sb.Select(r.Context(), "users", eq("username", username)+"&limit=1", &existing)
	if len(existing) > 0 {
		a.render(w, r, "invite", ViewData{Title: "Criar acesso", Error: "Esse usuário já existe.", Data: map[string]any{"Invite": inv, "Token": token}})
		return
	}
	var out []User
	if err := a.sb.Insert(r.Context(), "users", User{Name: name, Username: username, Email: inv.Email, PasswordHash: hash, Role: "viewer", Active: true}, &out); err != nil || len(out) == 0 {
		a.render(w, r, "invite", ViewData{Title: "Criar acesso", Error: "Não foi possível criar o usuário.", Data: map[string]any{"Invite": inv, "Token": token}})
		return
	}
	_ = a.sb.Update(r.Context(), "invites", eq("id", inv.ID), map[string]any{"used_at": time.Now().UTC().Format(time.RFC3339)}, nil)
	raw, _, _ := a.createSession(r.Context(), out[0].ID)
	a.setSessionCookie(w, r, raw)
	http.Redirect(w, r, "/?ok="+url.QueryEscape("Acesso criado com sucesso."), http.StatusSeeOther)
}

func (a *App) findInvite(r *http.Request, token string) (Invite, error) {
	if a.sb == nil {
		return Invite{}, fmt.Errorf("serviço de dados não configurado")
	}
	var rows []Invite
	if err := a.sb.Select(r.Context(), "invites", eq("token_hash", sha256Hex(token))+"&limit=1", &rows); err != nil || len(rows) == 0 {
		return Invite{}, fmt.Errorf("convite inválido ou inexistente")
	}
	inv := rows[0]
	if inv.UsedAt != "" {
		return Invite{}, fmt.Errorf("este convite já foi utilizado")
	}
	exp, err := time.Parse(time.RFC3339, inv.ExpiresAt)
	if err != nil || time.Now().UTC().After(exp) {
		return Invite{}, fmt.Errorf("este convite expirou")
	}
	return inv, nil
}
