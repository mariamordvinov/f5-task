package logger

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
)

var logWriter *bufio.Writer

type accessLogLine struct {
	Request requestLog  `json:"req"`
	Reponse responseLog `json:"rsp"`
}

type requestLog struct {
	Url          string `json:"url"`
	QueryPararms string `json:"qs_params"`
	Headers      string `json:"headers"`
	ReqBodyLen   int    `json:"req_body_len"`
}

type responseLog struct {
	Status      string `json:"status_class"`
	RespBodyLen int    `json:"rsp_body_len"`
}

// ResponseWriter wrapper that logs responses
type ResponseWriterWithLogs struct {
	responseWriter http.ResponseWriter
	logLine        *accessLogLine
}

func (w ResponseWriterWithLogs) Header() http.Header {
	return w.responseWriter.Header()
}

func (w ResponseWriterWithLogs) WriteHeader(statusCode int) {
	w.logLine.Reponse.Status = statusToStatusClass(statusCode)
	w.responseWriter.WriteHeader(statusCode)
}

func (w ResponseWriterWithLogs) Write(b []byte) (int, error) {
	w.logLine.Reponse.RespBodyLen = len(b)
	return w.responseWriter.Write(b)
}

// Middleware to log each request response
func LogHandler(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		//creating my own writer to be able to log responses
		writer := ResponseWriterWithLogs{
			responseWriter: w,
		}

		//creating the data to log request
		logLine := accessLogLine{}
		logLine.Request = createLogRequest(r)
		//by default responseWriter return status code 200 - so implemented the same logic
		logLine.Reponse.Status = statusToStatusClass(200)
		writer.logLine = &logLine

		next(writer, r)

		jsonData, err := json.Marshal(writer.logLine)
		if err == nil {
			writeSingleLineToLogFile(string(jsonData))
		}
	}
}

// create the log file
func InitLogger() error {
	file, err := os.Create("access_log.log")

	if err != nil {
		return err
	}

	logWriter = bufio.NewWriter(file)

	return nil
}

// gets a map of headers and returns them as string
func getHeadersAsString(headers http.Header) (string, error) {
	reqHeadersBytes, err := json.Marshal(headers)
	return string(reqHeadersBytes), err
}

// writes a line to the log file
func writeSingleLineToLogFile(line string) error {
	_, err := logWriter.WriteString(line + "\n")
	logWriter.Flush()
	return err
}

// Creates a requestLog struct from a http.request
func createLogRequest(r *http.Request) requestLog {
	headers, err := getHeadersAsString(r.Header)
	if err != nil {
		return requestLog{}
	}

	url := fmt.Sprintf("http://%s%s", r.Host, r.URL.Path)
	reqLog := requestLog{
		Url:          url,
		QueryPararms: r.URL.RawQuery,
		Headers:      headers,
		ReqBodyLen:   int(r.ContentLength),
	}
	return reqLog
}

func statusToStatusClass(status int) string {
	if status < 200 {
		return "1xx"
	}
	if status < 300 {
		return "2xx"
	}
	if status < 400 {
		return "3xx"
	}
	if status < 500 {
		return "4xx"
	}
	if status < 600 {
		return "5xx"
	}
	return ""
}
