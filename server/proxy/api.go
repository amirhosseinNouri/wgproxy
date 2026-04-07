//go:build linux

package proxy

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

var startTime time.Time

// --- JWT helpers (HS256) ---

type jwtClaims struct {
	Sub string `json:"sub"`
	Exp int64  `json:"exp"`
}

func createToken(secret []byte) (string, error) {
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))

	claims := jwtClaims{Sub: "admin", Exp: time.Now().Add(24 * time.Hour).Unix()}
	claimsJSON, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}
	payload := base64.RawURLEncoding.EncodeToString(claimsJSON)

	sigInput := header + "." + payload
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(sigInput))
	sig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))

	return sigInput + "." + sig, nil
}

func validateToken(tokenStr string, secret []byte) bool {
	parts := strings.SplitN(tokenStr, ".", 3)
	if len(parts) != 3 {
		return false
	}

	// Verify signature
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(parts[0] + "." + parts[1]))
	expectedSig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(parts[2]), []byte(expectedSig)) {
		return false
	}

	// Decode and check expiry
	claimsJSON, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return false
	}
	var claims jwtClaims
	if err := json.Unmarshal(claimsJSON, &claims); err != nil {
		return false
	}
	return time.Now().Unix() < claims.Exp
}

// --- Middleware ---

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func authMiddleware(next http.HandlerFunc, secret []byte) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if !strings.HasPrefix(auth, "Bearer ") {
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}
		token := strings.TrimPrefix(auth, "Bearer ")
		if !validateToken(token, secret) {
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}
		next(w, r)
	}
}

// --- JSON helpers ---

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

// --- Handlers ---

func handleLogin(store *Store, secret []byte) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Username string `json:"username"`
			Password string `json:"password"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}

		adminUser, adminPass, err := store.GetAdmin()
		if err != nil {
			writeError(w, http.StatusInternalServerError, "admin not configured")
			return
		}

		if req.Username != adminUser || req.Password != adminPass {
			writeError(w, http.StatusUnauthorized, "invalid credentials")
			return
		}

		token, err := createToken(secret)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to create token")
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"token":                token,
			"must_change_password": store.IsDefaultPassword(),
		})
	}
}

func handleChangePassword(store *Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Username string `json:"username"`
			Password string `json:"password"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		if req.Username == "" || req.Password == "" {
			writeError(w, http.StatusBadRequest, "username and password required")
			return
		}

		if err := store.SetAdmin(req.Username, req.Password); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to update password")
			return
		}

		writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
	}
}

func handleGetStats(store *Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		users, err := store.ListUsers()
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to list users")
			return
		}

		var totalUpload, totalDownload int64
		var activeUsers int
		for _, u := range users {
			totalUpload += u.Upload
			totalDownload += u.Download
			if u.Enabled {
				activeUsers++
			}
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"total_upload":   totalUpload,
			"total_download": totalDownload,
			"user_count":     len(users),
			"active_users":   activeUsers,
			"online_users":   len(store.OnlineUsers()),
			"uptime_seconds": int(time.Since(startTime).Seconds()),
		})
	}
}

func handleListUsers(store *Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		users, err := store.ListUsers()
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to list users")
			return
		}

		online := store.OnlineUsers()

		type userResp struct {
			Username    string `json:"username"`
			Upload      int64  `json:"upload"`
			Download    int64  `json:"download"`
			Enabled     bool   `json:"enabled"`
			CreatedAt   string `json:"created_at"`
			Online      bool   `json:"online"`
			Connections int    `json:"connections"`
		}

		resp := make([]userResp, 0, len(users))
		for _, u := range users {
			conns := online[u.Username]
			resp = append(resp, userResp{
				Username:    u.Username,
				Upload:      u.Upload,
				Download:    u.Download,
				Enabled:     u.Enabled,
				CreatedAt:   u.CreatedAt.Format(time.RFC3339),
				Online:      conns > 0,
				Connections: conns,
			})
		}

		writeJSON(w, http.StatusOK, map[string]any{"users": resp})
	}
}

func handleCreateUser(store *Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Username string `json:"username"`
			Password string `json:"password"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		if req.Username == "" || req.Password == "" {
			writeError(w, http.StatusBadRequest, "username and password required")
			return
		}

		if err := store.AddUser(req.Username, req.Password); err != nil {
			writeError(w, http.StatusConflict, err.Error())
			return
		}

		writeJSON(w, http.StatusCreated, map[string]bool{"ok": true})
	}
}

func handleUpdateUser(store *Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		username := r.PathValue("username")
		if username == "" {
			writeError(w, http.StatusBadRequest, "username required")
			return
		}

		var req struct {
			Password *string `json:"password"`
			Enabled  *bool   `json:"enabled"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}

		if err := store.UpdateUser(username, req.Password, req.Enabled); err != nil {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}

		writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
	}
}

func handleDeleteUser(store *Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		username := r.PathValue("username")
		if username == "" {
			writeError(w, http.StatusBadRequest, "username required")
			return
		}

		if err := store.DeleteUser(username); err != nil {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}

		writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
	}
}

// --- API Server ---

func runAPI(cfg *Config, store *Store) {
	startTime = time.Now()

	secret := []byte(cfg.JWTSecret)

	mux := http.NewServeMux()

	// Public
	mux.HandleFunc("POST /api/login", handleLogin(store, secret))

	// Protected
	mux.HandleFunc("POST /api/admin/password", authMiddleware(handleChangePassword(store), secret))
	mux.HandleFunc("GET /api/stats", authMiddleware(handleGetStats(store), secret))
	mux.HandleFunc("GET /api/users", authMiddleware(handleListUsers(store), secret))
	mux.HandleFunc("POST /api/users", authMiddleware(handleCreateUser(store), secret))
	mux.HandleFunc("PUT /api/users/{username}", authMiddleware(handleUpdateUser(store), secret))
	mux.HandleFunc("DELETE /api/users/{username}", authMiddleware(handleDeleteUser(store), secret))

	addr := ":" + cfg.APIPort
	slog.Info("API server started", "port", cfg.APIPort)

	if err := http.ListenAndServe(addr, corsMiddleware(mux)); err != nil {
		slog.Error("API server failed", "err", err)
	}
}
