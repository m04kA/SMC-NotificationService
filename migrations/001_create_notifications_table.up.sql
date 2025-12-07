-- Создание типов ENUM для уведомлений

-- Тип уведомления
CREATE TYPE notification_type AS ENUM (
    'welcome',              -- Приветственное сообщение при /start
    'booking_created',      -- Создание бронирования
    'booking_confirmed',    -- Подтверждение бронирования
    'booking_reminder',     -- Напоминание о бронировании
    'booking_cancelled',    -- Отмена бронирования
    'promo'                 -- Промо-сообщение от компании
);

-- Статус уведомления
CREATE TYPE notification_status AS ENUM (
    'pending',              -- Ожидает отправки (немедленное)
    'scheduled',            -- Запланировано на отправку (отложенное)
    'sent',                 -- Успешно отправлено
    'failed',               -- Ошибка при отправке
    'cancelled'             -- Отменено пользователем/системой
);

-- Создание таблицы уведомлений
CREATE TABLE IF NOT EXISTS notifications (
    id BIGSERIAL PRIMARY KEY,

    -- Получатель (одно из двух должно быть заполнено)
    telegram_user_id BIGINT,                    -- Telegram ID пользователя из UserService
    chat_id BIGINT,                             -- ID чата (для групп/каналов или личных сообщений)

    -- Группировка массовых рассылок
    span_id UUID,                               -- ID группы для массовых рассылок (NULL для одиночных)

    -- Содержимое сообщения
    message_text TEXT NOT NULL,                 -- Текст сообщения

    -- Медиа контент (массивы для поддержки множественных элементов)
    image_urls TEXT[],                          -- Массив URL изображений (до 10 для MediaGroup)

    -- Inline-кнопки (JSONB для гибкой структуры)
    -- Формат: [{"text": "Кнопка 1", "url": "https://..."}, {"text": "Кнопка 2", "url": "..."}]
    inline_buttons JSONB,

    -- Тип и статус
    notification_type notification_type NOT NULL,
    status notification_status NOT NULL DEFAULT 'pending',

    -- Планирование и отправка
    scheduled_for TIMESTAMP,                    -- Время отправки для отложенных уведомлений
    sent_at TIMESTAMP,                          -- Фактическое время отправки

    -- Дополнительные данные
    metadata JSONB DEFAULT '{}'::jsonb,         -- Дополнительные данные (booking_id, company_id и т.д.)
    error_message TEXT,                         -- Текст ошибки при failed статусе
    retry_count INT DEFAULT 0,                  -- Количество попыток отправки

    -- Аудит
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),

    -- Ограничения
    CONSTRAINT chk_recipient CHECK (
        telegram_user_id IS NOT NULL OR chat_id IS NOT NULL
    ),
    CONSTRAINT chk_scheduled CHECK (
        (status = 'scheduled' AND scheduled_for IS NOT NULL) OR
        (status != 'scheduled')
    ),
    CONSTRAINT chk_media_group_limit CHECK (
        image_urls IS NULL OR array_length(image_urls, 1) <= 10
    )
);

-- Индексы для оптимизации критичных запросов worker'а

-- Для processor: выборка pending уведомлений
CREATE INDEX idx_notifications_pending ON notifications(created_at)
WHERE status = 'pending';

-- Для scheduler: выборка scheduled уведомлений
CREATE INDEX idx_notifications_scheduled ON notifications(scheduled_for)
WHERE status = 'scheduled';

-- Для API: фильтрация списка по статусу и дате
CREATE INDEX idx_notifications_list ON notifications(status, created_at DESC);

-- Триггер для автоматического обновления updated_at
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_notifications_updated_at
    BEFORE UPDATE ON notifications
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Комментарии к таблице и столбцам
COMMENT ON TABLE notifications IS 'Таблица уведомлений для отправки через Telegram Bot API';
COMMENT ON COLUMN notifications.telegram_user_id IS 'Telegram ID пользователя из UserService (для личных сообщений)';
COMMENT ON COLUMN notifications.chat_id IS 'ID чата Telegram (для групп/каналов или явного указания личного чата)';
COMMENT ON COLUMN notifications.span_id IS 'UUID для группировки массовых рассылок (NULL для одиночных уведомлений)';
COMMENT ON COLUMN notifications.message_text IS 'Текст сообщения для отправки';
COMMENT ON COLUMN notifications.image_urls IS 'Массив URL изображений (до 10 штук для MediaGroup в Telegram)';
COMMENT ON COLUMN notifications.inline_buttons IS 'Массив inline-кнопок в формате JSON: [{"text": "Текст", "url": "https://..."}]';
COMMENT ON COLUMN notifications.notification_type IS 'Тип уведомления: welcome, booking_created, booking_confirmed, booking_reminder, booking_cancelled, promo';
COMMENT ON COLUMN notifications.status IS 'Статус: pending (ожидает), scheduled (запланировано), sent (отправлено), failed (ошибка), cancelled (отменено)';
COMMENT ON COLUMN notifications.scheduled_for IS 'Время отправки для отложенных уведомлений (required для status=scheduled)';
COMMENT ON COLUMN notifications.sent_at IS 'Фактическое время отправки уведомления';
COMMENT ON COLUMN notifications.metadata IS 'Дополнительные данные в формате JSON (booking_id, company_id, и т.д.)';
COMMENT ON COLUMN notifications.error_message IS 'Сообщение об ошибке при неудачной отправке';
COMMENT ON COLUMN notifications.retry_count IS 'Количество попыток повторной отправки';
