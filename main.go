package main

import (
	"archive-service/handlers"
	"archive-service/repository"
	"archive-service/pkg"

	"github.com/gin-gonic/gin"
)


func main() {
	// Инициализация компонентов
	storage := repository.NewStorage()
	worker := worker.NewWorker(storage)
	handler := handlers.NewHandler(storage, worker)

	// Запуск воркера
	//worker.Start()

	// Настройка роутера
	router := gin.Default()

	// Маршруты
	router.POST("/newtask", handler.CreateTask)
	router.GET("/tasks/:id", handler.GetTaskStatus)
	router.POST("/task/add-file/:id", handler.AddFileToTask)
	router.GET("/download/:id", handler.DownloadArchive)
	router.GET("/tasks", handler.ListTasks)
	//router.POST("/tasks/:id/start", handler.StartProcessing) // Единичная обработка
	router.POST("/tasks/start-all", handler.StartAllProcessing) // Запуск всех
	

	// Запуск сервера
	router.Run(":8080")
}