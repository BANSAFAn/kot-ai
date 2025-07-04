package voice

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"cloud.google.com/go/speech/apiv1"
	speechpb "cloud.google.com/go/speech/apiv1/speechpb"
	"github.com/faiface/beep"
	"github.com/faiface/beep/mp3"
	"github.com/faiface/beep/speaker"
	"github.com/faiface/beep/wav"
	"github.com/gen2brain/malgo"
	ttsengine "github.com/hegedustibor/htgo-tts"
	"github.com/sashabaranov/go-openai"
	"github.com/ztrue/tracerr"
)

// VoiceManager управляет голосовыми функциями
type VoiceManager struct {
	config         VoiceConfig
	context        *malgo.Context
	device         *malgo.Device
	captureConfig  malgo.DeviceConfig
	captureChan    chan []byte
	isListening    bool
	isProcessing   bool
	mutex          sync.Mutex
	openAIClient   *openai.Client
	googleClient   *speech.Client
	tts            *ttsengine.Speech
	wakeWordActive bool
	callbacks      struct {
		onCommand func(string)
	}
}

// VoiceConfig содержит настройки голосового модуля
type VoiceConfig struct {
	Enabled           bool    `json:"enabled"`
	WakeWord          string  `json:"wake_word"`
	Language          string  `json:"language"`
	VoiceRecognition  string  `json:"voice_recognition"` // google, local, whisper
	TTSProvider       string  `json:"tts_provider"`      // google, local
	VoiceThreshold    float64 `json:"voice_threshold"`
	SilenceThreshold  float64 `json:"silence_threshold"`
	InputDevice       string  `json:"input_device"`
	OutputDevice      string  `json:"output_device"`
	OpenAIAPIKey      string  `json:"openai_api_key"`
	GoogleAPIKey      string  `json:"google_api_key"`
}

// NewVoiceManager создает новый экземпляр VoiceManager
func NewVoiceManager(config VoiceConfig) *VoiceManager {
	return &VoiceManager{
		config:         config,
		captureChan:    make(chan []byte, 10),
		isListening:    false,
		isProcessing:   false,
		wakeWordActive: false,
	}
}

// Start запускает голосовой модуль
func (vm *VoiceManager) Start() error {
	if !vm.config.Enabled {
		log.Println("Голосовой модуль отключен в настройках")
		return nil
	}

	// Инициализация OpenAI клиента, если используется Whisper
	if vm.config.VoiceRecognition == "whisper" && vm.config.OpenAIAPIKey != "" {
		vm.openAIClient = openai.NewClient(vm.config.OpenAIAPIKey)
	}

	// Инициализация Google Speech клиента, если используется Google
	if (vm.config.VoiceRecognition == "google" || vm.config.TTSProvider == "google") && vm.config.GoogleAPIKey != "" {
		ctx := context.Background()
		client, err := speech.NewClient(ctx)
		if err != nil {
			return tracerr.Wrap(err)
		}
		vm.googleClient = client
	}

	// Инициализация TTS
	vm.tts = &ttsengine.Speech{
		Folder:   "audio",
		Language: vm.config.Language,
	}

	// Инициализация аудио контекста
	ctx, err := malgo.NewContext(nil, malgo.ContextConfig{}, func(message string) {
		log.Printf("[Audio Log] %s\n", message)
	})
	if err != nil {
		return tracerr.Wrap(err)
	}
	vm.context = ctx

	// Настройка захвата аудио
	vm.captureConfig = malgo.DefaultDeviceConfig(malgo.Capture)
	vm.captureConfig.Capture.Format = malgo.FormatS16
	vm.captureConfig.Capture.Channels = 1
	vm.captureConfig.SampleRate = 16000
	vm.captureConfig.Alsa.NoMMap = 1

	// Создание устройства захвата
	deviceCallbacks := malgo.DeviceCallbacks{
		Data: vm.onAudioData,
	}
	device, err := malgo.InitDevice(vm.context.Context, vm.captureConfig, &deviceCallbacks)
	if err != nil {
		vm.context.Close()
		return tracerr.Wrap(err)
	}
	vm.device = device

	// Запуск устройства
	if err := vm.device.Start(); err != nil {
		vm.device.Uninit()
		vm.context.Close()
		return tracerr.Wrap(err)
	}

	// Запуск обработки аудио в отдельной горутине
	go vm.processAudio()

	return nil
}

