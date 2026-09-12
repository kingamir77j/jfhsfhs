// Package config provides configuration management utilities for the 3x-ui panel,
// including version information, logging levels, database paths, and environment variable handling.
package config

import (
	_ "embed"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

//go:embed version
var version string

//go:embed name
var name string

var (
	buildCommit string
	buildDate   string
)

type LogLevel string

const (
	Debug   LogLevel = "debug"
	Info    LogLevel = "info"
	Notice  LogLevel = "notice"
	Warning LogLevel = "warning"
	Error   LogLevel = "error"
)

func GetBaseVersion() string { return strings.TrimSpace(version) }
func GetName() string        { return strings.TrimSpace(name) }
func GetBuildCommit() string { return strings.TrimSpace(buildCommit) }
func GetBuildDate() string   { return strings.TrimSpace(buildDate) }
func IsDevBuild() bool       { return GetBuildCommit() != "" }

func GetPanelVersion() string {
	if !IsDevBuild() {
		return GetBaseVersion()
	}
	commit := GetBuildCommit()
	if len(commit) > 8 {
		commit = commit[:8]
	}
	return "dev+" + commit
}

func GetLogLevel() LogLevel {
	if IsDebug() {
		return Debug
	}
	logLevel := os.Getenv("XUI_LOG_LEVEL")
	if logLevel == "" {
		return Info
	}
	return LogLevel(logLevel)
}

func IsDebug() bool    { return os.Getenv("XUI_DEBUG") == "true" }
func IsSkipHSTS() bool { return os.Getenv("XUI_SKIP_HSTS") == "true" }

// GetPortOverride returns the panel port. XUI_PORT has priority; when it is
// unset, Railway's PORT variable is used. If neither is set, the application
// keeps its normal built-in port.
func GetPortOverride() (port int, configured bool, err error) {
	value, ok := os.LookupEnv("XUI_PORT")
	key := "XUI_PORT"
	if !ok || strings.TrimSpace(value) == "" {
		value, ok = os.LookupEnv("PORT")
		key = "PORT"
	}
	if !ok || strings.TrimSpace(value) == "" {
		return 0, false, nil
	}

	port, err = strconv.Atoi(strings.TrimSpace(value))
	if err != nil {
		return 0, true, fmt.Errorf("parse %s: %w", key, err)
	}
	if port < 1 || port > 65535 {
		return 0, true, fmt.Errorf("%s must be between 1 and 65535", key)
	}
	return port, true, nil
}

func GetBinFolderPath() string {
	binFolderPath := os.Getenv("XUI_BIN_FOLDER")
	if binFolderPath == "" {
		binFolderPath = "bin"
	}
	return binFolderPath
}

func getBaseDir() string {
	exePath, err := os.Executable()
	if err != nil {
		return "."
	}
	exeDir := filepath.Dir(exePath)
	exeDirLower := strings.ToLower(filepath.ToSlash(exeDir))
	if strings.Contains(exeDirLower, "/appdata/local/temp/") || strings.Contains(exeDirLower, "/go-build") {
		wd, err := os.Getwd()
		if err != nil {
			return "."
		}
		return wd
	}
	return exeDir
}

func GetDBFolderPath() string {
	dbFolderPath := os.Getenv("XUI_DB_FOLDER")
	if dbFolderPath != "" {
		return dbFolderPath
	}
	if runtime.GOOS == "windows" {
		return getBaseDir()
	}
	return "/etc/x-ui"
}

func GetDBPath() string { return fmt.Sprintf("%s/%s.db", GetDBFolderPath(), GetName()) }

func GetUpdateStatusFilePath() string {
	return filepath.Join(GetDBFolderPath(), "update-status.json")
}

func GetDBKind() string {
	v := strings.ToLower(strings.TrimSpace(os.Getenv("XUI_DB_TYPE")))
	switch v {
	case "postgres", "postgresql", "pg":
		return "postgres"
	default:
		return "sqlite"
	}
}

func GetDBDSN() string { return strings.TrimSpace(os.Getenv("XUI_DB_DSN")) }
func GetNodeTokenEncryptionMode() string { return strings.TrimSpace(os.Getenv("NODE_TOKEN_ENCRYPTION")) }

func GetNodeTokenKeyFile() string {
	if p := strings.TrimSpace(os.Getenv("XUI_NODE_TOKEN_KEY_FILE")); p != "" {
		return p
	}
	return "/etc/x-ui/node_token_key.json"
}

func GetNodeTokenKeyEnv() string { return "XUI_NODE_TOKEN_KEY" }

func GetEnvFilePaths() []string {
	if runtime.GOOS == "windows" {
		return nil
	}
	return []string{"/etc/default/x-ui", "/etc/conf.d/x-ui", "/etc/sysconfig/x-ui"}
}

func GetLogFolder() string {
	logFolderPath := os.Getenv("XUI_LOG_FOLDER")
	if logFolderPath != "" {
		return logFolderPath
	}
	if testing.Testing() {
		return filepath.Join(os.TempDir(), "3x-ui-test-log")
	}
	if runtime.GOOS == "windows" {
		return filepath.Join(".", "log")
	}
	return "/var/log/x-ui"
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	if err != nil {
		return err
	}
	return out.Sync()
}

func init() {
	if runtime.GOOS != "windows" || os.Getenv("XUI_DB_FOLDER") != "" {
		return
	}
	oldDBPath := fmt.Sprintf("/etc/x-ui/%s.db", GetName())
	newDBPath := fmt.Sprintf("%s/%s.db", GetDBFolderPath(), GetName())
	if _, err := os.Stat(newDBPath); err == nil {
		return
	}
	if _, err := os.Stat(oldDBPath); os.IsNotExist(err) {
		return
	}
	_ = copyFile(oldDBPath, newDBPath)
}
