package main

import (
	"encoding/json"
	"net/http"
	"strings"

	"unifiedsubscriptionproxy/internal/platform/domain"
	"unifiedsubscriptionproxy/internal/platform/service"
)

const sessionCookieName = "usp_session"

type appHandler func(http.ResponseWriter, *http.Request, domain.User)

func sessionTokenFromRequest(r *http.Request) string {
	if cookie, err := r.Cookie(sessionCookieName); err == nil {
		return strings.TrimSpace(cookie.Value)
	}
	auth := strings.TrimSpace(r.Header.Get("Authorization"))
	if strings.HasPrefix(strings.ToLower(auth), "bearer ") {
		return strings.TrimSpace(auth[7:])
	}
	return ""
}

func setSessionCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}

func clearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
		SameSite: http.SameSiteLaxMode,
	})
}

func requireRole(svc *service.Service, role string, next appHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		enableCORS(w)
		if r.Method == http.MethodOptions {
			return
		}
		token := sessionTokenFromRequest(r)
		if token == "" {
			writeError(w, http.StatusUnauthorized, errString("missing session"))
			return
		}
		user, _, err := svc.SessionUser(token)
		if err != nil {
			writeError(w, http.StatusUnauthorized, err)
			return
		}
		if role != "" && user.Role != role {
			writeError(w, http.StatusForbidden, errString("insufficient role"))
			return
		}
		next(w, r, user)
	}
}

func loginHandler(svc *service.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		enableCORS(w)
		if r.Method != http.MethodPost {
			http.NotFound(w, r)
			return
		}
		var req struct {
			Email    string `json:"email"`
			Password string `json:"password"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		user, session, err := svc.AuthenticateUser(req.Email, req.Password)
		if err != nil {
			writeError(w, http.StatusUnauthorized, err)
			return
		}
		setSessionCookie(w, session.Token)
		writeJSON(w, http.StatusOK, map[string]any{"user": safeUser(user), "session": session})
	}
}

func logoutHandler(svc *service.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		enableCORS(w)
		if r.Method != http.MethodPost {
			http.NotFound(w, r)
			return
		}
		token := sessionTokenFromRequest(r)
		if token != "" {
			_ = svc.RevokeSession(token)
		}
		clearSessionCookie(w)
		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
	}
}

func meHandler(svc *service.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		enableCORS(w)
		if r.Method != http.MethodGet {
			http.NotFound(w, r)
			return
		}
		token := sessionTokenFromRequest(r)
		if token == "" {
			writeError(w, http.StatusUnauthorized, errString("missing session"))
			return
		}
		user, session, err := svc.SessionUser(token)
		if err != nil {
			writeError(w, http.StatusUnauthorized, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"user": safeUser(user), "session": session})
	}
}

func safeUser(user domain.User) domain.User {
	user.PasswordHash = ""
	return user
}
