package dashboard

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
)

var ErrAuthUserNotFound = fmt.Errorf("auth user not found")

type AuthUser struct {
	Username  string    `json:"username"`
	Password  string    `json:"password"`
	Enabled   bool      `json:"enabled"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type AuthUserStore interface {
	List() ([]AuthUser, error)
	Get(username string) (AuthUser, error)
	Put(user AuthUser) error
	Delete(username string) error
}

type FileAuthUserStore struct {
	path  string
	mutex sync.Mutex
}

type helperConfig struct {
	Users []helperUser `yaml:"Users"`
}

type helperUser struct {
	Username string `yaml:"Username"`
	Password string `yaml:"Password"`
}

func NewFileAuthUserStore(path string) (*FileAuthUserStore, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("create auth user store directory: %w", err)
	}
	return &FileAuthUserStore{path: path}, nil
}

func (s *FileAuthUserStore) List() ([]AuthUser, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	users, err := s.readLocked()
	if err != nil {
		return nil, err
	}
	out := make([]AuthUser, len(users))
	copy(out, users)
	return out, nil
}

func (s *FileAuthUserStore) Get(username string) (AuthUser, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	users, err := s.readLocked()
	if err != nil {
		return AuthUser{}, err
	}
	for _, user := range users {
		if user.Username == username {
			return user, nil
		}
	}
	return AuthUser{}, ErrAuthUserNotFound
}

func (s *FileAuthUserStore) Put(user AuthUser) error {
	if err := validateAuthUser(user); err != nil {
		return err
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	users, err := s.readLocked()
	if err != nil {
		return err
	}

	now := time.Now().UTC()
	updated := false
	for i := range users {
		if users[i].Username != user.Username {
			continue
		}
		user.CreatedAt = users[i].CreatedAt
		user.UpdatedAt = now
		users[i] = user
		updated = true
		break
	}
	if !updated {
		if user.CreatedAt.IsZero() {
			user.CreatedAt = now
		}
		user.UpdatedAt = now
		users = append(users, user)
	}

	return writeJSONAtomic(s.path, users)
}

func (s *FileAuthUserStore) Delete(username string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	users, err := s.readLocked()
	if err != nil {
		return err
	}

	index := -1
	for i := range users {
		if users[i].Username == username {
			index = i
			break
		}
	}
	if index == -1 {
		return nil
	}

	users = append(users[:index], users[index+1:]...)
	return writeJSONAtomic(s.path, users)
}

func (s *FileAuthUserStore) readLocked() ([]AuthUser, error) {
	var users []AuthUser
	if err := readJSONFile(s.path, &users); err != nil {
		return nil, err
	}
	if users == nil {
		return []AuthUser{}, nil
	}
	return users, nil
}

func WriteAuthHelperConfig(path string, users []AuthUser) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("ensure auth helper config directory: %w", err)
	}

	cfg := helperConfig{
		Users: make([]helperUser, 0, len(users)),
	}
	for _, user := range users {
		if !user.Enabled {
			continue
		}
		cfg.Users = append(cfg.Users, helperUser{
			Username: user.Username,
			Password: user.Password,
		})
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("encode auth helper config: %w", err)
	}
	return writeFileAtomic(path, data, 0o600, "auth-helper-*.tmp")
}

func validateAuthUser(user AuthUser) error {
	if user.Username == "" {
		return validationError("auth user username is required")
	}
	if user.Password == "" {
		return validationError("auth user password is required")
	}
	return nil
}
