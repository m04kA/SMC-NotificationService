package telegram

import (
	"fmt"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/m04kA/SMC-NotificationService/internal/domain"
	"github.com/m04kA/SMC-NotificationService/internal/service/telegram/templates"
)

// Service сервис для отправки сообщений через Telegram Bot API
type Service struct {
	bot BotAPI
}

// NewService создает новый экземпляр Telegram сервиса
func NewService(bot BotAPI) *Service {
	return &Service{
		bot: bot,
	}
}

// SendMessage отправляет уведомление через Telegram Bot API
// Автоматически определяет тип отправки (текст, фото, media group)
func (s *Service) SendMessage(msg *domain.TelegramMessage) error {
	if msg.ChatID == 0 {
		return ErrInvalidChatID
	}

	if msg.MessageText == "" {
		return ErrEmptyMessage
	}

	// Если есть изображения
	if msg.HasImages() {
		// Если изображений > 1 - отправляем как MediaGroup
		if msg.IsMediaGroup() {
			return s.sendMediaGroup(msg)
		}
		// Одно изображение - отправляем как фото с caption
		return s.sendPhoto(msg)
	}

	// Текстовое сообщение с кнопками
	return s.sendTextMessage(msg)
}

// sendTextMessage отправляет текстовое сообщение
func (s *Service) sendTextMessage(msg *domain.TelegramMessage) error {
	tgMsg := tgbotapi.NewMessage(msg.ChatID, msg.MessageText)
	tgMsg.ParseMode = msg.ParseMode

	// Добавляем inline-кнопки если есть
	if msg.HasButtons() {
		tgMsg.ReplyMarkup = s.buildInlineKeyboard(msg.InlineButtons)
	}

	_, err := s.bot.Send(tgMsg)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrSendMessage, err)
	}

	return nil
}

// sendPhoto отправляет фото с caption и кнопками
func (s *Service) sendPhoto(msg *domain.TelegramMessage) error {
	if len(msg.ImageURLs) == 0 {
		return fmt.Errorf("%w: no images provided", ErrSendPhoto)
	}

	photo := tgbotapi.NewPhoto(msg.ChatID, tgbotapi.FileURL(msg.ImageURLs[0]))
	photo.Caption = msg.MessageText
	photo.ParseMode = msg.ParseMode

	// Добавляем inline-кнопки если есть
	if msg.HasButtons() {
		photo.ReplyMarkup = s.buildInlineKeyboard(msg.InlineButtons)
	}

	_, err := s.bot.Send(photo)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrSendPhoto, err)
	}

	return nil
}

// sendMediaGroup отправляет группу изображений (2-10 штук)
// Кнопки добавляются к последнему изображению
func (s *Service) sendMediaGroup(msg *domain.TelegramMessage) error {
	if len(msg.ImageURLs) < 2 {
		return fmt.Errorf("%w: media group requires at least 2 images", ErrSendMediaGroup)
	}

	if len(msg.ImageURLs) > 10 {
		return fmt.Errorf("%w: media group supports maximum 10 images", ErrSendMediaGroup)
	}

	var mediaGroup []interface{}

	// Первое изображение с текстом
	firstPhoto := tgbotapi.NewInputMediaPhoto(tgbotapi.FileURL(msg.ImageURLs[0]))
	firstPhoto.Caption = msg.MessageText
	firstPhoto.ParseMode = msg.ParseMode
	mediaGroup = append(mediaGroup, firstPhoto)

	// Остальные изображения без текста
	for i := 1; i < len(msg.ImageURLs); i++ {
		photo := tgbotapi.NewInputMediaPhoto(tgbotapi.FileURL(msg.ImageURLs[i]))
		mediaGroup = append(mediaGroup, photo)
	}

	mediaGroupConfig := tgbotapi.NewMediaGroup(msg.ChatID, mediaGroup)

	// Используем Request вместо Send, так как MediaGroup возвращает массив сообщений
	resp, err := s.bot.Request(mediaGroupConfig)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrSendMediaGroup, err)
	}

	// Проверяем успешность отправки
	if !resp.Ok {
		return fmt.Errorf("%w: telegram API error: %s", ErrSendMediaGroup, resp.Description)
	}

	// Если есть кнопки - отправляем отдельным сообщением
	// Telegram не поддерживает inline-кнопки в MediaGroup
	if msg.HasButtons() {
		buttonMsg := tgbotapi.NewMessage(msg.ChatID, "⬆️") // Стрелка вверх указывает на MediaGroup
		buttonMsg.ReplyMarkup = s.buildInlineKeyboard(msg.InlineButtons)

		_, err := s.bot.Send(buttonMsg)
		if err != nil {
			// Не критично - MediaGroup уже отправлена
			return fmt.Errorf("%w: media group sent but buttons failed: %v", ErrSendMessage, err)
		}
	}

	return nil
}