// Stop останавливает голосовой модуль
func (vm *VoiceManager) Stop() {
	vm.mutex.Lock()
	defer vm.mutex.Unlock()

	if vm.device != nil {
		vm.device.Stop()
		vm.device.Uninit()
		vm.device = nil
	}

	if vm.context != nil {
		vm.context.Close()
		vm.context = nil
	}

	if vm.googleClient != nil {
		vm.googleClient.Close()
		vm.googleClient = nil
	}

	vm.isListening = false
	vm.isProcessing = false
}

// SetCommandCallback устанавливает функцию обратного вызова для команд
func (vm *VoiceManager) SetCommandCallback(callback func(string)) {
	vm.callbacks.onCommand = callback
}

// Speak произносит текст
func (vm *VoiceManager) Speak(text string) error {
	if !vm.config.Enabled {
		return nil
	}

	switch vm.config.TTSProvider {
	case "google":
		return vm.speakGoogle(text)
	case "local":
		return vm.speakLocal(text)
	default:
		return vm.speakLocal(text)
	}
}

// speakGoogle использует Google TTS для произнесения текста
func (vm *VoiceManager) speakGoogle(text string) error {
	// Используем локальный TTS, если Google недоступен
	return vm.tts.Speak(text)
}

// speakLocal использует локальный TTS для произнесения текста
func (vm *VoiceManager) speakLocal(text string) error {
	return vm.tts.Speak(text)
}

// onAudioData обрабатывает входящие аудио данные
func (vm *VoiceManager) onAudioData(pOutputSample, pInputSample []byte, framecount uint32) {
	if !vm.isListening {
		return
	}

	// Копируем данные, чтобы избежать проблем с памятью
	data := make([]byte, len(pInputSample))
	copy(data, pInputSample)

	// Отправляем данные в канал для обработки
	select {
	case vm.captureChan <- data:
		// Данные успешно отправлены
	default:
		// Канал заполнен, пропускаем данные
	}
}

// processAudio обрабатывает аудио данные в отдельной горутине
func (vm *VoiceManager) processAudio() {
	vm.isListening = true

	var buffer []byte
	var silenceCounter int
	var isRecording bool

	for vm.isListening {
		data := <-vm.captureChan

		// Проверяем уровень громкости
		level := vm.calculateAudioLevel(data)

		if !isRecording && level > vm.config.VoiceThreshold {
			// Начинаем запись
			isRecording = true
			buffer = nil
			silenceCounter = 0
		}

		if isRecording {
			// Добавляем данные в буфер
			buffer = append(buffer, data...)

			if level < vm.config.SilenceThreshold {
				silenceCounter++
			} else {
				silenceCounter = 0
			}

			// Если тишина длится достаточно долго, обрабатываем запись
			if silenceCounter > 30 { // примерно 1 секунда тишины
				isRecording = false

				// Обрабатываем запись в отдельной горутине
				go func(audioData []byte) {
					vm.mutex.Lock()
					if vm.isProcessing {
						vm.mutex.Unlock()
						return
					}
					vm.isProcessing = true
					vm.mutex.Unlock()

					defer func() {
						vm.mutex.Lock()
						vm.isProcessing = false
						vm.mutex.Unlock()
					}()

					// Распознаем речь
					text, err := vm.recognizeSpeech(audioData)
					if err != nil {
						log.Printf("Ошибка распознавания речи: %v", err)
						return
					}

					if text == "" {
						return
					}

					text = strings.ToLower(text)
					log.Printf("Распознано: %s", text)

					// Проверяем наличие ключевого слова
					if !vm.wakeWordActive {
						if strings.Contains(text, strings.ToLower(vm.config.WakeWord)) {
							vm.wakeWordActive = true
							vm.Speak("Слушаю")
						}
						return
					}

					// Обрабатываем команду
					if vm.callbacks.onCommand != nil {
						vm.callbacks.onCommand(text)
						vm.wakeWordActive = false // Сбрасываем активацию после выполнения команды
					}
				}(buffer)
			}
		}
	}
}

