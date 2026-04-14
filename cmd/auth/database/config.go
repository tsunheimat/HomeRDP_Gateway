package database

import (
	"log"
	"os"
	"sync"
	"time"

	"github.com/bolkedebruin/rdpgw/cmd/auth/config"
)

type Config struct {
	mutex      sync.RWMutex
	users      map[string]config.UserConfig
	configFile string
	modTime    time.Time
}

func NewConfig(users []config.UserConfig) *Config {
	return &Config{
		users: usersToMap(users),
	}
}

func NewConfigFile(configFile string) (*Config, error) {
	conf, modTime, err := loadConfigFile(configFile)
	if err != nil {
		return nil, err
	}
	return &Config{
		users:      usersToMap(conf.Users),
		configFile: configFile,
		modTime:    modTime,
	}, nil
}

func (c *Config) GetPassword(username string) string {
	c.reloadIfChanged()

	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.users[username].Password
}

func (c *Config) reloadIfChanged() {
	if c.configFile == "" {
		return
	}

	info, err := os.Stat(c.configFile)
	if err != nil {
		if !os.IsNotExist(err) {
			log.Printf("Cannot stat auth config %s due to: %s", c.configFile, err)
		}
		return
	}

	c.mutex.RLock()
	unchanged := !info.ModTime().After(c.modTime)
	c.mutex.RUnlock()
	if unchanged {
		return
	}

	conf, modTime, err := loadConfigFile(c.configFile)
	if err != nil {
		log.Printf("Cannot reload auth config %s due to: %s", c.configFile, err)
		return
	}

	c.mutex.Lock()
	defer c.mutex.Unlock()
	if !modTime.After(c.modTime) {
		return
	}
	c.users = usersToMap(conf.Users)
	c.modTime = modTime
}

func loadConfigFile(configFile string) (config.Configuration, time.Time, error) {
	info, err := os.Stat(configFile)
	if err != nil {
		if os.IsNotExist(err) {
			return config.Configuration{}, time.Time{}, nil
		}
		return config.Configuration{}, time.Time{}, err
	}
	conf, err := config.LoadFile(configFile)
	if err != nil {
		return config.Configuration{}, time.Time{}, err
	}
	return conf, info.ModTime(), nil
}

func usersToMap(users []config.UserConfig) map[string]config.UserConfig {
	usersMap := map[string]config.UserConfig{}
	for _, user := range users {
		usersMap[user.Username] = user
	}
	return usersMap
}

var _ Database = (*Config)(nil)
