package utils

import (
	"os"
	"time"

	"github.com/sirupsen/logrus"
)

// Logger — это глобальный логгер для всего приложения
var Logger *logrus.Logger

// InitLogger — функция инициализации логгера
func InitLogger(logFile string) {
	Logger = logrus.New() // Создаем новый логгер

	// Устанавливаем формат вывода (JSON или текст)
	Logger.SetFormatter(&logrus.JSONFormatter{
		TimestampFormat: time.RFC3339, // Формат времени
	})

	// Устанавливаем уровень логирования (Debug, Info, Warn, Error, Fatal, Panic)
	Logger.SetLevel(logrus.DebugLevel)

	// Настраиваем вывод логов
	// Если указан logFile, логи будут записываться в файл
	if logFile != "" {
		file, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			Logger.Fatalf("Не удалось открыть файл логов: %v", err)
		}
		Logger.SetOutput(file)
		return
	}

	// Если файл не указан, выводим логи в стандартный вывод (консоль)
	Logger.SetOutput(os.Stdout)
}
