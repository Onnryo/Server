package errors

type Error struct {
	Message string `json:"error"`
	HttpStatus int
}
