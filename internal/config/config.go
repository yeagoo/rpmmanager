package config

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
	"golang.org/x/crypto/bcrypt"
)

type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Auth     AuthConfig     `mapstructure:"auth"`
	Database DatabaseConfig `mapstructure:"database"`
	Storage  StorageConfig  `mapstructure:"storage"`
	GPG      GPGConfig      `mapstructure:"gpg"`
	Tools    ToolsConfig    `mapstructure:"tools"`
	Monitor  MonitorConfig  `mapstructure:"monitor"`
	Log      LogConfig      `mapstructure:"log"`
}

type ServerConfig struct {
	Listen  string `mapstructure:"listen"`
	BaseURL string `mapstructure:"base_url"`
}

type AuthConfig struct {
	Username     string `mapstructure:"username"`
	PasswordHash string `mapstructure:"password_hash"`
	APIToken     string `mapstructure:"api_token"`
	JWTSecret    string `mapstructure:"jwt_secret"`
}

type DatabaseConfig struct {
	Path string `mapstructure:"path"`
}

type StorageConfig struct {
	RepoRoot  string `mapstructure:"repo_root"`
	BuildLogs string `mapstructure:"build_logs"`
	TempDir   string `mapstructure:"temp_dir"`
}

type GPGConfig struct {
	HomeDir string `mapstructure:"home_dir"`
}

type ToolsConfig struct {
	NfpmPath       string `mapstructure:"nfpm_path"`
	CreaterepoPath string `mapstructure:"createrepo_path"`
	GPGPath        string `mapstructure:"gpg_path"`
	RPMPath        string `mapstructure:"rpm_path"`
	RPMLintPath    string `mapstructure:"rpmlint_path"`
}

type MonitorConfig struct {
	Enabled         bool   `mapstructure:"enabled"`
	DefaultInterval string `mapstructure:"default_interval"`
	GithubToken     string `mapstructure:"github_token"`
}

type LogConfig struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"`
}

func Load(cfgFile string) (*Config, error) {
	v := viper.New()

	// Defaults
	v.SetDefault("server.listen", "0.0.0.0:8080")
	v.SetDefault("server.base_url", "http://localhost:8080")
	v.SetDefault("auth.username", "admin")
	v.SetDefault("database.path", "./data/rpmmanager.db")
	v.SetDefault("storage.repo_root", "./data/repos")
	v.SetDefault("storage.build_logs", "./data/logs")
	v.SetDefault("storage.temp_dir", "./data/tmp")
	v.SetDefault("gpg.home_dir", "./data/gnupg")
	v.SetDefault("tools.nfpm_path", "nfpm")
	v.SetDefault("tools.createrepo_path", "createrepo_c")
	v.SetDefault("tools.gpg_path", "gpg")
	v.SetDefault("tools.rpm_path", "rpm")
	v.SetDefault("tools.rpmlint_path", "rpmlint")
	v.SetDefault("monitor.enabled", true)
	v.SetDefault("monitor.default_interval", "6h")
	v.SetDefault("log.level", "info")
	v.SetDefault("log.format", "text")

	if cfgFile != "" {
		v.SetConfigFile(cfgFile)
	} else {
		v.SetConfigName("config")
		v.SetConfigType("yaml")
		v.AddConfigPath(".")
		v.AddConfigPath("/etc/rpmmanager")
	}

	v.SetEnvPrefix("RPMMANAGER")
	v.AutomaticEnv()

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("read config: %w", err)
		}
		// Config file not found is OK - use defaults
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}

	if err := cfg.ensureDefaults(); err != nil {
		return nil, err
	}

	if err := cfg.ensureDirectories(); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func (c *Config) ensureDefaults() error {
	// Generate JWT secret if empty
	if c.Auth.JWTSecret == "" {
		secret, err := randomHex(32)
		if err != nil {
			return fmt.Errorf("generate jwt secret: %w", err)
		}
		c.Auth.JWTSecret = secret
	}

	// Generate password and hash on first run
	if c.Auth.PasswordHash == "" {
		password, err := randomHex(16)
		if err != nil {
			return fmt.Errorf("generate password: %w", err)
		}
		hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			return fmt.Errorf("hash password: %w", err)
		}
		c.Auth.PasswordHash = string(hash)
		fmt.Fprintf(os.Stderr, "\n========================================\n")
		fmt.Fprintf(os.Stderr, "  Generated admin password: %s\n", password)
		fmt.Fprintf(os.Stderr, "  Username: %s\n", c.Auth.Username)
		fmt.Fprintf(os.Stderr, "  Please save this password!\n")
		fmt.Fprintf(os.Stderr, "========================================\n\n")
	}

	// Generate API token if empty
	if c.Auth.APIToken == "" {
		token, err := randomHex(32)
		if err != nil {
			return fmt.Errorf("generate api token: %w", err)
		}
		c.Auth.APIToken = token
	}

	return nil
}

func (c *Config) ensureDirectories() error {
	dirs := []string{
		filepath.Dir(c.Database.Path),
		c.Storage.RepoRoot,
		c.Storage.BuildLogs,
		c.Storage.TempDir,
	}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("create directory %s: %w", dir, err)
		}
	}
	// GPG home needs restricted permissions - create parent first, then leaf with 0700
	gpgParent := filepath.Dir(c.GPG.HomeDir)
	if err := os.MkdirAll(gpgParent, 0755); err != nil {
		return fmt.Errorf("create gpg parent dir: %w", err)
	}
	if err := os.MkdirAll(c.GPG.HomeDir, 0700); err != nil {
		return fmt.Errorf("create gpg home dir: %w", err)
	}
	// Ensure permissions are correct even if directory already existed
	if err := os.Chmod(c.GPG.HomeDir, 0700); err != nil {
		return fmt.Errorf("chmod gpg home: %w", err)
	}
	return nil
}

func randomHex(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
