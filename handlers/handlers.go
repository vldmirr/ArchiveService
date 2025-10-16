package handlers

import (
    "archive-service/models"
    "archive-service/repository"
    "archive-service/internal/worker"
    "net/http"
    "os"
    "path/filepath"

    "github.com/gin-gonic/gin"
    "github.com/google/uuid"
)

type Handler struct {
    storage *repository.Storage
    worker  *worker.Worker
}

func NewHandler(storage *repository.Storage, worker *worker.Worker) *Handler {
    return &Handler{storage: storage, worker: worker}
}

func (h *Handler) CreateTask(c *gin.Context) {
    var req models.TaskRequest
    if err := c.BindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
        return
    }

    taskID := uuid.New().String()
    task := &models.Task{
        ID:       taskID,
        Status:   models.StatusPending,
        FileURLs: req.FileURLs,
    }

    h.storage.CreateTask(task)

    c.JSON(http.StatusCreated, models.CreateTaskResponse{TaskID: taskID})
}

func (h *Handler) GetTaskStatus(c *gin.Context) {
    taskID := c.Param("id")

    task, exists := h.storage.GetTask(taskID)
    if !exists {
        c.JSON(http.StatusNotFound, gin.H{"error": "Task not found"})
        return
    }

    c.JSON(http.StatusOK, task)
}

func (h *Handler) DownloadArchive(c *gin.Context) {
    taskID := c.Param("id")

    task, exists := h.storage.GetTask(taskID)
    if !exists {
        c.JSON(http.StatusNotFound, gin.H{"error": "Task not found"})
        return
    }

    if task.Status != models.StatusCompleted {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Archive not ready"})
        return
    }

    archivePath := filepath.Join(os.TempDir(), taskID+".zip")

    // Проверяем существование файла
    if _, err := os.Stat(archivePath); os.IsNotExist(err) {
        c.JSON(http.StatusNotFound, gin.H{"error": "Archive file not found"})
        return
    }

    c.FileAttachment(archivePath, "archive_"+taskID+".zip")
}

func (h *Handler) ListTasks(c *gin.Context) {
    tasks := h.storage.GetAllTasks()
    c.JSON(http.StatusOK, tasks)
}

// Новый endpoint для разрешения обработки задачи
func (h *Handler) StartAllProcessing(c *gin.Context) {
    h.worker.Start()
    c.JSON(http.StatusOK, gin.H{"message": "All pending tasks started"})
}

//TODO: II.II
func (h *Handler) AddFileToTask(c *gin.Context) {
    var req models.AddFileRequest
    if err := c.BindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
        return
    }

    taskID := c.Param("id")

    task, exists := h.storage.GetTask(taskID)
    if !exists {
        c.JSON(http.StatusNotFound, gin.H{"error": "Task not found"})
        return
    }

    if task.Status != models.StatusPending {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot add files to task in progress"})
        return
    }

    task.FileURLs = append(task.FileURLs, req.URL)
    h.storage.UpdateTask(task)

    c.JSON(http.StatusOK, gin.H{"status": "File added successfully"})
}