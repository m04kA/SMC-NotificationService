package templates

import "fmt"

const (
	// WelcomeMessageText приветственное сообщение при команде /start
	WelcomeMessageText = `Добро пожаловать!

Для удобного доступа к нашему сервису, вы можете создать иконку приложения на главном экране вашего устройства.
1. Нажать на три точки в правом верхнем углу и в выпадающем меню нажать «Добавить на экран домой»
2. В открывшейся страницы нажать на значок поделиться
3. Промотать всплывающее меню и нажать на кнопку «На экран Домой»
`

	// WelcomeButtonText текст кнопки в приветственном сообщении
	WelcomeButtonText = "Открыть приложение"

	// WelcomeButtonBaseURL базовый URL кнопки в приветственном сообщении (без query параметров)
	WelcomeButtonBaseURL = "https://faberon24.vercel.app/index.html"
)

// GetWelcomeButtonURL возвращает URL кнопки с подставленным tgUserId
func GetWelcomeButtonURL(tgUserID int64) string {
	return fmt.Sprintf("%s?X-UserID=%d", WelcomeButtonBaseURL, tgUserID)
}
