package domain

type PageResponse struct {
	PageStart int `json:"pageStart"`
	PageEnd   int `json:"pageEnd"`
	Limit     int `json:"limit"`
	Total     int `json:"total"`
}

type PaginatedResponse[T any] struct {
	Data         []T          `json:"data"`
	PageResponse PageResponse `json:"pageResponse"`
}
