package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"

	symspell "symspell/pkg"
	"symspell/pkg/options"
	"symspell/pkg/verbosity"
)

func main() {
	// Настройки SymSpell
	maxEditDistance := 2

	spellChecker := symspell.NewSymSpell(
		options.WithMaxDictionaryEditDistance(maxEditDistance),
		options.WithPrefixLength(4),
		options.WithCountThreshold(1),
		options.WithSmartFrequencyCorrection(),
	)

	dictionaryPath := "en_full.txt"
	fmt.Printf("Загружаем словарь из файла: %s\n", dictionaryPath)

	ok, err := spellChecker.LoadDictionary(dictionaryPath, 0, 1, " ")
	if err != nil {
		log.Fatalf("Ошибка при загрузке словаря: %v", err)
	}
	if !ok {
		log.Fatal("Не удалось загрузить словарь")
	}
	spellChecker.ClearTransformData()

	fmt.Println("Словарь успешно загружен!")
	fmt.Println("Введите слова для проверки (каждое слово или фразу на отдельной строке).")
	fmt.Println("Для выхода введите 'quit' или нажмите Ctrl+C")
	fmt.Println()

	scanner := bufio.NewScanner(os.Stdin)

	for {
		fmt.Print("Введите слова: ")
		if !scanner.Scan() {
			break
		}

		input := strings.TrimSpace(scanner.Text())
		if input == "" {
			continue
		}
		if strings.ToLower(input) == "quit" {
			break
		}

		correctedWords := correctWords(spellChecker, input, maxEditDistance)

		fmt.Printf("Исходный текст: %s\n", input)
		fmt.Printf("Исправленный:   %s\n", correctedWords)
		fmt.Println("---")
	}

	if err := scanner.Err(); err != nil {
		log.Printf("Ошибка при чтении ввода: %v", err)
	}
}

// correctWords исправляет слова в строке
func correctWords(spellChecker symspell.SymSpell, input string, maxEditDistance int) string {
	// Сначала пытаемся исправить всю фразу как составное слово
	if strings.Contains(input, " ") {
		compoundResult := spellChecker.LookupCompound(input, maxEditDistance)
		if compoundResult != nil && compoundResult.Distance <= maxEditDistance {
			return compoundResult.Term
		}
	}

	// Если составное исправление не подходит, исправляем каждое слово отдельно
	words := strings.Fields(input)
	correctedWords := make([]string, 0, len(words))

	for _, word := range words {
		correctedWord := correctSingleWord(spellChecker, word, maxEditDistance)
		correctedWords = append(correctedWords, correctedWord)
	}

	return strings.Join(correctedWords, " ")
}

// correctSingleWord исправляет одно слово
func correctSingleWord(spellChecker symspell.SymSpell, word string, maxEditDistance int) string {
	// Используем verbosity.Top для получения лучшего варианта
	suggestions, err := spellChecker.Lookup(word, verbosity.Top, maxEditDistance)
	if err != nil {
		log.Printf("Ошибка при поиске исправлений для '%s': %v", word, err)
		return word
	}

	// Если нет исправлений, возвращаем исходное слово
	if len(suggestions) == 0 {
		return word
	}

	// Возвращаем лучшее исправление
	bestSuggestion := suggestions[0]

	// Если расстояние редактирования 0, значит слово уже правильное
	if bestSuggestion.Distance == 0 {
		return word
	}

	return bestSuggestion.Term
}

// createSampleDictionary создает пример файла словаря
func createSampleDictionary() {
	fmt.Println("Создание примера словаря...")

	sampleWords := []string{
		"привет 1000",
		"мир 800",
		"программирование 500",
		"компьютер 750",
		"код 600",
		"разработка 450",
		"алгоритм 300",
		"данные 900",
		"система 850",
		"проект 400",
		"функция 350",
		"переменная 250",
		"массив 200",
		"строка 300",
		"число 400",
		"файл 500",
		"папка 150",
		"документ 300",
		"текст 450",
		"слово 600",
	}

	file, err := os.Create("dictionary.txt")
	if err != nil {
		log.Printf("Ошибка создания файла словаря: %v", err)
		return
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	for _, word := range sampleWords {
		writer.WriteString(word + "\n")
	}
	writer.Flush()

	fmt.Println("Пример словаря создан в файле dictionary.txt")
}

func init() {
	// Проверяем, существует ли файл словаря
	if _, err := os.Stat("dictionary.txt"); os.IsNotExist(err) {
		fmt.Println("Файл словаря не найден. Создаем пример...")
		createSampleDictionary()
		fmt.Println()
	}
}
