package config

import (
	"errors"
	"fmt"
	"os"
	"strings"

	sml "codeberg.org/crowdware/sml-go"
)

type Config struct {
	Name          string
	BaseURL       string
	DBPath        string
	Port          string
	SessionSecret string
	AdminEmail    string
	SMTP          SMTPConfig
}

type SMTPConfig struct {
	Host string
	Port string
	User string
	Pass string
	From string
}

const (
	defaultPort   = "8090"
	defaultDBPath = "./data/crowdbook.db"
)

func LoadConfig(path string) (*Config, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	source := strings.TrimSpace(string(raw))
	if source == "" {
		return nil, errors.New("config file is empty")
	}

	doc, err := sml.ParseDocument(source)
	if err != nil {
		return nil, fmt.Errorf("parse SML config: %w", err)
	}

	appNode := findNodeByName(doc.Roots, "App")
	if appNode == nil {
		return nil, errors.New(`missing root node "App"`)
	}

	cfg := &Config{
		Port:   defaultPort,
		DBPath: defaultDBPath,
	}
	cfg.Name = appNode.GetValue("name", "")
	cfg.BaseURL = appNode.GetValue("base_url", "")
	cfg.DBPath = appNode.GetValue("db", cfg.DBPath)
	cfg.Port = appNode.GetValue("port", cfg.Port)
	cfg.SessionSecret = appNode.GetValue("session_secret", "")
	cfg.AdminEmail = appNode.GetValue("admin_email", "")

	smtpNode := findNodeByName(appNode.Children, "SMTP")
	if smtpNode != nil {
		cfg.SMTP = SMTPConfig{
			Host: smtpNode.GetValue("host", ""),
			Port: smtpNode.GetValue("port", ""),
			User: smtpNode.GetValue("user", ""),
			Pass: smtpNode.GetValue("pass", ""),
			From: smtpNode.GetValue("from", ""),
		}
	}

	return cfg, nil
}

func findNodeByName(nodes []sml.Node, name string) *sml.Node {
	for i := range nodes {
		if nodes[i].Name == name {
			return &nodes[i]
		}
	}

	return nil
}
