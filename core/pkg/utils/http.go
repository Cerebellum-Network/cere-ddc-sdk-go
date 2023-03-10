package utils

func IsSuccessHttpStatus(code int) bool {
	return code >= 200 && code < 300
}
