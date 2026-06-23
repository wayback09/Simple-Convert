// Helper to format bytes to human-readable size
function formatBytes(bytes) {
    if (bytes === 0 || !bytes) return '0 Bytes';
    const k = 1024;
    const sizes = ['Bytes', 'KB', 'MB', 'GB', 'TB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
}

// Helper to format seconds to HH:MM:SS
function formatDuration(sec) {
    if (!sec || isNaN(sec)) return '00:00:00';
    const hours = Math.floor(sec / 3600);
    const minutes = Math.floor((sec % 3600) / 60);
    const seconds = Math.floor(sec % 60);
    return [
        hours.toString().padStart(2, '0'),
        minutes.toString().padStart(2, '0'),
        seconds.toString().padStart(2, '0')
    ].join(':');
}

document.addEventListener('DOMContentLoaded', () => {
    // Window control elements and actions
    const winMinBtn = document.getElementById('win-min-btn');
    const winMaxBtn = document.getElementById('win-max-btn');
    const winCloseBtn = document.getElementById('win-close-btn');

    if (winMinBtn) {
        winMinBtn.addEventListener('click', () => {
            if (window.runtime && window.runtime.WindowMinimise) {
                window.runtime.WindowMinimise();
            }
        });
    }
    if (winMaxBtn) {
        winMaxBtn.addEventListener('click', () => {
            if (window.runtime && window.runtime.WindowToggleMaximise) {
                window.runtime.WindowToggleMaximise();
            }
        });
    }
    if (winCloseBtn) {
        winCloseBtn.addEventListener('click', () => {
            if (window.runtime && window.runtime.Quit) {
                window.runtime.Quit();
            }
        });
    }

    // State management
    let selectedInputPath = '';
    let selectedOutputPath = '';
    let inputMediaInfo = null;
    let isConverting = false;

    // DOM Elements - Screens
    const setupScreen = document.getElementById('setup-screen');
    const appScreen = document.getElementById('app-screen');

    // DOM Elements - Setup
    const downloadBtn = document.getElementById('download-btn');
    const retryBtn = document.getElementById('retry-btn');
    const setupProgressContainer = document.getElementById('setup-progress-container');
    const setupStatus = document.getElementById('setup-status');
    const setupProgressPct = document.getElementById('setup-progress-pct');
    const setupProgressBar = document.getElementById('setup-progress-bar');
    const setupDownloaded = document.getElementById('setup-downloaded');
    const setupTotal = document.getElementById('setup-total');
    const setupErrorCard = document.getElementById('setup-error-card');
    const setupErrorMsg = document.getElementById('setup-error-msg');

    // DOM Elements - File Input & Details
    const dropZone = document.getElementById('drop-zone');
    const selectFileBtn = document.getElementById('select-file-btn');
    const fileInfoCard = document.getElementById('file-info-card');
    const clearFileBtn = document.getElementById('clear-file-btn');
    const infoPath = document.getElementById('info-path');
    const infoSize = document.getElementById('info-size');
    const infoFormat = document.getElementById('info-format');
    const infoDuration = document.getElementById('info-duration');
    const infoVideoSection = document.getElementById('info-video-section');
    const infoVideoCodec = document.getElementById('info-video-codec');
    const infoVideoRes = document.getElementById('info-video-res');
    const infoVideoFps = document.getElementById('info-video-fps');
    const infoAudioSection = document.getElementById('info-audio-section');
    const infoAudioCodec = document.getElementById('info-audio-codec');
    const infoAudioChannels = document.getElementById('info-audio-channels');

    // DOM Elements - Parameters & Configuration
    const targetFormat = document.getElementById('target-format');
    const videoSettingsGroup = document.getElementById('video-settings-group');
    const targetResolution = document.getElementById('target-resolution');
    const targetQuality = document.getElementById('target-quality');
    const enableTrim = document.getElementById('enable-trim');
    const trimInputs = document.getElementById('trim-inputs');
    const trimStart = document.getElementById('trim-start');
    const trimEnd = document.getElementById('trim-end');
    const stripAudioWrapper = document.getElementById('strip-audio-wrapper');
    const stripAudio = document.getElementById('strip-audio');
    const extractAudioWrapper = document.getElementById('extract-audio-wrapper');
    const extractAudio = document.getElementById('extract-audio');
    const outputPathInput = document.getElementById('output-path');
    const selectOutputBtn = document.getElementById('select-output-btn');
    const convertBtn = document.getElementById('convert-btn');

    const noReencode = document.getElementById('no-reencode');
    const noReencodeWrapper = document.getElementById('no-reencode-wrapper');
    const hwAccel = document.getElementById('hw-accel');
    const hwAccelGroup = document.getElementById('hw-accel-group');
    const optNvenc = document.getElementById('opt-nvenc');
    const optAmf = document.getElementById('opt-amf');
    const optQsv = document.getElementById('opt-qsv');

    // DOM Elements - Conversion Progress Panel
    const progressPanel = document.getElementById('progress-panel');
    const conversionStatus = document.getElementById('conversion-status');
    const cancelBtn = document.getElementById('cancel-btn');
    const conversionProgressBar = document.getElementById('conversion-progress-bar');
    const conversionPct = document.getElementById('conversion-pct');
    const metricFrame = document.getElementById('metric-frame');
    const metricFps = document.getElementById('metric-fps');
    const metricSpeed = document.getElementById('metric-speed');
    const metricBitrate = document.getElementById('metric-bitrate');
    const logsTerminal = document.getElementById('logs-terminal');
    const toggleTerminalBtn = document.getElementById('toggle-terminal-btn');

    // ==========================================
    // INITIALIZATION & CHECK SETUP
    // ==========================================

    async function checkFFmpegAndInit() {
        try {
            const hasFFmpeg = await window.go.main.App.CheckFFmpeg();
            if (hasFFmpeg) {
                setupScreen.classList.add('hidden');
                appScreen.classList.remove('hidden');
                initHardwareEncoders();
            } else {
                setupScreen.classList.remove('hidden');
                appScreen.classList.add('hidden');
                setupProgressContainer.classList.add('hidden');
                setupErrorCard.classList.add('hidden');
                downloadBtn.removeAttribute('disabled');
            }
        } catch (err) {
            showSetupError("Failed to check for FFmpeg: " + err);
        }
    }

    async function initHardwareEncoders() {
        try {
            const encoders = await window.go.main.App.GetAvailableHardwareEncoders();
            
            if (encoders.nvenc) {
                optNvenc.textContent = "NVIDIA NVENC (Supported)";
                optNvenc.removeAttribute('disabled');
            } else {
                optNvenc.textContent = "NVIDIA NVENC (Unsupported)";
                optNvenc.setAttribute('disabled', 'true');
            }

            if (encoders.amf) {
                optAmf.textContent = "AMD AMF (Supported)";
                optAmf.removeAttribute('disabled');
            } else {
                optAmf.textContent = "AMD AMF (Unsupported)";
                optAmf.setAttribute('disabled', 'true');
            }

            if (encoders.qsv) {
                optQsv.textContent = "Intel QSV (Supported)";
                optQsv.removeAttribute('disabled');
            } else {
                optQsv.textContent = "Intel QSV (Unsupported)";
                optQsv.setAttribute('disabled', 'true');
            }

            const hasAnyHw = encoders.nvenc || encoders.amf || encoders.qsv;
            const autoOpt = hwAccel.querySelector('option[value="auto"]');
            if (!hasAnyHw) {
                hwAccel.value = "none";
                if (autoOpt) {
                    autoOpt.textContent = "Auto-Detect GPU (No GPU detected)";
                    autoOpt.setAttribute('disabled', 'true');
                }
            } else {
                if (autoOpt) {
                    autoOpt.textContent = "Auto-Detect GPU";
                    autoOpt.removeAttribute('disabled');
                }
            }
        } catch (err) {
            console.error("Failed to query hardware encoders", err);
        }
    }

    // Call check on start
    if (window.go && window.go.main && window.go.main.App) {
        checkFFmpegAndInit();
    } else {
        // Fallback for development if loaded before go bindings
        window.addEventListener('wailsbind', checkFFmpegAndInit);
    }

    // ==========================================
    // FFmpeg DOWNLOAD & SETUP FLOW
    // ==========================================

    async function startSetup() {
        downloadBtn.setAttribute('disabled', 'true');
        retryBtn.setAttribute('disabled', 'true');
        setupErrorCard.classList.add('hidden');
        setupProgressContainer.classList.remove('hidden');

        // Set initial state
        setupProgressBar.style.width = '0%';
        setupProgressPct.textContent = '0%';
        setupStatus.textContent = 'Connecting...';
        setupDownloaded.textContent = '0 MB';
        setupTotal.textContent = '0 MB';

        // Setup event listener
        window.runtime.EventsOn('setup:progress', (data) => {
            if (data.percent !== undefined) {
                setupProgressBar.style.width = data.percent + '%';
                setupProgressPct.textContent = data.percent + '%';
            }
            if (data.status) {
                setupStatus.textContent = data.status;
            }
            if (data.downloaded) {
                setupDownloaded.textContent = data.downloaded;
            }
            if (data.total) {
                setupTotal.textContent = data.total;
            }
        });

        try {
            await window.go.main.App.SetupFFmpeg();
            // Success
            window.runtime.EventsOff('setup:progress');
            checkFFmpegAndInit();
        } catch (err) {
            window.runtime.EventsOff('setup:progress');
            showSetupError(err);
        }
    }

    function showSetupError(msg) {
        setupProgressContainer.classList.add('hidden');
        setupErrorCard.classList.remove('hidden');
        setupErrorMsg.textContent = msg;
        retryBtn.removeAttribute('disabled');
    }

    downloadBtn.addEventListener('click', startSetup);
    retryBtn.addEventListener('click', startSetup);

    // ==========================================
    // FILE INPUT HANDLING (DRAG/DROP & BROWSE)
    // ==========================================

    // Drag and Drop (using Wails native API - HTML5 drop.path is unavailable in Wails webview)
    // useDropTarget=false so drops are accepted anywhere on the window, not just on
    // elements marked with the --wails-drop-target CSS property.
    window.runtime.OnFileDrop((x, y, paths) => {
        if (paths && paths.length > 0) {
            const filePath = paths[0];
            if (filePath) {
                dropZone.classList.remove('dragover');
                handleFileSelected(filePath);
            }
        }
    }, false);

    // Visual feedback only (dragover/dragleave events DO fire in the webview;
    // only the File.path property is unavailable, which is why we use OnFileDrop above)
    dropZone.addEventListener('dragover', (e) => {
        e.preventDefault();
        dropZone.classList.add('dragover');
    });

    dropZone.addEventListener('dragleave', () => {
        dropZone.classList.remove('dragover');
    });

    // Browse Button
    selectFileBtn.addEventListener('click', async () => {
        try {
            const filePath = await window.go.main.App.SelectInputFile();
            if (filePath) {
                handleFileSelected(filePath);
            }
        } catch (err) {
            console.error("File selection failed", err);
        }
    });

    async function handleFileSelected(filePath) {
        try {
            const info = await window.go.main.App.GetMediaInfo(filePath);
            inputMediaInfo = info;
            selectedInputPath = filePath;

            // Populate Metadata
            infoPath.textContent = info.path;
            infoSize.textContent = formatBytes(info.size);
            infoFormat.textContent = info.format;
            infoDuration.textContent = formatDuration(info.duration);

            // Populate Video Stream info
            if (info.hasVideo) {
                infoVideoSection.classList.remove('hidden');
                infoVideoCodec.textContent = info.videoCodec;
                infoVideoRes.textContent = info.resolution;
                infoVideoFps.textContent = info.frameRate;
            } else {
                infoVideoSection.classList.add('hidden');
            }

            // Populate Audio Stream info
            if (info.hasAudio) {
                infoAudioSection.classList.remove('hidden');
                infoAudioCodec.textContent = info.audioCodec;
                infoAudioChannels.textContent = info.audioChannels + ' ch';
            } else {
                infoAudioSection.classList.add('hidden');
            }

            // Toggle card visibility
            dropZone.classList.add('hidden');
            fileInfoCard.classList.remove('hidden');

            updateFormStates();
        } catch (err) {
            alert("Error parsing file metadata: " + err);
        }
    }

    function updateFormStates() {
        if (!selectedInputPath) return;

        const format = targetFormat.value;
        const isAudioOnly = (format === 'mp3' || format === 'wav');

        // Direct Stream Copy option visibility/disabled
        if (isAudioOnly) {
            noReencode.checked = false;
            noReencodeWrapper.classList.add('hidden');
        } else {
            noReencodeWrapper.classList.remove('hidden');
        }

        if (noReencode.checked) {
            // Disable re-encoding settings
            videoSettingsGroup.classList.add('hidden');
            hwAccelGroup.classList.add('hidden');

            if (inputMediaInfo && inputMediaInfo.hasVideo) {
                stripAudioWrapper.classList.remove('hidden');
            } else {
                stripAudioWrapper.classList.add('hidden');
            }
            if (inputMediaInfo && inputMediaInfo.hasAudio) {
                extractAudioWrapper.classList.remove('hidden');
            } else {
                extractAudioWrapper.classList.add('hidden');
            }
        } else {
            // Re-encoding active
            if (isAudioOnly) {
                videoSettingsGroup.classList.add('hidden');
                stripAudioWrapper.classList.add('hidden');
                extractAudioWrapper.classList.add('hidden');
                hwAccelGroup.classList.add('hidden');
            } else {
                hwAccelGroup.classList.remove('hidden');
                if (inputMediaInfo && inputMediaInfo.hasVideo) {
                    videoSettingsGroup.classList.remove('hidden');
                    stripAudioWrapper.classList.remove('hidden');
                } else {
                    videoSettingsGroup.classList.add('hidden');
                    stripAudioWrapper.classList.add('hidden');
                }
                if (inputMediaInfo && inputMediaInfo.hasAudio) {
                    extractAudioWrapper.classList.remove('hidden');
                } else {
                    extractAudioWrapper.classList.add('hidden');
                }
            }
        }

        suggestOutputFile();
        validateForm();
    }

    // Clear Selected File
    clearFileBtn.addEventListener('click', () => {
        selectedInputPath = '';
        selectedOutputPath = '';
        inputMediaInfo = null;
        outputPathInput.value = '';
        noReencode.checked = false;

        fileInfoCard.classList.add('hidden');
        dropZone.classList.remove('hidden');

        updateFormStates();
    });

    // ==========================================
    // PARAMETERS & OUTPUT CONFIGURATION
    // ==========================================

    targetFormat.addEventListener('change', updateFormStates);
    noReencode.addEventListener('change', updateFormStates);

    enableTrim.addEventListener('change', () => {
        if (enableTrim.checked) {
            trimInputs.classList.remove('hidden');
        } else {
            trimInputs.classList.add('hidden');
        }
    });

    // Output Location Picker
    selectOutputBtn.addEventListener('click', async () => {
        if (!selectedInputPath) return;

        try {
            const ext = targetFormat.value;
            const filter = `*.${ext}`;
            const defaultFilename = getProposedFilename(selectedInputPath, ext);

            const filePath = await window.go.main.App.SelectOutputFile(defaultFilename, filter);
            if (filePath) {
                selectedOutputPath = filePath;
                outputPathInput.value = filePath;
                validateForm();
            }
        } catch (err) {
            console.error("Output selection failed", err);
        }
    });

    function getProposedFilename(inputPath, extension) {
        // Extract filename without path and extension
        const baseName = inputPath.replace(/^.*[\\\/]/, '');
        const lastDot = baseName.lastIndexOf('.');
        const nameWithoutExt = lastDot !== -1 ? baseName.substring(0, lastDot) : baseName;
        return `${nameWithoutExt}_converted.${extension}`;
    }

    function suggestOutputFile() {
        if (!selectedInputPath) return;

        const ext = targetFormat.value;
        const dir = selectedInputPath.substring(0, selectedInputPath.lastIndexOf(/[\\\/]/) + 1);
        const name = getProposedFilename(selectedInputPath, ext);
        selectedOutputPath = dir + name;
        outputPathInput.value = selectedOutputPath;
    }

    function validateForm() {
        const isValid = (selectedInputPath !== '' && selectedOutputPath !== '');
        convertBtn.disabled = !isValid;
    }

    // ==========================================
    // MEDIA CONVERSION FLOW
    // ==========================================

    async function startConversion() {
        if (isConverting) return;

        isConverting = true;
        setFieldsDisabled(true);
        progressPanel.classList.remove('hidden');
        logsTerminal.classList.add('hidden');
        toggleTerminalBtn.textContent = 'SHOW LOGS';

        conversionProgressBar.style.width = '0%';
        conversionPct.textContent = '0%';
        conversionStatus.textContent = 'CONVERTING MEDIA FILE...';
        metricFrame.textContent = '-';
        metricFps.textContent = '-';
        metricSpeed.textContent = '-';
        metricBitrate.textContent = '-';
        logsTerminal.textContent = '';
        cancelBtn.classList.remove('hidden');

        // Setup live progress listener
        window.runtime.EventsOn('conversion:progress', (data) => {
            if (data.percent !== undefined) {
                conversionProgressBar.style.width = data.percent + '%';
                conversionPct.textContent = data.percent + '%';
            }
            if (data.frame) metricFrame.textContent = data.frame;
            if (data.fps) metricFps.textContent = data.fps;
            if (data.speed) metricSpeed.textContent = data.speed;
            if (data.bitrate) metricBitrate.textContent = data.bitrate;
        });

        // Setup live stderr log listener
        window.runtime.EventsOn('conversion:log', (line) => {
            logsTerminal.textContent += line + '\n';
            // Auto scroll terminal to bottom
            logsTerminal.scrollTop = logsTerminal.scrollHeight;
        });

        const params = {
            inputPath: selectedInputPath,
            outputPath: selectedOutputPath,
            targetFormat: targetFormat.value,
            resolutionScale: targetResolution.value,
            qualityPreset: targetQuality.value,
            enableTrim: enableTrim.checked,
            trimStart: trimStart.value.trim(),
            trimEnd: trimEnd.value.trim(),
            stripAudio: stripAudio.checked,
            extractAudio: extractAudio.checked,
            hwAccel: hwAccel.value,
            noReencode: noReencode.checked
        };

        try {
            await window.go.main.App.StartConversion(params);
            
            // Completed successfully
            conversionStatus.textContent = 'CONVERSION COMPLETE';
            conversionProgressBar.style.width = '100%';
            conversionPct.textContent = '100%';
        } catch (err) {
            if (err.toString().includes("cancelled")) {
                conversionStatus.textContent = 'CONVERSION CANCELLED';
            } else {
                conversionStatus.textContent = 'CONVERSION ERROR';
                logsTerminal.classList.remove('hidden');
                toggleTerminalBtn.textContent = 'HIDE LOGS';
                logsTerminal.textContent += '\n\nERROR: ' + err + '\n';
                logsTerminal.scrollTop = logsTerminal.scrollHeight;
            }
        } finally {
            // Remove listeners
            window.runtime.EventsOff('conversion:progress');
            window.runtime.EventsOff('conversion:log');

            isConverting = false;
            cancelBtn.classList.add('hidden');
            setFieldsDisabled(false);
        }
    }

    // Cancel Button
    cancelBtn.addEventListener('click', async () => {
        conversionStatus.textContent = 'CANCELLING...';
        cancelBtn.classList.add('hidden');
        try {
            await window.go.main.App.CancelConversion();
        } catch (err) {
            console.error("Cancellation request failed", err);
        }
    });

    function setFieldsDisabled(disabled) {
        const elements = [
            targetFormat, targetResolution, targetQuality,
            enableTrim, trimStart, trimEnd, stripAudio,
            extractAudio, selectOutputBtn, clearFileBtn,
            selectFileBtn, convertBtn, noReencode, hwAccel
        ];
        elements.forEach(el => {
            if (el) {
                if (disabled) {
                    el.setAttribute('disabled', 'true');
                } else {
                    el.removeAttribute('disabled');
                }
            }
        });
        validateForm(); // correct convertBtn state based on validation
    }

    convertBtn.addEventListener('click', startConversion);

    // Toggle logs visibility
    toggleTerminalBtn.addEventListener('click', () => {
        if (logsTerminal.classList.contains('hidden')) {
            logsTerminal.classList.remove('hidden');
            toggleTerminalBtn.textContent = 'HIDE LOGS';
            logsTerminal.scrollTop = logsTerminal.scrollHeight;
        } else {
            logsTerminal.classList.add('hidden');
            toggleTerminalBtn.textContent = 'SHOW LOGS';
        }
    });
});
