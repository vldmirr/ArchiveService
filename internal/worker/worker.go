package worker

import (
    "archive-service/models"
    "archive-service/repository"
    "archive/zip"
    "io"
    "net/http"
    "os"
    "path/filepath"
    "sync"
    "time"
    "fmt"
)

type Worker struct {
    storage    *repository.Storage
    taskWaiter *TaskWaiter
}

type TaskWaiter struct {
    waitGroups map[string]*sync.WaitGroup
    taskStates map[string]bool // true - processing allowed, false - waiting
    mutex      sync.RWMutex
}

func NewTaskWaiter() *TaskWaiter {
    return &TaskWaiter{
        waitGroups: make(map[string]*sync.WaitGroup),
        taskStates: make(map[string]bool),
    }
}

func (tw *TaskWaiter) Add(taskID string) {
    tw.mutex.Lock()
    defer tw.mutex.Unlock()
    
    if _, exists := tw.waitGroups[taskID]; !exists {
        tw.waitGroups[taskID] = &sync.WaitGroup{}
        tw.waitGroups[taskID].Add(1)
        tw.taskStates[taskID] = false // initially waiting
    }
}

func (tw *TaskWaiter) AllowProcessing(taskID string) {
    tw.mutex.Lock()
    defer tw.mutex.Unlock()
    
    if wg, exists := tw.waitGroups[taskID]; exists {
        if !tw.taskStates[taskID] {
            tw.taskStates[taskID] = true
            wg.Done()
        }
    }
}

//метод преднамеренной задежки
// func (tw *TaskWaiter) Wait(taskID string, timeout time.Duration) bool {
//     tw.mutex.RLock()
//     wg, exists := tw.waitGroups[taskID]
//     state := tw.taskStates[taskID]
//     tw.mutex.RUnlock()
    
//     if !exists||state  {
//         return true // no waiting needed
//     }
    
//     // Создаем канал для ожидания с таймаутом
//     done := make(chan struct{})
//     go func() {
//         wg.Wait()
//         close(done)
//     }()
    
//     // Ждем либо завершения, либо таймаута
//     select {
//     case <-done:
//         return true
//     case <-time.After(timeout):
//         // При таймауте принудительно разрешаем обработку
//         tw.mutex.Lock()
//         if wg, exists := tw.waitGroups[taskID]; exists && !tw.taskStates[taskID] {
//             tw.taskStates[taskID] = true
//             wg.Done()
//         }
//         tw.mutex.Unlock()
//         return true
//     }
// }

func (tw *TaskWaiter) Remove(taskID string) {
    tw.mutex.Lock()
    defer tw.mutex.Unlock()
    delete(tw.waitGroups, taskID)
    delete(tw.taskStates, taskID)
}

func (tw *TaskWaiter) IsWaiting(taskID string) bool {
    tw.mutex.RLock()
    defer tw.mutex.RUnlock()
    
    state, exists := tw.taskStates[taskID]
    return exists && !state
}

func NewWorker(storage *repository.Storage) *Worker {
    return &Worker{
        storage:    storage,
        taskWaiter: NewTaskWaiter(),
    }
}

func (w *Worker) ProcessTask(taskID string) {
    // Ждем разрешения на обработку (60 секунд таймаут)
    //w.taskWaiter.Wait(taskID, 60*time.Second)

    task, exists := w.storage.GetTask(taskID)
    if !exists {
        w.taskWaiter.Remove(taskID)
        return
    }

    // Обновляем статус задачи
    task.Status = models.StatusInProcess
    w.storage.UpdateTask(task)

    // Создаем временный архив
    archivePath := filepath.Join(os.TempDir(), taskID+".zip")
    archiveFile, err := os.Create(archivePath)
    if err != nil {
        w.handleError(task, "", "Failed to create archive: "+err.Error())
        w.taskWaiter.Remove(taskID)
        return
    }
    defer archiveFile.Close()

    zipWriter := zip.NewWriter(archiveFile)
    defer zipWriter.Close()

    var errors []models.FileError

    // Скачиваем и добавляем файлы в архив
    for i, url := range task.FileURLs {
        err := w.downloadAndAddToZip(zipWriter, url, i)
        if err != nil {
            errors = append(errors, models.FileError{
                URL:   url,
                Error: err.Error(),
            })
        }
        time.Sleep(100 * time.Millisecond) // небольшая задержка между файлами
    }

    task.Errors = errors

    if len(errors) > 0 {
        task.Status = models.StatusFailed
    } else {
        task.Status = models.StatusCompleted
        task.ArchiveURL = "/download/" + taskID
    }

    w.storage.UpdateTask(task)
    w.taskWaiter.Remove(taskID)
    
    fmt.Printf("Task %s processing completed with status: %s\n", taskID, task.Status)
}

func (w *Worker) downloadAndAddToZip(zipWriter *zip.Writer, url string, index int) error {
    // Скачиваем файл
    client := &http.Client{Timeout: 30 * time.Second}
    resp, err := client.Get(url)
    if err != nil {
        return fmt.Errorf("download failed: %v", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return fmt.Errorf("HTTP status %d", resp.StatusCode)
    }

    // Получаем имя файла из URL или используем индекс
    filename := filepath.Base(url)
    if filename == "." || filename == "/" {
        filename = fmt.Sprintf("file_%d", index)
    }

    // Создаем файл в архиве
    zipFile, err := zipWriter.Create(filename)
    if err != nil {
        return fmt.Errorf("create zip entry failed: %v", err)
    }

    // Копируем содержимое
    _, err = io.Copy(zipFile, resp.Body)
    if err != nil {
        return fmt.Errorf("copy to zip failed: %v", err)
    }
    
    return nil
}

func (w *Worker) handleError(task *models.Task, url, errorMsg string) {
    task.Status = models.StatusFailed
    if url != "" {
        task.Errors = append(task.Errors, models.FileError{
            URL:   url,
            Error: errorMsg,
        })
    }
    w.storage.UpdateTask(task)
    w.taskWaiter.Remove(task.ID)
}

//TODO: II.I
func (w *Worker) Start() {
    go func() {
        for {
            time.Sleep(3 * time.Second) // увеличиваем интервал проверки
            
            tasks := w.storage.GetAllTasks()
            for _, task := range tasks {
                if task.Status == models.StatusPending {
                    // Проверяем, не добавлена ли уже задача в ожидание
                    if !w.taskWaiter.IsWaiting(task.ID) {
                        w.taskWaiter.Add(task.ID)
                        go w.ProcessTask(task.ID)
                        fmt.Printf("Task %s added to processing queue\n", task.ID)
                    }
                }
            }
        }
    }()
}

// Получить информацию о задаче
func (w *Worker) GetTaskInfo(taskID string) (bool, bool) {
    w.taskWaiter.mutex.RLock()
    defer w.taskWaiter.mutex.RUnlock()
    
    _, exists := w.taskWaiter.waitGroups[taskID]
    state, stateExists := w.taskWaiter.taskStates[taskID]
    
    return exists, stateExists && state
}