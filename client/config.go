// FILE: client/config.go
package main

import (
	"ahcli/common/logger"
	"encoding/json"
	"os"
)

type AudioProcessingConfig struct {
	NoiseGate struct {
		Enabled     bool    `json:"enabled"`
		ThresholdDB float32 `json:"threshold_db"`
	} `json:"noise_gate"`
	Compressor struct {
		Enabled     bool    `json:"enabled"`
		ThresholdDB float32 `json:"threshold_db"`
		Ratio       float32 `json:"ratio"`
	} `json:"compressor"`
	MakeupGain struct {
		Enabled bool    `json:"enabled"`
		GainDB  float32 `json:"gain_db"`
	} `json:"makeup_gain"`
	Preset string `json:"preset"`
}

type ServerEntry struct {
	IP string `json:"ip"`
}

type ClientConfig struct {
	Nickname        []string               `json:"nickname"`
	PreferredServer string                 `json:"preferred_server"`
	PTTKey          string                 `json:"ptt_key"`
	AudioProcessing AudioProcessingConfig  `json:"audio_processing"`
	Servers         map[string]ServerEntry `json:"servers"`
}

func loadClientConfig(path string) (*ClientConfig, error) {
	logger.Info("Loading client configuration from: %s", path)

	data, err := os.ReadFile(path)
	if err != nil {
		logger.Error("Failed to read config file %s: %v", path, err)
		return nil, err
	}

	var config ClientConfig
	if err := json.Unmarshal(data, &config); err != nil {
		logger.Error("Failed to parse JSON in config file %s: %v", path, err)
		return nil, err
	}

	// Log what was loaded
	logger.Info("Configuration loaded successfully")
	logger.Debug("Nicknames: %v", config.Nickname)
	logger.Debug("Preferred server: %s", config.PreferredServer)
	logger.Debug("PTT key: %s", config.PTTKey)
	logger.Debug("Audio preset: %s", config.AudioProcessing.Preset)
	logger.Debug("Configured servers: %d", len(config.Servers))

	// Log server details
	for name, server := range config.Servers {
		logger.Debug("Server '%s': %s", name, server.IP)
	}

	// Log audio processing settings
	logger.Debug("Audio processing - NoiseGate: enabled=%t, threshold=%.1fdB",
		config.AudioProcessing.NoiseGate.Enabled,
		config.AudioProcessing.NoiseGate.ThresholdDB)
	logger.Debug("Audio processing - Compressor: enabled=%t, threshold=%.1fdB, ratio=%.1f",
		config.AudioProcessing.Compressor.Enabled,
		config.AudioProcessing.Compressor.ThresholdDB,
		config.AudioProcessing.Compressor.Ratio)
	logger.Debug("Audio processing - MakeupGain: enabled=%t, gain=%.1fdB",
		config.AudioProcessing.MakeupGain.Enabled,
		config.AudioProcessing.MakeupGain.GainDB)

	return &config, nil
}

func saveClientConfig(path string, config *ClientConfig) error {
	logger.Info("Saving client configuration to: %s", path)

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		logger.Error("Failed to marshal config to JSON: %v", err)
		return err
	}

	err = os.WriteFile(path, data, 0644)
	if err != nil {
		logger.Error("Failed to write config file %s: %v", path, err)
		return err
	}

	logger.Info("Configuration saved successfully")
	logger.Debug("Saved preset: %s", config.AudioProcessing.Preset)
	logger.Debug("Config file size: %d bytes", len(data))

	return nil
}