// calculateAudioLevel вычисляет уровень громкости аудио
func (vm *VoiceManager) calculateAudioLevel(data []byte) float64 {
	if len(data) == 0 {
		return 0
	}

	var sum float64
	for i := 0; i < len(data); i += 2 {
		if i+1 >= len(data) {
			break
		}

		// Преобразуем два байта в 16-битное значение
		sample := int16(data[i]) | (int16(data[i+1]) << 8)
		value := float64(sample) / 32768.0 // Нормализуем до [-1, 1]
		sum += value * value
	}

	return sum / float64(len(data)/2)
}

// recognizeSpeech распознает речь из аудио данных
func (vm *VoiceManager) recognizeSpeech(audioData []byte) (string, error) {
	switch vm.config.VoiceRecognition {
	case "whisper":
		return vm.recognizeWithWhisper(audioData)
	case "google":
		return vm.recognizeWithGoogle(audioData)
	default:
		return vm.recognizeWithWhisper(audioData)
	}
}

// recognizeWithWhisper распознает речь с помощью OpenAI Whisper API
func (vm *VoiceManager) recognizeWithWhisper(audioData []byte) (string, error) {
	if vm.openAIClient == nil {
		return "", tracerr.New("OpenAI клиент не инициализирован")
	}

	// Сохраняем аудио во временный файл
	tmpDir := os.TempDir()
	tmpFile := filepath.Join(tmpDir, fmt.Sprintf("kot_audio_%d.wav", time.Now().UnixNano()))
	defer os.Remove(tmpFile)

	// Преобразуем аудио в WAV формат и сохраняем
	if err := vm.saveAudioAsWAV(audioData, tmpFile); err != nil {
		return "", tracerr.Wrap(err)
	}

	// Открываем файл для отправки
	file, err := os.Open(tmpFile)
	if err != nil {
		return "", tracerr.Wrap(err)
	}
	defer file.Close()

	// Отправляем запрос в OpenAI Whisper API
	resp, err := vm.openAIClient.CreateTranscription(
		context.Background(),
		openai.AudioRequest{
				Model:    openai.Whisper1,
				Reader:   file,
			},
	)
	if err != nil {
		return "", tracerr.Wrap(err)
	}

	return resp.Text, nil
}

// recognizeWithGoogle распознает речь с помощью Google Speech-to-Text API
func (vm *VoiceManager) recognizeWithGoogle(audioData []byte) (string, error) {
	if vm.googleClient == nil {
		return "", tracerr.New("Google Speech клиент не инициализирован")
	}

	// Создаем контекст с таймаутом
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()

	// Отправляем запрос в Google Speech-to-Text API
	resp, err := vm.googleClient.Recognize(ctx, &speechpb.RecognizeRequest{
		Config: &speechpb.RecognitionConfig{
			Encoding:        speechpb.RecognitionConfig_LINEAR16,
			SampleRateHertz: 16000,
			LanguageCode:    vm.config.Language,
		},
		Audio: &speechpb.RecognitionAudio{
			AudioSource: &speechpb.RecognitionAudio_Content{Content: audioData},
		},
	})
	if err != nil {
		return "", tracerr.Wrap(err)
	}

	if len(resp.Results) == 0 || len(resp.Results[0].Alternatives) == 0 {
		return "", nil
	}

	return resp.Results[0].Alternatives[0].Transcript, nil
}

