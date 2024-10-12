package middleware

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/sing3demons/go-api/logger"
)

type HandlerResponse struct {
	ResultCode       string      `json:"resultCode"`
	ResultDesc       string      `json:"resultDesc"`
	DeveloperMessage string      `json:"developerMessage,omitempty"`
	Data             interface{} `json:"data,omitempty"`
}

type customResponseWriter struct {
	http.ResponseWriter
	statusCode int
	body       *bytes.Buffer
}

// newCustomResponseWriter initializes a new instance of CustomResponseWriter
func newCustomResponseWriter(w http.ResponseWriter) *customResponseWriter {
	return &customResponseWriter{
		ResponseWriter: w,
		statusCode:     http.StatusOK,
		body:           &bytes.Buffer{},
	}
}

// WriteHeader captures the status code
func (crw *customResponseWriter) WriteHeader(code int) {
	crw.statusCode = code
	crw.ResponseWriter.WriteHeader(code)
}

// Write captures the response body
func (crw *customResponseWriter) Write(data []byte) (int, error) {
	crw.body.Write(data) // Capture the response body
	return crw.ResponseWriter.Write(data)
}

func Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		invokeId := uuid.New().String()
		hostName, _ := os.Hostname()
		// Store request body
		bodyBytes, _ := io.ReadAll(r.Body)

		fmt.Println("+++> ", r.URL.Query())
		r.Body.Close() //  must close
		r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
		// ctx = context.WithValue(ctx, constant.BodyBytes, bodyBytes)
		startTime := time.Now()

		// concurrent_gauge
		// prometheus.ConcurrentGauge.Inc()
		// defer prometheus.ConcurrentGauge.Dec()

		resultReqBytes, _ := Minify(bodyBytes)

		summaryLog := logger.Summary{
			Hostname: hostName,
			Appname:  "go-api",
			Ssid:     r.RequestURI,
			Intime:   startTime.Format(time.RFC3339),
			Invoke:   invokeId,
			Input:    string(resultReqBytes),
		}

		ctx = logger.SetInvoke(ctx, invokeId)

		// Create a new request with the updated context
		// Call the next handler
		crw := newCustomResponseWriter(w)

		// Call the next handler in the chain
		next.ServeHTTP(crw, r.WithContext(ctx))

		var res HandlerResponse
		resBytes := crw.body.Bytes()
		json.Unmarshal(resBytes, &res)
		resultResBytes, _ := Minify(resBytes)
		summaryLog.Output = string(resultResBytes)
		endTime := time.Now()
		summaryLog.Outtime = endTime.Format(time.RFC3339)
		duration := endTime.Sub(startTime)
		summaryLog.DiffTime = duration.Milliseconds()
		summaryLog.Status = crw.statusCode

		go logger.ToSummaryLog(summaryLog)

		// Optionally, write the log summary in the response
		// w.Write([]byte("\nLog Summary:\n" + logSummary))
	})
}

func Minify(jsonB []byte) ([]byte, error) {
	var buff *bytes.Buffer = new(bytes.Buffer)
	errCompact := json.Compact(buff, jsonB)
	if errCompact != nil {
		newErr := fmt.Errorf("failure encountered compacting json := %v", errCompact)
		return []byte{}, newErr
	}

	b, err := io.ReadAll(buff)
	if err != nil {
		readErr := fmt.Errorf("read buffer error encountered := %v", err)
		return []byte{}, readErr
	}

	return b, nil
}
