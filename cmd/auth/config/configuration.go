package config

import (
	"fmt"
	"log"
	"os"

	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/confmap"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
)

type Configuration struct {
	Users []UserConfig `koanf:"users"`
}

type UserConfig struct {
	Username string `koanf:"username"`
	Password string `koanf:"password"`
}

func Load(configFile string) Configuration {
	conf, err := LoadFile(configFile)
	if err != nil {
		log.Fatalf("Error loading config from file: %v", err)
	}
	return conf
}

func LoadFile(configFile string) (Configuration, error) {
	conf := Configuration{}

	var k = koanf.New(".")

	k.Load(confmap.Provider(map[string]interface{}{}, "."), nil)

	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		log.Printf("Config file %s not found, skipping config file", configFile)
	} else {
		if err := k.Load(file.Provider(configFile), yaml.Parser()); err != nil {
			return conf, fmt.Errorf("load auth config %s: %w", configFile, err)
		}
	}

	koanfTag := koanf.UnmarshalConf{Tag: "koanf"}
	if err := k.UnmarshalWithConf("Users", &conf.Users, koanfTag); err != nil {
		return conf, fmt.Errorf("unmarshal auth users: %w", err)
	}

	return conf, nil

}
