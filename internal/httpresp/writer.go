package httpresp

import "net/http"

// ResponseWriter — обёртка над http.ResponseWriter, которая запоминает статус-код.
// Стандартный http.ResponseWriter не даёт прочитать статус после WriteHeader,
// а он нужен для логирования.
type ResponseWriter struct {
	http.ResponseWriter
	statusCode  int
	wroteHeader bool
}

func NewResponseWriter(w http.ResponseWriter) *ResponseWriter {
	return &ResponseWriter{
		ResponseWriter: w,
		statusCode:     http.StatusOK,
	}
}

// WriteHeader перехватывает статус-код и передаёт его дальше в оригинальный ResponseWriter.
func (rw *ResponseWriter) WriteHeader(statusCode int) {
	if rw.wroteHeader {
		return
	}

	rw.statusCode = statusCode
	rw.wroteHeader = true
	rw.ResponseWriter.WriteHeader(statusCode)
}

// Write оборачивает стандартный Write, записывая статус 200
// по умолчанию, если WriteHeader не вызывался.
func (rw *ResponseWriter) Write(b []byte) (int, error) {
	if !rw.wroteHeader {
		rw.WriteHeader(http.StatusOK)
	}

	return rw.ResponseWriter.Write(b)
}

// Unwrap нужен, чтобы http.ResponseController мог получить
// доступ к оригинальному ResponseWriter (важно для WebSockets, SSE и Flush).
func (rw *ResponseWriter) Unwrap() http.ResponseWriter {
	return rw.ResponseWriter
}

// GetStatusCode возвращает записанный статус-код.
func (rw *ResponseWriter) GetStatusCode() int {
	return rw.statusCode
}
