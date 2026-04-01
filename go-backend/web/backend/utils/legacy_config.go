package utils

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/raynaythegreat/octai-app/pkg/config"
	"github.com/raynaythegreat/octai-app/pkg/logger"
)

// ImportLegacyConfigIfNeeded copies model catalogs from the old PicoClaw config
// when the current OctAi config was onboarded without any configured models.
func ImportLegacyConfigIfNeeded(configPath string) error {
	shouldImport, err := configFileNeedsLegacyImport(configPath)
	if err != nil {
		return fmt.Errorf("inspect current config: %w", err)
	}
	if !shouldImport {
		return nil
	}

	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		return fmt.Errorf("load current config: %w", err)
	}

	legacyPath, err := legacyConfigPath()
	if err != nil {
		return fmt.Errorf("resolve legacy config path: %w", err)
	}
	if _, err := os.Stat(legacyPath); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("stat legacy config: %w", err)
	}

	legacyCfg, err := config.LoadConfig(legacyPath)
	if err != nil {
		return fmt.Errorf("load legacy config: %w", err)
	}
	if len(legacyCfg.ModelList) == 0 && len(legacyCfg.ImageModelList) == 0 && len(legacyCfg.VideoModelList) == 0 {
		return nil
	}

	cfg.ModelList = cloneModelList(legacyCfg.ModelList)
	cfg.ImageModelList = cloneModelList(legacyCfg.ImageModelList)
	cfg.VideoModelList = cloneModelList(legacyCfg.VideoModelList)

	if cfg.Agents.Defaults.Provider == "" && legacyCfg.Agents.Defaults.Provider != "" {
		cfg.Agents.Defaults.Provider = legacyCfg.Agents.Defaults.Provider
	}
	if cfg.Agents.Defaults.ModelName == "" && legacyCfg.Agents.Defaults.ModelName != "" {
		cfg.Agents.Defaults.ModelName = legacyCfg.Agents.Defaults.ModelName
	}
	if cfg.Agents.Defaults.ImageModel == "" && legacyCfg.Agents.Defaults.ImageModel != "" {
		cfg.Agents.Defaults.ImageModel = legacyCfg.Agents.Defaults.ImageModel
	}
	if cfg.Agents.Defaults.VideoModel == "" && legacyCfg.Agents.Defaults.VideoModel != "" {
		cfg.Agents.Defaults.VideoModel = legacyCfg.Agents.Defaults.VideoModel
	}
	if len(cfg.Agents.Defaults.ModelFallbacks) == 0 && len(legacyCfg.Agents.Defaults.ModelFallbacks) > 0 {
		cfg.Agents.Defaults.ModelFallbacks = append([]string(nil), legacyCfg.Agents.Defaults.ModelFallbacks...)
	}
	if len(cfg.Agents.Defaults.ImageModelFallbacks) == 0 && len(legacyCfg.Agents.Defaults.ImageModelFallbacks) > 0 {
		cfg.Agents.Defaults.ImageModelFallbacks = append([]string(nil), legacyCfg.Agents.Defaults.ImageModelFallbacks...)
	}
	if len(cfg.Agents.Defaults.VideoModelFallbacks) == 0 && len(legacyCfg.Agents.Defaults.VideoModelFallbacks) > 0 {
		cfg.Agents.Defaults.VideoModelFallbacks = append([]string(nil), legacyCfg.Agents.Defaults.VideoModelFallbacks...)
	}
	if cfg.Agents.Defaults.Routing == nil && legacyCfg.Agents.Defaults.Routing != nil {
		cfg.Agents.Defaults.Routing = &config.RoutingConfig{
			Enabled:    legacyCfg.Agents.Defaults.Routing.Enabled,
			LightModel: legacyCfg.Agents.Defaults.Routing.LightModel,
			Threshold:  legacyCfg.Agents.Defaults.Routing.Threshold,
		}
	}

	if err := config.SaveConfig(configPath, cfg); err != nil {
		return fmt.Errorf("save imported config: %w", err)
	}

	logger.InfoC(
		"web",
		fmt.Sprintf(
			"Imported legacy model catalogs from %s (%d chat, %d image, %d video)",
			legacyPath,
			len(cfg.ModelList),
			len(cfg.ImageModelList),
			len(cfg.VideoModelList),
		),
	)
	return nil
}

func legacyConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".picoclaw", "config.json"), nil
}

func configFileNeedsLegacyImport(configPath string) (bool, error) {
	raw, err := os.ReadFile(configPath)
	if err != nil {
		return false, err
	}

	var catalog struct {
		ModelList      []json.RawMessage `json:"model_list"`
		ImageModelList []json.RawMessage `json:"image_model_list"`
		VideoModelList []json.RawMessage `json:"video_model_list"`
	}
	if err := json.Unmarshal(raw, &catalog); err != nil {
		return false, err
	}

	return len(catalog.ModelList) == 0 &&
		len(catalog.ImageModelList) == 0 &&
		len(catalog.VideoModelList) == 0, nil
}

func cloneModelList(models []*config.ModelConfig) []*config.ModelConfig {
	cloned := make([]*config.ModelConfig, 0, len(models))
	for _, model := range models {
		if model == nil || model.IsVirtual() {
			continue
		}

		copyModel := &config.ModelConfig{
			ModelName:      model.ModelName,
			Model:          model.Model,
			APIBase:        model.APIBase,
			Proxy:          model.Proxy,
			Fallbacks:      append([]string(nil), model.Fallbacks...),
			AuthMethod:     model.AuthMethod,
			ConnectMode:    model.ConnectMode,
			Workspace:      model.Workspace,
			RPM:            model.RPM,
			MaxTokensField: model.MaxTokensField,
			RequestTimeout: model.RequestTimeout,
			ThinkingLevel:  model.ThinkingLevel,
		}
		if model.ExtraBody != nil {
			copyModel.ExtraBody = make(map[string]any, len(model.ExtraBody))
			for k, v := range model.ExtraBody {
				copyModel.ExtraBody[k] = v
			}
		}
		if apiKey := model.APIKey(); apiKey != "" {
			copyModel.SetAPIKey(apiKey)
		}
		cloned = append(cloned, copyModel)
	}
	return cloned
}