// buildInlineKeyboard создает inline-клавиатуру из массива кнопок
func (s *Service) buildInlineKeyboard(buttons []domain.InlineButton) tgbotapi.InlineKeyboardMarkup {
	var rows [][]tgbotapi.InlineKeyboardButton

	// Каждая кнопка на отдельной строке
	for _, btn := range buttons {
		row := tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonURL(btn.Text, btn.URL),
		)
		rows = append(rows, row)
	}

	return tgbotapi.NewInlineKeyboardMarkup(rows...)
}

// SendWelcomeMessage отправляет приветственное сообщение при команде /start
// Отправляет медиагруппу из 3 изображений с текстом и кнопку в отдельном сообщении
// tgUserID опционален - если передан nil, используется дефолтный URL без параметра
func (s *Service) SendWelcomeMessage(chatID int64, tgUserID *int64) error {
	// Пути к изображениям в правильном порядке
	imageFiles := []string{
		"./static/welcome/Step1.PNG",
		"./static/welcome/Step2.PNG",
		"./static/welcome/Step3.PNG",
		"./static/welcome/Step4.PNG",
		"./static/welcome/Step5.PNG",
	}

	// Создаем медиагруппу
	var mediaGroup []interface{}

	// Первое изображение с текстом приветствия
	firstPhoto := tgbotapi.NewInputMediaPhoto(tgbotapi.FilePath(imageFiles[0]))
	firstPhoto.Caption = templates.WelcomeMessageText
	mediaGroup = append(mediaGroup, firstPhoto)

	// Остальные изображения без текста
	for i := 1; i < len(imageFiles); i++ {
		photo := tgbotapi.NewInputMediaPhoto(tgbotapi.FilePath(imageFiles[i]))
		mediaGroup = append(mediaGroup, photo)
	}

	// Отправляем медиагруппу
	mediaGroupConfig := tgbotapi.NewMediaGroup(chatID, mediaGroup)

	// Используем Request вместо Send, так как MediaGroup возвращает массив сообщений
	resp, err := s.bot.Request(mediaGroupConfig)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrSendMediaGroup, err)
	}

	// Проверяем успешность отправки
	if !resp.Ok {
		return fmt.Errorf("%w: telegram API error: %s", ErrSendMediaGroup, resp.Description)
	}

	// Формируем URL кнопки
	buttonURL := templates.WelcomeButtonBaseURL
	if tgUserID != nil {
		buttonURL = templates.GetWelcomeButtonURL(*tgUserID)
	}

	// Отправляем кнопку отдельным сообщением (Telegram не поддерживает inline-кнопки в MediaGroup)
	buttonMsg := tgbotapi.NewMessage(chatID, "Нажмите на кнопку ниже, чтобы открыть приложение:")
	buttonMsg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonWebApp(templates.WelcomeButtonText, tgbotapi.WebAppInfo{
				URL: buttonURL,
			}),
		),
	)

	_, err = s.bot.Send(buttonMsg)
	if err != nil {
		// Медиагруппа уже отправлена, ошибка кнопки не критична
		return fmt.Errorf("%w: media group sent but button failed: %v", ErrSendMessage, err)
	}

	return nil
}

// SetWebhook устанавливает webhook URL для получения обновлений от Telegram
func (s *Service) SetWebhook(webhookURL string) error {
	webhook, err := tgbotapi.NewWebhook(webhookURL)
	if err != nil {
		return fmt.Errorf("%w: failed to create webhook config: %v", ErrSetWebhook, err)
	}

	_, err = s.bot.Request(webhook)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrSetWebhook, err)
	}

	return nil
}

// DeleteWebhook удаляет webhook (переключает на long polling)
func (s *Service) DeleteWebhook() error {
	deleteWebhook := tgbotapi.DeleteWebhookConfig{
		DropPendingUpdates: false, // Сохраняем необработанные сообщения
	}

	_, err := s.bot.Request(deleteWebhook)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrDeleteWebhook, err)
	}

	return nil
}

// GetUpdatesChan возвращает канал для получения обновлений в режиме long polling
func (s *Service) GetUpdatesChan(offset int) tgbotapi.UpdatesChannel {
	updateConfig := tgbotapi.NewUpdate(offset)
	updateConfig.Timeout = 60 // Long polling timeout

	return s.bot.GetUpdatesChan(updateConfig)
}
