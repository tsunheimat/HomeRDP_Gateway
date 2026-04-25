package dashboard

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

var ErrIconNotFound = fmt.Errorf("icon not found")

type Icon struct {
	ID           string    `json:"id"`
	Filename     string    `json:"filename"`
	OriginalName string    `json:"originalName"`
	ContentType  string    `json:"contentType"`
	Active       bool      `json:"active"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

type IconStore interface {
	ListIcons() ([]Icon, error)
	SaveIcon(originalName string, data []byte) (Icon, error)
	SetActiveIcon(id string) error
	DeleteIcon(id string) error
	ActiveIcon() (Icon, error)
	ResolveIcon(filename string) string
}

type FileIconStore struct {
	metadataPath string
	iconDir      string
	options      FileIconStoreOptions
	mutex        sync.Mutex
}

type FileIconStoreOptions struct {
	MaxIcons        int
	MaxStorageBytes int64
}

func NewFileIconStore(storePath, iconDir string, options ...FileIconStoreOptions) (*FileIconStore, error) {
	if strings.TrimSpace(iconDir) == "" {
		if strings.TrimSpace(storePath) == "" {
			iconDir = "icons"
		} else {
			iconDir = filepath.Join(storePath, "icons")
		}
	}
	if err := os.MkdirAll(storePath, 0o755); err != nil {
		return nil, fmt.Errorf("create icon metadata path: %w", err)
	}
	if err := os.MkdirAll(iconDir, 0o755); err != nil {
		return nil, fmt.Errorf("create icon directory: %w", err)
	}
	opts := FileIconStoreOptions{}
	if len(options) > 0 {
		opts = options[0]
	}
	return &FileIconStore{
		metadataPath: filepath.Join(storePath, "icons.json"),
		iconDir:      iconDir,
		options:      opts,
	}, nil
}

func (s *FileIconStore) ListIcons() ([]Icon, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	icons, err := s.readLocked()
	if err != nil {
		return nil, err
	}
	out := make([]Icon, len(icons))
	copy(out, icons)
	return out, nil
}

func (s *FileIconStore) SaveIcon(originalName string, data []byte) (Icon, error) {
	ext, contentType, err := validateIconName(originalName)
	if err != nil {
		return Icon{}, err
	}
	if len(data) == 0 {
		return Icon{}, validationError("icon upload is empty")
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	icons, err := s.readLocked()
	if err != nil {
		return Icon{}, err
	}
	if err := enforceFileQuota(s.iconDir, s.options.MaxIcons, s.options.MaxStorageBytes, int64(len(data)), "icon upload"); err != nil {
		return Icon{}, err
	}

	id, err := newIconID()
	if err != nil {
		return Icon{}, err
	}
	filename := id + ext
	fullPath := filepath.Join(s.iconDir, filename)
	if err := os.WriteFile(fullPath, data, 0o600); err != nil {
		return Icon{}, fmt.Errorf("write icon file: %w", err)
	}

	now := time.Now().UTC()
	icon := Icon{
		ID:           id,
		Filename:     filename,
		OriginalName: filepath.Base(strings.TrimSpace(originalName)),
		ContentType:  contentType,
		Active:       len(icons) == 0,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	icons = append(icons, icon)
	if err := s.writeLocked(icons); err != nil {
		_ = os.Remove(fullPath)
		return Icon{}, err
	}

	return icon, nil
}

func (s *FileIconStore) SetActiveIcon(id string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return validationError("icon id is required")
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	icons, err := s.readLocked()
	if err != nil {
		return err
	}
	found := false
	now := time.Now().UTC()
	for i := range icons {
		active := icons[i].ID == id
		if active {
			found = true
		}
		if icons[i].Active != active {
			icons[i].Active = active
			icons[i].UpdatedAt = now
		}
	}
	if !found {
		return ErrIconNotFound
	}
	return s.writeLocked(icons)
}

func (s *FileIconStore) DeleteIcon(id string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return validationError("icon id is required")
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	icons, err := s.readLocked()
	if err != nil {
		return err
	}
	index := -1
	for i := range icons {
		if icons[i].ID == id {
			index = i
			break
		}
	}
	if index == -1 {
		return ErrIconNotFound
	}

	deleted := icons[index]
	icons = append(icons[:index], icons[index+1:]...)
	if deleted.Active && len(icons) > 0 {
		icons[0].Active = true
		icons[0].UpdatedAt = time.Now().UTC()
	}
	if err := s.writeLocked(icons); err != nil {
		return err
	}
	_ = removeUploadFile(s.ResolveIcon(deleted.Filename))
	return nil
}

func (s *FileIconStore) ActiveIcon() (Icon, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	icons, err := s.readLocked()
	if err != nil {
		return Icon{}, err
	}
	for _, icon := range icons {
		if icon.Active {
			return icon, nil
		}
	}
	return Icon{}, ErrIconNotFound
}

func (s *FileIconStore) ResolveIcon(filename string) string {
	return filepath.Join(s.iconDir, filepath.Base(filename))
}

func (s *FileIconStore) readLocked() ([]Icon, error) {
	data, err := os.ReadFile(s.metadataPath)
	if err != nil {
		if os.IsNotExist(err) {
			return []Icon{}, nil
		}
		return nil, fmt.Errorf("read icon metadata: %w", err)
	}
	if len(data) == 0 {
		return []Icon{}, nil
	}

	var icons []Icon
	if err := json.Unmarshal(data, &icons); err != nil {
		return nil, fmt.Errorf("decode icon metadata: %w", err)
	}
	if icons == nil {
		return []Icon{}, nil
	}
	return filterSafeIcons(icons), nil
}

func filterSafeIcons(icons []Icon) []Icon {
	out := make([]Icon, 0, len(icons))
	for _, icon := range icons {
		if isSafeStoredIcon(icon) {
			out = append(out, icon)
		}
	}
	return out
}

func isSafeStoredIcon(icon Icon) bool {
	_, contentType, err := validateIconName(icon.Filename)
	if err != nil {
		return false
	}
	if icon.ContentType != contentType {
		return false
	}
	return true
}

func (s *FileIconStore) writeLocked(icons []Icon) error {
	if err := os.MkdirAll(filepath.Dir(s.metadataPath), 0o755); err != nil {
		return fmt.Errorf("ensure icon metadata directory: %w", err)
	}

	data, err := json.MarshalIndent(icons, "", "  ")
	if err != nil {
		return fmt.Errorf("encode icon metadata: %w", err)
	}

	tmpFile, err := os.CreateTemp(filepath.Dir(s.metadataPath), "icons-*.tmp")
	if err != nil {
		return fmt.Errorf("create icon metadata temp file: %w", err)
	}
	tmpName := tmpFile.Name()

	if _, err := tmpFile.Write(data); err != nil {
		tmpFile.Close()
		_ = os.Remove(tmpName)
		return fmt.Errorf("write icon metadata temp file: %w", err)
	}
	if err := tmpFile.Close(); err != nil {
		_ = os.Remove(tmpName)
		return fmt.Errorf("close icon metadata temp file: %w", err)
	}

	if err := os.Rename(tmpName, s.metadataPath); err != nil {
		_ = os.Remove(tmpName)
		return fmt.Errorf("replace icon metadata file: %w", err)
	}
	return nil
}

func validateIconName(filename string) (string, string, error) {
	ext := strings.ToLower(filepath.Ext(strings.TrimSpace(filename)))
	switch ext {
	case ".ico", ".icon":
		return ext, "image/x-icon", nil
	case ".png":
		return ext, "image/png", nil
	case ".jpg", ".jpeg":
		return ext, "image/jpeg", nil
	default:
		return "", "", validationError("icon must be .ico, .icon, .png, .jpg, or .jpeg")
	}
}

func newIconID() (string, error) {
	random := make([]byte, 16)
	if _, err := rand.Read(random); err != nil {
		return "", fmt.Errorf("generate icon id: %w", err)
	}
	return hex.EncodeToString(random), nil
}
