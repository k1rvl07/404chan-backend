package upload

type ConfirmFilesRequest struct {
	FileIDs []string `json:"file_ids"`
}

type ConfirmFilesResponse struct {
	Files []UploadedFileResponse `json:"files"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}
