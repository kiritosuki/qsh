package config

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"

	. "github.com/kiritosuki/qsh/types"
	"go.yaml.in/yaml/v3"
)

const (
	ConfigFilePath       = ".qsh/config.yaml"
	BackupConfigFilePath = ".shell-ai/.backup-config.yaml"
)

//go:embed config.yaml
var embeddedConfigFile []byte

type AppConfig struct {
	Models      []ModelConfig `yaml:"models"`
	Preferences Preferences   `yaml:"preferences"`
	Version     string        `yaml:"config_format_version"`
}

func LoadAppConfig() (config AppConfig, err error) {
	fullFilePath, err := FullFilePath(ConfigFilePath)
	if err != nil {
		return config, fmt.Errorf("error getting config file path: %s", err)
	}
	// 如果配置文件不存在 创建默认的
	if _, err := os.Stat(fullFilePath); os.IsNotExist(err) {
		return createConfigWithDefaults(ConfigFilePath)
	}
	// 如果配置文件存在 使用它
	return loadExistingConfig(ConfigFilePath)
}

func FullFilePath(relativeFilePath string) (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("error getting home directory: %s", err)
	}
	configFilePath := filepath.Join(homeDir, relativeFilePath)
	return configFilePath, nil
}

// 需传入相对路径
func createConfigWithDefaults(filePath string) (AppConfig, error) {
	fullFilePath, _ := FullFilePath(filePath)
	config := AppConfig{}
	err := yaml.Unmarshal(embeddedConfigFile, &config)
	if err != nil {
		return config, fmt.Errorf("error unmarshalling embedded config: %s", err)
	}
	// 允许环境变量覆盖配置文件 - 模型名称
	modelOverride := os.Getenv("QSH_MODEL_OVERRIDE")
	if modelOverride != "" {
		config.Preferences.DefaultModel = modelOverride
	}

	return config, writeConfigToFile(config, fullFilePath)
}

// 需传入绝对路径
func writeConfigToFile(config AppConfig, fullFilePath string) error {
	// 如果目录不存在就创建目录
	dir := filepath.Dir(fullFilePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("error creating directories: %s", err)
	}
	configData, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("error marshalling config: %s", err)
	}

	err = os.WriteFile(fullFilePath, configData, 0644)
	if err != nil {
		return fmt.Errorf("error writing config to file: %s", err)
	}
	return SaveBackupConfig(config)
}

func SaveBackupConfig(config AppConfig) error {
	filePath, err := FullFilePath(BackupConfigFilePath)
	if err != nil {
		return err
	}
	configData, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("error marshalling config: %s", err)
	}

	err = os.WriteFile(filePath, configData, 0644)
	if err != nil {
		return fmt.Errorf("error writing config to file: %s", err)
	}
	return nil
}

// 需传入相对路径
func loadExistingConfig(filePath string) (AppConfig, error) {
	fullFilepath, _ := FullFilePath(filePath)
	config := AppConfig{}
	yamlFile, err := os.ReadFile(fullFilepath)
	if err != nil {
		return config, fmt.Errorf("error reading config file: %s", err)
	}
	err = yaml.Unmarshal(yamlFile, &config)
	if err != nil {
		return config, fmt.Errorf("error unmarshalling config file: %s", err)
	}
	return config, nil
}

func GetModelConfig(appConfig AppConfig) (ModelConfig, error) {
	if len(appConfig.Models) == 0 {
		return ModelConfig{}, fmt.Errorf("no models available")
	}
	for _, model := range appConfig.Models {
		if model.ModelName == appConfig.Preferences.DefaultModel {
			return model, nil
		}
	}
	// If the preferred model is not found, return the first model
	return appConfig.Models[0], nil
}

func SaveAppConfig(config AppConfig) error {
	fullFilePath, _ := FullFilePath(ConfigFilePath)
	return writeConfigToFile(config, fullFilePath)
}