// Audio preset system
func applyAudioPreset(config *ClientConfig, preset string) {
	logger.Info("Applying audio preset: %s", preset)

	oldPreset := config.AudioProcessing.Preset
	config.AudioProcessing.Preset = preset

	switch preset {
	case "off":
		logger.Debug("Setting audio preset 'off' - disabling all processing")
		config.AudioProcessing.NoiseGate.Enabled = false
		config.AudioProcessing.Compressor.Enabled = false
		config.AudioProcessing.MakeupGain.Enabled = false

	case "light":
		logger.Debug("Setting audio preset 'light' - minimal processing")
		config.AudioProcessing.NoiseGate.Enabled = true
		config.AudioProcessing.NoiseGate.ThresholdDB = -45
		config.AudioProcessing.Compressor.Enabled = true
		config.AudioProcessing.Compressor.ThresholdDB = -18
		config.AudioProcessing.Compressor.Ratio = 2.0
		config.AudioProcessing.MakeupGain.Enabled = true
		config.AudioProcessing.MakeupGain.GainDB = 3

	case "balanced":
		logger.Debug("Setting audio preset 'balanced' - moderate processing")
		config.AudioProcessing.NoiseGate.Enabled = true
		config.AudioProcessing.NoiseGate.ThresholdDB = -35
		config.AudioProcessing.Compressor.Enabled = true
		config.AudioProcessing.Compressor.ThresholdDB = -18
		config.AudioProcessing.Compressor.Ratio = 3.0
		config.AudioProcessing.MakeupGain.Enabled = true
		config.AudioProcessing.MakeupGain.GainDB = 6

	case "aggressive":
		logger.Debug("Setting audio preset 'aggressive' - heavy processing")
		config.AudioProcessing.NoiseGate.Enabled = true
		config.AudioProcessing.NoiseGate.ThresholdDB = -25
		config.AudioProcessing.Compressor.Enabled = true
		config.AudioProcessing.Compressor.ThresholdDB = -18
		config.AudioProcessing.Compressor.Ratio = 4.0
		config.AudioProcessing.MakeupGain.Enabled = true
		config.AudioProcessing.MakeupGain.GainDB = 9

	default:
		logger.Warn("Unknown audio preset: %s", preset)
		return
	}

	logger.Info("Audio preset changed: %s -> %s", oldPreset, preset)

	// Log the new settings
	logger.Debug("New settings - NoiseGate: enabled=%t, threshold=%.1fdB",
		config.AudioProcessing.NoiseGate.Enabled,
		config.AudioProcessing.NoiseGate.ThresholdDB)
	logger.Debug("New settings - Compressor: enabled=%t, threshold=%.1fdB, ratio=%.1f",
		config.AudioProcessing.Compressor.Enabled,
		config.AudioProcessing.Compressor.ThresholdDB,
		config.AudioProcessing.Compressor.Ratio)
	logger.Debug("New settings - MakeupGain: enabled=%t, gain=%.1fdB",
		config.AudioProcessing.MakeupGain.Enabled,
		config.AudioProcessing.MakeupGain.GainDB)
}

// Apply audio settings to the processor
func applyAudioConfigToProcessor(config *ClientConfig) {
	if audioProcessor == nil {
		logger.Error("Cannot apply audio config: audioProcessor is nil")
		return
	}

	logger.Info("Applying audio configuration to processor")

	// Log what we're about to apply
	logger.Debug("Applying to processor - NoiseGate: %t, Compressor: %t, MakeupGain: %t",
		config.AudioProcessing.NoiseGate.Enabled,
		config.AudioProcessing.Compressor.Enabled,
		config.AudioProcessing.MakeupGain.Enabled)

	// Update processor settings based on config
	audioProcessor.enableNoiseGate = config.AudioProcessing.NoiseGate.Enabled
	audioProcessor.enableCompressor = config.AudioProcessing.Compressor.Enabled
	audioProcessor.enableMakeupGain = config.AudioProcessing.MakeupGain.Enabled

	// Update thresholds and parameters
	if audioProcessor.noiseGate != nil {
		oldThreshold := audioProcessor.noiseGate.threshold
		audioProcessor.noiseGate.threshold = config.AudioProcessing.NoiseGate.ThresholdDB
		logger.Debug("NoiseGate threshold: %.1fdB -> %.1fdB", oldThreshold, config.AudioProcessing.NoiseGate.ThresholdDB)
	} else {
		logger.Warn("NoiseGate processor is nil, cannot update threshold")
	}

	if audioProcessor.compressor != nil {
		oldThreshold := audioProcessor.compressor.threshold
		oldRatio := audioProcessor.compressor.ratio
		audioProcessor.compressor.threshold = config.AudioProcessing.Compressor.ThresholdDB
		audioProcessor.compressor.ratio = config.AudioProcessing.Compressor.Ratio
		logger.Debug("Compressor threshold: %.1fdB -> %.1fdB, ratio: %.1f -> %.1f",
			oldThreshold, config.AudioProcessing.Compressor.ThresholdDB,
			oldRatio, config.AudioProcessing.Compressor.Ratio)
	} else {
		logger.Warn("Compressor processor is nil, cannot update settings")
	}

	if audioProcessor.makeupGain != nil {
		oldGainDB := audioProcessor.makeupGain.gainDB
		audioProcessor.makeupGain.gainDB = config.AudioProcessing.MakeupGain.GainDB
		// Recalculate linear gain
		oldLinear := audioProcessor.makeupGain.gainLinear
		audioProcessor.makeupGain.gainLinear = powf(10.0, audioProcessor.makeupGain.gainDB/20.0)
		logger.Debug("MakeupGain: %.1fdB -> %.1fdB (linear: %.3f -> %.3f)",
			oldGainDB, config.AudioProcessing.MakeupGain.GainDB,
			oldLinear, audioProcessor.makeupGain.gainLinear)
	} else {
		logger.Warn("MakeupGain processor is nil, cannot update gain")
	}

	logger.Info("Audio configuration applied to processor successfully")
}
