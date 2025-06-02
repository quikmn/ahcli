package main

import (
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
	Nickname         []string              `json:"nickname"`
	PreferredServer  string                `json:"preferred_server"`
	PTTKey           string                `json:"ptt_key"`
	AudioProcessing  AudioProcessingConfig `json:"audio_processing"`
	Servers          map[string]ServerEntry `json:"servers"`
}

func loadClientConfig(path string) (*ClientConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var config ClientConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}
	return &config, nil
}

func saveClientConfig(path string, config *ClientConfig) error {
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// Audio preset system
func applyAudioPreset(config *ClientConfig, preset string) {
	config.AudioProcessing.Preset = preset
	
	switch preset {
	case "off":
		config.AudioProcessing.NoiseGate.Enabled = false
		config.AudioProcessing.Compressor.Enabled = false
		config.AudioProcessing.MakeupGain.Enabled = false
		
	case "light":
		config.AudioProcessing.NoiseGate.Enabled = true
		config.AudioProcessing.NoiseGate.ThresholdDB = -45
		config.AudioProcessing.Compressor.Enabled = true
		config.AudioProcessing.Compressor.ThresholdDB = -18
		config.AudioProcessing.Compressor.Ratio = 2.0
		config.AudioProcessing.MakeupGain.Enabled = true
		config.AudioProcessing.MakeupGain.GainDB = 3
		
	case "balanced":
		config.AudioProcessing.NoiseGate.Enabled = true
		config.AudioProcessing.NoiseGate.ThresholdDB = -35
		config.AudioProcessing.Compressor.Enabled = true
		config.AudioProcessing.Compressor.ThresholdDB = -18
		config.AudioProcessing.Compressor.Ratio = 3.0
		config.AudioProcessing.MakeupGain.Enabled = true
		config.AudioProcessing.MakeupGain.GainDB = 6
		
	case "aggressive":
		config.AudioProcessing.NoiseGate.Enabled = true
		config.AudioProcessing.NoiseGate.ThresholdDB = -25
		config.AudioProcessing.Compressor.Enabled = true
		config.AudioProcessing.Compressor.ThresholdDB = -18
		config.AudioProcessing.Compressor.Ratio = 4.0
		config.AudioProcessing.MakeupGain.Enabled = true
		config.AudioProcessing.MakeupGain.GainDB = 9
	}
}

// Apply audio settings to the processor
func applyAudioConfigToProcessor(config *ClientConfig) {
	if audioProcessor == nil {
		return
	}
	
	// Update processor settings based on config
	audioProcessor.enableNoiseGate = config.AudioProcessing.NoiseGate.Enabled
	audioProcessor.enableCompressor = config.AudioProcessing.Compressor.Enabled
	audioProcessor.enableMakeupGain = config.AudioProcessing.MakeupGain.Enabled
	
	// Update thresholds and parameters
	if audioProcessor.noiseGate != nil {
		audioProcessor.noiseGate.threshold = config.AudioProcessing.NoiseGate.ThresholdDB
	}
	
	if audioProcessor.compressor != nil {
		audioProcessor.compressor.threshold = config.AudioProcessing.Compressor.ThresholdDB
		audioProcessor.compressor.ratio = config.AudioProcessing.Compressor.Ratio
	}
	
	if audioProcessor.makeupGain != nil {
		audioProcessor.makeupGain.gainDB = config.AudioProcessing.MakeupGain.GainDB
		// Recalculate linear gain
		audioProcessor.makeupGain.gainLinear = powf(10.0, audioProcessor.makeupGain.gainDB/20.0)
	}
}