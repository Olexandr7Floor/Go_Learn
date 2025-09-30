package main

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

// BookData представляє дані про один файл (книгу), які будуть відправлені на фронтенд.
type BookData struct {
	DisplayName string `json:"displayName"`
	FileName    string `json:"fileName"`
	URL         string `json:"url"`
}

const (
	literatureDir = "Література" // Назва папки з літературою
	serverPort    = ":8080"      // Порт, на якому буде працювати сервер
	staticDir     = "static"     // Папка для index.html, reader.html, JS/CSS та PDF-viewer
)

func main() {
	// 1. Подача статичних файлів (HTML, CSS, JS) з папки 'static'
	// Це дозволить браузеру завантажувати index.html (за замовчуванням при "/"),
	// reader.html, а також усі ресурси всередині static/ (включаючи js/pdfjs_viewer/).
	http.Handle("/", http.FileServer(http.Dir(staticDir)))

	// 2. Подача PDF-файлів з папки 'Література'
	// Фронтенд звертається до цих файлів через /literature/назва_файлу.pdf
	http.Handle("/literature/", http.StripPrefix("/literature/", http.FileServer(http.Dir(literatureDir))))

	// 3. API для отримання списку літератури у форматі JSON
	http.HandleFunc("/api/literature", literatureAPIHandler)

	// 4. API для Go-компілятора (проксі до Go Playground)
	http.HandleFunc("/api/run-code", runCodeProxyHandler)

	log.Printf("Сервер запущено на http://localhost%s", serverPort)
	log.Printf("Відкрийте http://localhost%s/index.html", serverPort)
	log.Fatal(http.ListenAndServe(serverPort, nil))
}

// literatureAPIHandler читає папку "Література" і повертає список файлів у JSON.
func literatureAPIHandler(w http.ResponseWriter, r *http.Request) {
	// Встановлюємо CORS-заголовки для уникнення проблем з fetch на деяких конфігураціях.
	w.Header().Set("Access-Control-Allow-Origin", "*")

	files, err := os.ReadDir(literatureDir)
	if err != nil {
		// Якщо папка не знайдена або є помилка, повертаємо 500
		http.Error(w, "Помилка читання папки: "+err.Error(), http.StatusInternalServerError)
		return
	}

	var books []BookData
	for _, file := range files {
		if file.IsDir() || strings.HasPrefix(file.Name(), ".") {
			continue // Ігноруємо папки та приховані файли
		}

		fileName := file.Name()
		displayName := cleanFileName(fileName)

		// Створюємо URL для прямого доступу до файлу, кодуючи його назву
		// Фронтенд використовуватиме цей URL всередині iframe
		url := "/literature/" + url.PathEscape(fileName)

		books = append(books, BookData{
			DisplayName: displayName,
			FileName:    fileName,
			URL:         url,
		})
	}

	// Встановлюємо заголовок і кодуємо список книг у JSON
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(books)
}

// runCodeProxyHandler проксує запит від фронтенду до Go Playground API.
func runCodeProxyHandler(w http.ResponseWriter, r *http.Request) {
	// Додаємо заголовки для дозволу міждоменних запитів
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
    w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")

	// 1. Обробка OPTIONS запиту (Pre-flight request)
    if r.Method == http.MethodOptions {
        w.WriteHeader(http.StatusOK)
        return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Підтримується тільки POST-запит", http.StatusMethodNotAllowed)
		return
	}

	// URL Go Playground для виконання коду
	playgroundURL := "https://play.golang.org/compile"

	// Створення нового запиту до Playground, використовуючи тіло запиту клієнта
	req, err := http.NewRequest(http.MethodPost, playgroundURL, r.Body)
	if err != nil {
		http.Error(w, "Помилка створення запиту", http.StatusInternalServerError)
		return
	}

	// Важливо: копіюємо Content-Type для коректної роботи API Go Playground
	req.Header.Set("Content-Type", r.Header.Get("Content-Type"))

	// Виконання запиту
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		http.Error(w, "Помилка зв'язку з Go Playground: "+err.Error(), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	// Копіювання заголовків та тіла відповіді від Playground назад клієнту
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

// cleanFileName очищає ім'я файлу для кращого відображення.
func cleanFileName(name string) string {
	nameWithoutExt := strings.TrimSuffix(name, filepath.Ext(name))
	name = strings.ReplaceAll(nameWithoutExt, "_", " ")
	name = strings.ReplaceAll(name, "-", " ")

	// Видаляємо префікси з цифрами і пробілом (наприклад, "1 " або "2_")
	if len(name) > 2 && name[1] == ' ' && name[0] >= '0' && name[0] <= '9' {
		name = name[2:]
	}
	return name
}
