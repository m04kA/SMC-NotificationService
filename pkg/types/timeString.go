package types

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

// TimeString кастомный тип для TIME полей PostgreSQL
// Формат: "HH:MM" (например: "09:30", "14:00", "23:59")
// Реализует интерфейсы:
// - sql.Scanner (для чтения из БД)
// - driver.Valuer (для записи в БД)
// - json.Marshaler (для JSON сериализации)
// - json.Unmarshaler (для JSON десериализации)
type TimeString string

const (
	// TimeFormat формат времени для TimeString
	TimeFormat = "15:04"
)

var (
	// ErrInvalidTimeFormat возвращается при некорректном формате времени
	ErrInvalidTimeFormat = errors.New("invalid time format, expected HH:MM")

	// ErrInvalidTimeValue возвращается при некорректном значении времени
	ErrInvalidTimeValue = errors.New("invalid time value")
)

// Scan implements sql.Scanner interface
// Поддерживает чтение из PostgreSQL TIME полей
func (t *TimeString) Scan(value interface{}) error {
	if value == nil {
		*t = ""
		return nil
	}

	switch v := value.(type) {
	case time.Time:
		*t = TimeString(v.Format(TimeFormat))
		return nil
	case []byte:
		*t = TimeString(v)
		return nil
	case string:
		*t = TimeString(v)
		return nil
	default:
		return fmt.Errorf("cannot scan type %T into TimeString", value)
	}
}

// Value implements driver.Valuer interface
// Поддерживает запись в PostgreSQL TIME поля
func (t TimeString) Value() (driver.Value, error) {
	if t == "" {
		return nil, nil
	}
	return string(t), nil
}

// MarshalJSON implements json.Marshaler interface
func (t TimeString) MarshalJSON() ([]byte, error) {
	if t == "" {
		return json.Marshal(nil)
	}
	return json.Marshal(string(t))
}

// UnmarshalJSON implements json.Unmarshaler interface
func (t *TimeString) UnmarshalJSON(data []byte) error {
	var s *string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}

	if s == nil {
		*t = ""
		return nil
	}

	*t = TimeString(*s)
	return nil
}

// String возвращает строковое представление времени
func (t TimeString) String() string {
	return string(t)
}

// IsZero возвращает true, если время не установлено
func (t TimeString) IsZero() bool {
	return t == ""
}

// Parse парсит TimeString в time.Time для указанной даты
// Возвращает полный time.Time объект с установленной датой и временем
func (t TimeString) Parse(date time.Time) (time.Time, error) {
	if t.IsZero() {
		return time.Time{}, ErrInvalidTimeValue
	}

	// Парсим время в контексте указанной даты
	layout := "2006-01-02 15:04"
	dateStr := date.Format("2006-01-02")
	fullDateTimeStr := fmt.Sprintf("%s %s", dateStr, string(t))

	parsed, err := time.Parse(layout, fullDateTimeStr)
	if err != nil {
		return time.Time{}, fmt.Errorf("%w: %v", ErrInvalidTimeFormat, err)
	}

	return parsed, nil
}

// ToTime парсит TimeString в time.Time (используя текущую дату)
func (t TimeString) ToTime() (time.Time, error) {
	return t.Parse(time.Now())
}

// Validate проверяет корректность формата времени
func (t TimeString) Validate() error {
	if t.IsZero() {
		return nil
	}

	_, err := time.Parse(TimeFormat, string(t))
	if err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidTimeFormat, err)
	}

	return nil
}

// IsBefore проверяет, что время меньше другого времени
func (t TimeString) IsBefore(other TimeString) bool {
	if t.IsZero() || other.IsZero() {
		return false
	}

	t1, err1 := time.Parse(TimeFormat, string(t))
	t2, err2 := time.Parse(TimeFormat, string(other))

	if err1 != nil || err2 != nil {
		return false
	}

	return t1.Before(t2)
}

// IsAfter проверяет, что время больше другого времени
func (t TimeString) IsAfter(other TimeString) bool {
	if t.IsZero() || other.IsZero() {
		return false
	}

	t1, err1 := time.Parse(TimeFormat, string(t))
	t2, err2 := time.Parse(TimeFormat, string(other))

	if err1 != nil || err2 != nil {
		return false
	}

	return t1.After(t2)
}

// Equal проверяет равенство времён
func (t TimeString) Equal(other TimeString) bool {
	return string(t) == string(other)
}

// AddMinutes добавляет минуты к времени
// Возвращает новый TimeString
func (t TimeString) AddMinutes(minutes int) (TimeString, error) {
	if t.IsZero() {
		return "", ErrInvalidTimeValue
	}

	parsed, err := time.Parse(TimeFormat, string(t))
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrInvalidTimeFormat, err)
	}

	newTime := parsed.Add(time.Duration(minutes) * time.Minute)
	return TimeString(newTime.Format(TimeFormat)), nil
}

// MinutesBetween вычисляет разницу в минутах между двумя временами
// Возвращает положительное число, если other позже текущего времени
func (t TimeString) MinutesBetween(other TimeString) (int, error) {
	if t.IsZero() || other.IsZero() {
		return 0, ErrInvalidTimeValue
	}

	t1, err1 := time.Parse(TimeFormat, string(t))
	t2, err2 := time.Parse(TimeFormat, string(other))

	if err1 != nil {
		return 0, fmt.Errorf("%w: %v", ErrInvalidTimeFormat, err1)
	}
	if err2 != nil {
		return 0, fmt.Errorf("%w: %v", ErrInvalidTimeFormat, err2)
	}

	duration := t2.Sub(t1)
	return int(duration.Minutes()), nil
}

// NewTimeString создает новый TimeString из time.Time
func NewTimeString(t time.Time) TimeString {
	return TimeString(t.Format(TimeFormat))
}

// NewTimeStringFromString создает TimeString из строки с валидацией
func NewTimeStringFromString(s string) (TimeString, error) {
	t := TimeString(s)
	if err := t.Validate(); err != nil {
		return "", err
	}
	return t, nil
}

// MustNewTimeString создает TimeString из строки, паникует при ошибке
// Использовать только для константных значений
func MustNewTimeString(s string) TimeString {
	t, err := NewTimeStringFromString(s)
	if err != nil {
		panic(fmt.Sprintf("invalid time string: %s", s))
	}
	return t
}
