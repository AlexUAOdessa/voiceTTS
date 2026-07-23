package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/difyz9/edge-tts-go/pkg/communicate"
)

const MaxChunkSize = 4000

// Голоса по умолчанию
var voiceMap = map[string]string{
	"RU": "ru-RU-DmitryNeural",
	"EN": "en-US-AndrewNeural",
	"UA": "uk-UA-OstapNeural",
}

// Рекомендуемые голоса
var availableVoices = map[string][]string{
	"RU": {"ru-RU-DmitryNeural (Мужской, по умолчанию)", "ru-RU-SvetlanaNeural (Женский)"},
	"EN": {"en-US-AndrewNeural (Мужской, по умолчанию)", "en-US-AvaNeural (Женский)"},
	"UA": {"uk-UA-OstapNeural (Мужской, по умолчанию)", "uk-UA-PolinaNeural (Женский)"},
}

func main() {
	var customVoice string
	var listVoices bool

	flag.StringVar(&customVoice, "voice", "", "Указать кастомный голос")
	flag.StringVar(&customVoice, "v", "", "Указать кастомный голос (коротко)")

	flag.BoolVar(&listVoices, "list-voices", false, "Показать список голосов")
	flag.BoolVar(&listVoices, "vl", false, "Показать список голосов (коротко)")

	flag.Usage = showHelp
	flag.CommandLine.Init(os.Args[0], flag.ContinueOnError)

	flag.Parse()

	// Обработка помощи
	if isHelpRequested() {
		showHelp()
		return
	}

	if listVoices {
		showVoiceList()
		return
	}

	// Файл обязательно должен быть указан
	args := flag.Args()
	if len(args) == 0 {
		fmt.Println("Ошибка: Не указан входной файл.\n")
		showHelp()
		return
	}

	requestedFile := args[0]

	inputFile, voice, lang := resolveInputAndVoice(requestedFile, customVoice)
	if inputFile == "" || voice == "" {
		showHelp()
		return
	}

	// === Основная работа ===
	outputFile := "output_" + strings.ToLower(lang) + ".mp3"

	content, err := os.ReadFile(inputFile)
	if err != nil {
		log.Fatalf("Ошибка чтения файла %s: %v", inputFile, err)
	}

	text := strings.TrimSpace(string(content))
	if len(text) == 0 {
		log.Fatalf("Файл %s пуст!", inputFile)
	}

	chunks := splitText(text, MaxChunkSize)
	fmt.Printf("Файл: %s | Язык: %s | Голос: %s\n", inputFile, lang, voice)
	fmt.Printf("Текст прочитан (%d символов). Разбито на %d частей.\n", len(text), len(chunks))

	outFile, err := os.Create(outputFile)
	if err != nil {
		log.Fatalf("Не удалось создать выходной файл: %v", err)
	}
	defer outFile.Close()

	ctx := context.Background()

	for i, chunk := range chunks {
		fmt.Printf("[%d/%d] Озвучка фрагмента...\n", i+1, len(chunks))

		comm, err := communicate.NewCommunicate(chunk, voice, "+0%", "+0%", "+0Hz", "", 10, 60)
		if err != nil {
			log.Fatalf("Ошибка инициализации на шаге %d: %v", i+1, err)
		}

		chunkChan, errChan := comm.Stream(ctx)

		for chunkData := range chunkChan {
			if len(chunkData.Data) > 0 {
				_, err := outFile.Write(chunkData.Data)
				if err != nil {
					log.Fatalf("Ошибка записи в MP3: %v", err)
				}
			}
		}

		if err := <-errChan; err != nil {
			log.Fatalf("Ошибка синтеза на шаге %d: %v", i+1, err)
		}
	}

	fmt.Printf("\nГотово! Аудио сохранено в файл: %s\n", outputFile)
}

func isHelpRequested() bool {
	for _, arg := range os.Args[1:] {
		a := strings.ToLower(strings.TrimLeft(arg, "-"))
		if a == "h" || a == "help" {
			return true
		}
	}
	return false
}

func resolveInputAndVoice(requestedFile, customVoice string) (inputFile, voice, lang string) {
	inputFile = requestedFile

	if _, err := os.Stat(inputFile); err != nil {
		fmt.Printf("Файл не найден: %s\n\n", inputFile)
		return "", "", ""
	}

	name := strings.ToUpper(filepath.Base(inputFile))
	if strings.Contains(name, "EN") {
		lang = "EN"
	} else if strings.Contains(name, "UA") {
		lang = "UA"
	} else {
		lang = "RU"
	}

	if customVoice != "" {
		voice = customVoice
	} else {
		voice = voiceMap[lang]
	}

	return inputFile, voice, lang
}

func showVoiceList() {
	fmt.Println("=== Рекомендуемые голоса ===")
	for lang, voices := range availableVoices {
		fmt.Printf("\n[%s]:\n", lang)
		for _, v := range voices {
			fmt.Printf("  • %s\n", v)
		}
	}
	fmt.Println("\nПрограмма использует голос по умолчанию, если не указан через -v.")
}

func showHelp() {
	fmt.Println("Программа переводит TXT-файл в MP3 с помощью Edge TTS.")
	fmt.Println()
	fmt.Println("Использование: voiceEdge.exe [флаги] <inputФайл.txt>")
	fmt.Println()
	fmt.Println("Флаги:")
	fmt.Println("  -h, --help              Показать эту справку")
	fmt.Println("  -vl, -list-voices       Показать список доступных голосов")
	fmt.Println("  -v, -voice <голос>      Указать конкретный голос")
	fmt.Println()
	fmt.Println("Примеры:")
	fmt.Println("  voiceEdge.exe inputRU.txt")
	fmt.Println("  voiceEdge.exe inputEN.txt")
	fmt.Println("  voiceEdge.exe -v ru-RU-SvetlanaNeural inputRU.txt")
	fmt.Println("  voiceEdge.exe -vl")
	fmt.Println("  voiceEdge.exe -h")
	fmt.Println()
	fmt.Println("Поддерживаемые имена файлов:")
	fmt.Println("  inputEN.txt  → английский (по умолчанию AndrewNeural)")
	fmt.Println("  inputUA.txt  → украинский (по умолчанию OstapNeural)")
	fmt.Println("  inputRU.txt  → русский (по умолчанию DmitryNeural)")
}

func splitText(text string, maxLen int) []string {
	var chunks []string
	runes := []rune(text)
	for len(runes) > 0 {
		if len(runes) <= maxLen {
			chunks = append(chunks, string(runes))
			break
		}
		splitIdx := maxLen
		for i := maxLen; i > maxLen-1000 && i > 0; i-- {
			if runes[i] == '.' || runes[i] == '!' || runes[i] == '?' || runes[i] == '\n' {
				splitIdx = i + 1
				break
			}
		}
		chunks = append(chunks, string(runes[:splitIdx]))
		runes = runes[splitIdx:]
	}
	return chunks
}
