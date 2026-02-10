package api

import (
	"crypto/rand"
	"encoding/hex"
	"html/template"
	"log/slog"
	"net/http"
	"time"

	"github.com/cevrimxe/go-mini-rmm/internal/server/db"
	"github.com/cevrimxe/go-mini-rmm/web"
	"golang.org/x/crypto/bcrypt"
)

type AuthHandler struct {
	store     *db.Store
	loginTmpl *template.Template
	setupTmpl *template.Template
}

func NewAuthHandler(store *db.Store) *AuthHandler {
	loginTmpl := template.Must(template.ParseFS(web.TemplateFS, "templates/login.html"))
	setupTmpl := template.Must(template.ParseFS(web.TemplateFS, "templates/setup.html"))
	return &AuthHandler{store: store, loginTmpl: loginTmpl, setupTmpl: setupTmpl}
}

func (h *AuthHandler) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hasUsers, _ := h.store.HasUsers()
		if !hasUsers {
			http.Redirect(w, r, "/setup", http.StatusFound)
			return
		}

		cookie, err := r.Cookie("session")
		if err != nil || cookie.Value == "" {
			http.Redirect(w, r, "/login", http.StatusFound)
			return
		}

		user, err := h.store.GetUserBySession(cookie.Value)
		if err != nil || user == nil {
			http.Redirect(w, r, "/login", http.StatusFound)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (h *AuthHandler) SetupPage(w http.ResponseWriter, r *http.Request) {
	hasUsers, _ := h.store.HasUsers()
	if hasUsers {
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}
	h.setupTmpl.Execute(w, nil)
}

func (h *AuthHandler) Setup(w http.ResponseWriter, r *http.Request) {
	hasUsers, _ := h.store.HasUsers()
	if hasUsers {
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}

	r.ParseForm()
	username := r.FormValue("username")
	password := r.FormValue("password")
	confirm := r.FormValue("confirm")

	if username == "" || password == "" {
		h.setupTmpl.Execute(w, map[string]string{"Error": "Tüm alanlar zorunludur"})
		return
	}
	if len(password) < 6 {
		h.setupTmpl.Execute(w, map[string]string{"Error": "Şifre en az 6 karakter olmalı"})
		return
	}
	if password != confirm {
		h.setupTmpl.Execute(w, map[string]string{"Error": "Şifreler eşleşmiyor"})
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		slog.Error("bcrypt hash error", "error", err)
		h.setupTmpl.Execute(w, map[string]string{"Error": "Sunucu hatası"})
		return
	}

	if err := h.store.CreateUser(username, string(hash)); err != nil {
		slog.Error("create user error", "error", err)
		h.setupTmpl.Execute(w, map[string]string{"Error": "Kullanıcı oluşturulamadı"})
		return
	}

	slog.Info("first user created", "username", username)
	http.Redirect(w, r, "/login", http.StatusFound)
}

func (h *AuthHandler) LoginPage(w http.ResponseWriter, r *http.Request) {
	hasUsers, _ := h.store.HasUsers()
	if !hasUsers {
		http.Redirect(w, r, "/setup", http.StatusFound)
		return
	}

	if h.isLoggedIn(r) {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	h.loginTmpl.Execute(w, nil)
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	username := r.FormValue("username")
	password := r.FormValue("password")

	user, err := h.store.GetUserByUsername(username)
	if err != nil || user == nil {
		h.loginTmpl.Execute(w, map[string]string{"Error": "Geçersiz kullanıcı adı veya şifre"})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		h.loginTmpl.Execute(w, map[string]string{"Error": "Geçersiz kullanıcı adı veya şifre"})
		return
	}

	token, err := generateToken()
	if err != nil {
		slog.Error("generate token error", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	expires := time.Now().UTC().Add(7 * 24 * time.Hour)
	if err := h.store.CreateSession(token, user.ID, expires); err != nil {
		slog.Error("create session error", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "session",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		MaxAge:   86400 * 7,
	})

	slog.Info("user logged in", "username", username)
	http.Redirect(w, r, "/", http.StatusFound)
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("session")
	if err == nil {
		h.store.DeleteSession(cookie.Value)
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "session",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
	})

	http.Redirect(w, r, "/login", http.StatusFound)
}

func (h *AuthHandler) isLoggedIn(r *http.Request) bool {
	cookie, err := r.Cookie("session")
	if err != nil || cookie.Value == "" {
		return false
	}
	user, err := h.store.GetUserBySession(cookie.Value)
	return err == nil && user != nil
}

func generateToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