// saveAudioAsWAV сохраняет аудио данные в WAV формате
func (vm *VoiceManager) saveAudioAsWAV(audioData []byte, filePath string) error {
	// Создаем WAV заголовок
	header := make([]byte, 44)

	// "RIFF" chunk descriptor
	copy(header[0:4], []byte("RIFF"))

	// Размер файла - 8
	fileSize := uint32(len(audioData) + 36)
	header[4] = byte(fileSize)
	header[5] = byte(fileSize >> 8)
	header[6] = byte(fileSize >> 16)
	header[7] = byte(fileSize >> 24)

	// "WAVE" формат
	copy(header[8:12], []byte("WAVE"))

	// "fmt " subchunk
	copy(header[12:16], []byte("fmt "))

	// Размер subchunk (16 для PCM)
	header[16] = 16
	header[17] = 0
	header[18] = 0
	header[19] = 0

	// Аудио формат (1 для PCM)
	header[20] = 1
	header[21] = 0

	// Количество каналов (1 для моно)
	header[22] = 1
	header[23] = 0

	// Частота дискретизации (16000 Hz)
	sampleRate := uint32(16000)
	header[24] = byte(sampleRate)
	header[25] = byte(sampleRate >> 8)
	header[26] = byte(sampleRate >> 16)
	header[27] = byte(sampleRate >> 24)

	// Байт в секунду (16000 * 2 = 32000)
	byteRate := uint32(32000)
	header[28] = byte(byteRate)
	header[29] = byte(byteRate >> 8)
	header[30] = byte(byteRate >> 16)
	header[31] = byte(byteRate >> 24)

	// Выравнивание блока (2 байта)
	header[32] = 2
	header[33] = 0

	// Бит на сэмпл (16 бит)
	header[34] = 16
	header[35] = 0

	// "data" subchunk
	copy(header[36:40], []byte("data"))

	// Размер данных
	dataSize := uint32(len(audioData))
	header[40] = byte(dataSize)
	header[41] = byte(dataSize >> 8)
	header[42] = byte(dataSize >> 16)
	header[43] = byte(dataSize >> 24)

	// Создаем файл
	file, err := os.Create(filePath)
	if err != nil {
		return tracerr.Wrap(err)
	}
	defer file.Close()

	// Записываем заголовок и данные
	if _, err := file.Write(header); err != nil {
		return tracerr.Wrap(err)
	}

	if _, err := file.Write(audioData); err != nil {
		return tracerr.Wrap(err)
	}

	return nil
}

// PlayAudioFile воспроизводит аудио файл
func (vm *VoiceManager) PlayAudioFile(filePath string) error {
	// Открываем файл
	file, err := os.Open(filePath)
	if err != nil {
		return tracerr.Wrap(err)
	}
	defer file.Close()

	// Определяем тип файла по расширению
	switch filepath.Ext(filePath) {
	case ".mp3":
		return vm.playMP3(file)
	case ".wav":
		return vm.playWAV(file)
	default:
		return tracerr.New("Неподдерживаемый формат аудио")
	}
}

// playMP3 воспроизводит MP3 файл
func (vm *VoiceManager) playMP3(file io.ReadCloser) error {
	// Декодируем MP3
	streamer, format, err := mp3.Decode(file)
	if err != nil {
		return tracerr.Wrap(err)
	}
	defer streamer.Close()

	// Инициализируем динамик
	speaker.Init(format.SampleRate, format.SampleRate.N(time.Second/10))

	// Воспроизводим
	done := make(chan bool)
	speaker.Play(beep.Seq(streamer, beep.Callback(func() {
		done <- true
	})))

	// Ждем завершения воспроизведения
	<-done

	return nil
}

// playWAV воспроизводит WAV файл
func (vm *VoiceManager) playWAV(file io.ReadCloser) error {
	// Декодируем WAV
	streamer, format, err := wav.Decode(file)
	if err != nil {
		return tracerr.Wrap(err)
	}
	defer streamer.Close()

	// Инициализируем динамик
	speaker.Init(format.SampleRate, format.SampleRate.N(time.Second/10))

	// Воспроизводим
	done := make(chan bool)
	speaker.Play(beep.Seq(streamer, beep.Callback(func() {
		done <- true
	})))

	// Ждем завершения воспроизведения
	<-done

	return nil
}