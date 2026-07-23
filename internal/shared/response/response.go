package response

type WebResponse[T any] struct {
	Data    T             `json:"data"`
	Message string        `json:"message,omitempty"`
	Success bool          `json:"success,omitempty"`
	Paging  *PageMetadata `json:"paging,omitempty"`
}

type ApiErrorResponse struct {
	Message    string              `json:"message"`
	StatusCode int                 `json:"statusCode"`
	Errors     map[string][]string `json:"errors,omitempty"`
}

type PageResponse[T any] struct {
	Data         []T          `json:"data,omitempty"`
	PageMetadata PageMetadata `json:"paging,omitempty"`
}

type PageMetadata struct {
	Page      int   `json:"page"`
	Size      int   `json:"size"`
	TotalItem int64 `json:"total_item"`
	TotalPage int64 `json:"total_page"`
}