package models

type TaskStatus string

const (
    StatusPending   TaskStatus = "pending"
    StatusInProcess TaskStatus = "in_process"
    StatusCompleted TaskStatus = "completed"
    StatusFailed    TaskStatus = "failed"
)

//TODO: I
type Task struct {
    ID         string      `json:"id"`
    Status     TaskStatus  `json:"status"`
    FileURLs   []string    `json:"file_urls"`
    Errors     []FileError `json:"errors,omitempty"`
    ArchiveURL string      `json:"archive_url,omitempty"`
}

type FileError struct {
    URL   string `json:"url"`
    Error string `json:"error"`
}

type CreateTaskResponse struct {
    TaskID string `json:"task_id"`
}

type AddFileRequest struct {
    TaskID string `json:"task_id"`
    URL    string `json:"url"`
}

type TaskRequest struct {
    FileURLs []string `json:"file_urls"`
}