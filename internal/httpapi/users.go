package httpapi

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/agnaldom/crosswatc/internal/store"
)

// UserStore manages users (satisfied by *store.Store).
type UserStore interface {
	ListUsers(ctx context.Context) ([]store.User, error)
	GetUser(ctx context.Context, id int64) (store.User, error)
	CreateUser(ctx context.Context, u store.User, password string) (store.User, error)
	UpdateUser(ctx context.Context, u store.User, password string) (store.User, error)
	DeleteUser(ctx context.Context, id int64) error
}

type inMemoryUserStore struct {
	mu     sync.RWMutex
	nextID int64
	users  map[int64]store.User
}

func newInMemoryUserStore() *inMemoryUserStore {
	now := time.Now().UTC()
	return &inMemoryUserStore{
		nextID: 2,
		users: map[int64]store.User{
			1: {
				ID:        1,
				Username:  "admin",
				Email:     "admin@crosswatc.local",
				Role:      "admin",
				CreatedAt: now,
				UpdatedAt: now,
			},
		},
	}
}

func (m *inMemoryUserStore) ListUsers(_ context.Context) ([]store.User, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]store.User, 0, len(m.users))
	for _, u := range m.users {
		out = append(out, u)
	}
	return out, nil
}

func (m *inMemoryUserStore) GetUser(_ context.Context, id int64) (store.User, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	u, ok := m.users[id]
	if !ok {
		return store.User{}, store.ErrNotFound
	}
	return u, nil
}

func (m *inMemoryUserStore) CreateUser(_ context.Context, u store.User, _ string) (store.User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, existing := range m.users {
		if existing.Username == u.Username || existing.Email == u.Email {
			return store.User{}, store.ErrConflict
		}
	}
	now := time.Now().UTC()
	u.ID = m.nextID
	m.nextID++
	u.CreatedAt = now
	u.UpdatedAt = now
	m.users[u.ID] = u
	return u, nil
}

func (m *inMemoryUserStore) UpdateUser(_ context.Context, u store.User, _ string) (store.User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	existing, ok := m.users[u.ID]
	if !ok {
		return store.User{}, store.ErrNotFound
	}
	for _, o := range m.users {
		if o.ID != u.ID && (o.Username == u.Username || o.Email == u.Email) {
			return store.User{}, store.ErrConflict
		}
	}
	now := time.Now().UTC()
	u.CreatedAt = existing.CreatedAt
	u.UpdatedAt = now
	m.users[u.ID] = u
	return u, nil
}

func (m *inMemoryUserStore) DeleteUser(_ context.Context, id int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.users[id]; !ok {
		return store.ErrNotFound
	}
	delete(m.users, id)
	return nil
}

func (h *handlers) listUsers(c *gin.Context) {
	users, err := h.users.ListUsers(c.Request.Context())
	if err != nil {
		problem(c, http.StatusInternalServerError, "users-list-failed", "Failed to list users", err.Error())
		return
	}
	if users == nil {
		users = []store.User{}
	}
	c.JSON(http.StatusOK, users)
}

func (h *handlers) createUser(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required"`
		Email    string `json:"email" binding:"required"`
		Password string `json:"password"`
		Role     string `json:"role"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		problem(c, http.StatusBadRequest, "invalid-user-data", "Username and email are required", err.Error())
		return
	}
	role := req.Role
	if role != "admin" && role != "user" {
		role = "user"
	}
	user := store.User{
		Username: req.Username,
		Email:    req.Email,
		Role:     role,
	}
	created, err := h.users.CreateUser(c.Request.Context(), user, req.Password)
	if err != nil {
		if errors.Is(err, store.ErrConflict) {
			problem(c, http.StatusBadRequest, "user-already-exists", "Username or email already exists", err.Error())
			return
		}
		problem(c, http.StatusInternalServerError, "user-create-failed", "Failed to create user", err.Error())
		return
	}
	c.JSON(http.StatusCreated, created)
}

func (h *handlers) updateUser(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		problem(c, http.StatusBadRequest, "invalid-user-id", "User ID must be an integer", "")
		return
	}
	var req struct {
		Username string `json:"username"`
		Email    string `json:"email"`
		Password string `json:"password"`
		Role     string `json:"role"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		problem(c, http.StatusBadRequest, "invalid-user-data", "Invalid user payload", err.Error())
		return
	}
	existing, err := h.users.GetUser(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			problem(c, http.StatusNotFound, "user-not-found", "User not found", "")
			return
		}
		problem(c, http.StatusInternalServerError, "user-get-failed", "Failed to fetch user", err.Error())
		return
	}
	if req.Username != "" {
		existing.Username = req.Username
	}
	if req.Email != "" {
		existing.Email = req.Email
	}
	if req.Role == "admin" || req.Role == "user" {
		existing.Role = req.Role
	}
	updated, err := h.users.UpdateUser(c.Request.Context(), existing, req.Password)
	if err != nil {
		if errors.Is(err, store.ErrConflict) {
			problem(c, http.StatusBadRequest, "user-already-exists", "Username or email already exists", err.Error())
			return
		}
		problem(c, http.StatusInternalServerError, "user-update-failed", "Failed to update user", err.Error())
		return
	}
	c.JSON(http.StatusOK, updated)
}

func (h *handlers) deleteUser(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		problem(c, http.StatusBadRequest, "invalid-user-id", "User ID must be an integer", "")
		return
	}
	if err := h.users.DeleteUser(c.Request.Context(), id); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			problem(c, http.StatusNotFound, "user-not-found", "User not found", "")
			return
		}
		problem(c, http.StatusInternalServerError, "user-delete-failed", "Failed to delete user", err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}
