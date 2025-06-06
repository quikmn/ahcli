// Audio Visualization
const AudioViz = {
    advancedExpanded: false,
    
    // Initialize audio visualization
    init() {
        console.log('🎧 Initializing CLEAN audio visualization...');
        
        // Set up advanced controls toggle
        this.setupAdvancedControls();
        
        console.log('✅ Audio visualization ready (using Go PortAudio data only!)');
    },
    
    // Set up advanced controls panel
    setupAdvancedControls() {
        // Initialize collapsed state
        const panels = document.getElementById('controlPanels');
        const arrow = document.getElementById('toggleArrow');
        
        if (panels && arrow) {
            panels.classList.add('collapsed');
            arrow.classList.add('collapsed');
        }
    },
    
    // Update visualization based on state from Go app
    update(state) {
        if (!state) return;
        
        // Update RAW input level (before processing)
        this.updateRawInputLevel(state.rawInputLevel || 0);
        
        // Update PROCESSED input level (after processing)
        this.updateProcessedInputLevel(state.inputLevel || 0);
        
        // Update noise gate status with visual activity
        this.updateGateStatus(state.gateOpen || false);
        
        // Update compression meter
        this.updateCompression(state.gainReduction || 0);
        
        // Update sensitivity indicator
        this.updateSensitivity(state.rawInputLevel || 0, state.inputLevel || 0);
        
        // Update audio quality indicator
        this.updateAudioQuality(state.audioQuality || 'Unknown');
        
        // Update bypass status
        this.updateBypassStatus(state.bypassProcessing || false);
        
        // Update preset status
        if (state.audioPreset) {
            this.updatePresetDisplay(state.audioPreset);
        }
    },
    
    // Update RAW input level (before any processing)
    updateRawInputLevel(level) {
        const inputLevel = Math.min(level * 100, 100);
        const inputFill = document.getElementById('rawInputMeterFill');
        const inputText = document.getElementById('rawInputLevelText');
        
        if (inputFill) {
            inputFill.style.width = `${inputLevel}%`;
        }
        
        if (inputText) {
            inputText.textContent = `${Math.round(inputLevel)}%`;
        }
    },
    
    // Update PROCESSED input level (after processing)
    updateProcessedInputLevel(level) {
        const inputLevel = Math.min(level * 100, 100);
        const inputFill = document.getElementById('processedInputMeterFill');
        const inputText = document.getElementById('processedInputLevelText');
        
        if (inputFill) {
            inputFill.style.width = `${inputLevel}%`;
            
            // Color coding for processed level
            if (inputLevel > 80) {
                inputFill.style.background = 'linear-gradient(to right, #27ae60 0%, #f39c12 70%, #e74c3c 90%)';
            } else if (inputLevel > 60) {
                inputFill.style.background = 'linear-gradient(to right, #27ae60 0%, #f39c12 80%)';
            } else {
                inputFill.style.background = '#00FF00';
            }
        }
        
        if (inputText) {
            inputText.textContent = `${Math.round(inputLevel)}%`;
        }
    },
    
    // Update noise gate status with VISUAL ACTIVITY
    updateGateStatus(isOpen) {
        const gateStatus = document.getElementById('gateStatus');
        const gateActivity = document.getElementById('gateActivity');
        
        if (gateStatus) {
            const wasOpen = gateStatus.classList.contains('open');
            
            if (isOpen !== wasOpen) {
                if (isOpen) {
                    gateStatus.textContent = 'OPEN';
                    gateStatus.className = 'gate-status open';
                } else {
                    gateStatus.textContent = 'CLOSED';
                    gateStatus.className = 'gate-status closed';
                }
            }
        }
        
        // Visual activity indicator
        if (gateActivity) {
            if (isOpen) {
                gateActivity.style.color = '#00FF41';
                gateActivity.style.animation = 'pulse 0.5s infinite';
            } else {
                gateActivity.style.color = '#FF1744';
                gateActivity.style.animation = 'none';
            }
        }
    },
    
    // Update sensitivity indicator (shows if processing makes mic more/less sensitive)
    updateSensitivity(rawLevel, processedLevel) {
        const indicator = document.getElementById('sensitivityIndicator');
        const text = document.getElementById('sensitivityText');
        
        if (!indicator || !text) return;
        
        const bars = indicator.querySelectorAll('.sensitivity-bar');
        const ratio = rawLevel > 0 ? processedLevel / rawLevel : 1;
        
        // Clear all bars
        bars.forEach(bar => {
            bar.classList.remove('active', 'boost', 'cut');
        });
        
        if (ratio > 1.2) {
            // Processing is boosting the signal (more sensitive)
            text.textContent = 'Boosted';
            text.style.color = '#00FF41';
            bars.forEach((bar, i) => {
                if (i < Math.min(4, Math.floor(ratio * 2))) {
                    bar.classList.add('active', 'boost');
                }
            });
        } else if (ratio < 0.8) {
            // Processing is cutting the signal (less sensitive)
            text.textContent = 'Cut';
            text.style.color = '#FF1744';
            bars.forEach((bar, i) => {
                if (i < Math.min(4, Math.floor((1 - ratio) * 5))) {
                    bar.classList.add('active', 'cut');
                }
            });
        } else {
            // Normal level
            text.textContent = 'Normal';
            text.style.color = '#00BFFF';
            bars[2].classList.add('active'); // Middle bar
        }
    },
    
    // Toggle bypass mode
    toggleBypass(bypass) {
        console.log('Toggling bypass mode:', bypass);
        
        // Send bypass command to server
        fetch('/api/command', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
                command: 'bypass_processing',
                args: bypass ? 'true' : 'false'
            })
        }).catch(error => {
            console.error('Failed to toggle bypass:', error);
        });
        
        // Update UI immediately
        this.updateBypassStatus(bypass);
    },
    
    // Update bypass status display
    updateBypassStatus(bypass) {
        const statusElement = document.getElementById('bypassStatus');
        const checkbox = document.getElementById('bypassProcessing');
        
        if (statusElement) {
            if (bypass) {
                statusElement.textContent = 'Processing: BYPASSED';
                statusElement.style.color = '#FF1744';
                statusElement.style.fontWeight = 'bold';
            } else {
                statusElement.textContent = 'Processing: ACTIVE';
                statusElement.style.color = '#00FF41';
                statusElement.style.fontWeight = 'normal';
            }
        }
        
        if (checkbox) {
            checkbox.checked = bypass;
        }
    },
    
    // Update compression meter (from Go audio processor)
    updateCompression(gainReduction) {
        const compressionLevel = Math.min(gainReduction * 100, 100);
        const compressionFill = document.getElementById('compressionMeterFill');
        const compressionText = document.getElementById('compressionText');
        
        if (compressionFill) {
            compressionFill.style.width = `${compressionLevel}%`;
        }
        
        if (compressionText) {
            if (compressionLevel > 1) {
                const reductionDB = -(compressionLevel / 10); // Rough conversion to dB
                compressionText.textContent = `${reductionDB.toFixed(1)}dB`;
            } else {
                compressionText.textContent = '0.0dB';
            }
        }
    },
    
    // Update audio quality indicator (from Go audio processor)
    updateAudioQuality(quality) {
        const qualityElement = document.getElementById('audioQuality');
        if (qualityElement) {
            qualityElement.textContent = quality;
            qualityElement.className = `quality-${quality.toLowerCase()}`;
        }
    },
    
    // Toggle advanced controls panel
    toggleAdvanced() {
        this.advancedExpanded = !this.advancedExpanded;
        const panels = document.getElementById('controlPanels');
        const arrow = document.getElementById('toggleArrow');
        
        if (this.advancedExpanded) {
            panels?.classList.remove('collapsed');
            arrow?.classList.remove('collapsed');
        } else {
            panels?.classList.add('collapsed');
            arrow?.classList.add('collapsed');
        }
    },
    
    // Change audio preset
    changePreset(preset) {
        console.log('Changing audio preset to:', preset);
        
        // Send preset change to server
        fetch('/api/command', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
                command: 'audio_preset',
                args: preset
            })
        }).catch(error => {
            console.error('Failed to change preset:', error);
        });
        
        // Update UI immediately for responsiveness
        this.updatePresetDisplay(preset);
        this.updateControlsFromPreset(preset);
    },
    
    // Update individual audio setting
    updateSetting(section, param, value) {
        console.log(`Updating ${section}.${param} = ${value}`);
        
        // Set preset to custom when manual changes are made
        const presetSelect = document.getElementById('audioPreset');
        if (presetSelect) {
            presetSelect.value = 'custom';
            this.updatePresetDisplay('custom');
        }
        
        // Send setting to server
        fetch('/api/command', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
                command: 'audio_setting',
                args: JSON.stringify({ section, param, value })
            })
        }).catch(error => {
            console.error('Failed to update setting:', error);
        });
    },
    
    // Update slider display values
    updateSliderValue(section, param, value) {
        switch (section) {
            case 'noiseGate':
                if (param === 'threshold') {
                    const element = document.getElementById('noiseGateValue');
                    if (element) element.textContent = `${value}dB`;
                }
                break;
            case 'compressor':
                if (param === 'threshold') {
                    const element = document.getElementById('compThresholdValue');
                    if (element) element.textContent = `${value}dB`;
                } else if (param === 'ratio') {
                    const element = document.getElementById('compRatioValue');
                    if (element) element.textContent = `${value}:1`;
                }
                break;
            case 'makeupGain':
                if (param === 'gain') {
                    const element = document.getElementById('makeupGainValue');
                    if (element) element.textContent = `+${value}dB`;
                }
                break;
        }
    },
    
    // Update controls based on preset selection
    updateControlsFromPreset(preset) {
        const presets = {
            'off': {
                noiseGate: { enabled: false, threshold: -40 },
                compressor: { enabled: false, threshold: -18, ratio: 3.0 },
                makeupGain: { enabled: false, gain: 6 }
            },
            'light': {
                noiseGate: { enabled: true, threshold: -45 },
                compressor: { enabled: true, threshold: -18, ratio: 2.0 },
                makeupGain: { enabled: true, gain: 3 }
            },
            'balanced': {
                noiseGate: { enabled: true, threshold: -35 },
                compressor: { enabled: true, threshold: -18, ratio: 3.0 },
                makeupGain: { enabled: true, gain: 6 }
            },
            'aggressive': {
                noiseGate: { enabled: true, threshold: -25 },
                compressor: { enabled: true, threshold: -18, ratio: 4.0 },
                makeupGain: { enabled: true, gain: 9 }
            }
        };
        
        if (presets[preset]) {
            const config = presets[preset];
            
            // Update checkboxes
            const noiseGateEnabled = document.getElementById('noiseGateEnabled');
            const compressorEnabled = document.getElementById('compressorEnabled');
            const makeupGainEnabled = document.getElementById('makeupGainEnabled');
            
            if (noiseGateEnabled) noiseGateEnabled.checked = config.noiseGate.enabled;
            if (compressorEnabled) compressorEnabled.checked = config.compressor.enabled;
            if (makeupGainEnabled) makeupGainEnabled.checked = config.makeupGain.enabled;
            
            // Update sliders
            const noiseGateThreshold = document.getElementById('noiseGateThreshold');
            const compThreshold = document.getElementById('compThreshold');
            const compRatio = document.getElementById('compRatio');
            const makeupGainLevel = document.getElementById('makeupGainLevel');
            
            if (noiseGateThreshold) noiseGateThreshold.value = config.noiseGate.threshold;
            if (compThreshold) compThreshold.value = config.compressor.threshold;
            if (compRatio) compRatio.value = config.compressor.ratio;
            if (makeupGainLevel) makeupGainLevel.value = config.makeupGain.gain;
            
            // Update display values
            this.updateSliderValue('noiseGate', 'threshold', config.noiseGate.threshold);
            this.updateSliderValue('compressor', 'threshold', config.compressor.threshold);
            this.updateSliderValue('compressor', 'ratio', config.compressor.ratio);
            this.updateSliderValue('makeupGain', 'gain', config.makeupGain.gain);
        }
    },
    
    // Update preset status display
    updatePresetDisplay(preset) {
        const statusElement = document.getElementById('presetStatus');
        if (!statusElement) return;
        
        statusElement.className = `preset-status ${preset}`;
        
        switch (preset) {
            case 'off':
                statusElement.textContent = 'Processing: Off';
                break;
            case 'light':
                statusElement.textContent = 'Processing: Light';
                break;
            case 'balanced':
                statusElement.textContent = 'Processing: Balanced';
                break;
            case 'aggressive':
                statusElement.textContent = 'Processing: Aggressive';
                break;
            case 'custom':
                statusElement.textContent = 'Processing: Custom';
                break;
        }
    },
    
    // Test microphone
    testMicrophone() {
        console.log('Testing microphone...');
        fetch('/api/command', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
                command: 'test_microphone',
                args: ''
            })
        }).catch(error => {
            console.error('Failed to test microphone:', error);
        });
    },
    
    // Reset to defaults
    resetDefaults() {
        if (confirm('Reset all audio settings to defaults?')) {
            const presetSelect = document.getElementById('audioPreset');
            if (presetSelect) {
                presetSelect.value = 'balanced';
                this.changePreset('balanced');
            }
        }
    },
    
    // Save custom preset
    saveCustom() {
        console.log('Saving custom preset...');
        fetch('/api/command', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
                command: 'save_custom_preset',
                args: ''
            })
        }).catch(error => {
            console.error('Failed to save custom preset:', error);
        });
    }
};