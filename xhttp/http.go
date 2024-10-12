package xhttp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/sing3demons/go-api/logger"
)

type ServiceConfig struct {
	Name       string
	Method     string
	Url        string
	System     string
	Timeout    int
	StatusCode string
}

type BasicAuth struct {
	Username string
	Password string
}

type Option struct {
	Body   *[]byte
	Query  map[string]string
	Param  map[string]string
	Header map[string]string
}

const (
	ContentType = "Content-Type"
	ContentJson = "application/json"
)

func New() *ServiceConfig {
	return &ServiceConfig{}
}

func (cfg *ServiceConfig) Call(ctx context.Context, option Option) ([]byte, error) {
	startTime := time.Now()
	invokeId := logger.GetInvoke(ctx)
	hostName, _ := os.Hostname()
	summaryLog := logger.Summary{
		Hostname: hostName,
		Appname:  cfg.Name,
		Ssid:     cfg.Url,
		Intime:   startTime.Format(time.RFC3339),
		Invoke:   invokeId,
	}
	// Store request body
	// bodyBytes, _ := io.ReadAll(r.Body)

	// ctx = context.WithValue(ctx, constant.BodyBytes, bodyBytes)

	// concurrent_gauge
	// prometheus.ConcurrentGauge.Inc()
	// defer prometheus.ConcurrentGauge.Dec()

	client := &http.Client{}
	// client.Timeout = time.Duration(cfg.Timeout) * time.Millisecond
	var input map[string]interface{}
	// Build Query Parameters
	// fmt.Println(option.Query)
	if len(option.Query) > 0 {
		query := url.Values{}
		for k, v := range option.Query {
			query.Add(k, v)
		}
		url := fmt.Sprintf("%s?%s", cfg.Url, query.Encode())
		cfg.Url = url
		// jsonString, _ := json.Marshal(query)
		input = map[string]interface{}{"query": query}
		summaryLog.Input = ParseString(query)
	}

	// Replace URL Parameters
	// fmt.Println(option.Param)
	if len(option.Param) > 0 {
		for k, v := range option.Param {
			url := strings.Replace(cfg.Url, k, v, 1)
			cfg.Url = url
			summaryLog.Input = ParseString(option.Param)
		}
	}

	// Build Request Body
	var bodyReader io.Reader
	if option.Body != nil {
		bodyReader = bytes.NewReader(*option.Body)
		summaryLog.Input = string(*option.Body)
	}

	// Create New HTTP Request
	req, err := http.NewRequestWithContext(ctx, cfg.Method, cfg.Url, bodyReader)
	if err != nil {
		return nil, err
	}

	// Add Headers
	if option.Header[ContentType] == "" {
		req.Header.Add(ContentType, ContentJson)
	}
	fmt.Println(option.Header)
	// for key, value := range option.Header {
	// 	req.Header.Add(key, value)
	// }

	// Channel for HTTP response and error
	ch := make(chan *http.Response, 1)
	serviceError := make(chan error, 1)

	// Perform HTTP call asynchronously
	go func() {
		res, err := client.Do(req)
		if err != nil {
			serviceError <- err
			return
		}
		ch <- res
	}()

	// Wait for result or timeout using context
	select {
	case result := <-ch:
		defer result.Body.Close()

		body, err := io.ReadAll(result.Body)
		if err != nil {
			return nil, err
		}
		cfg.StatusCode = ParseString(result.StatusCode)


		cleanedString := strings.ReplaceAll(string(body), "  ", " ")
		cleanedString = regexp.MustCompile(`([a-zA-Z])([A-Z])`).ReplaceAllString(cleanedString, "$1 $2")
		cleanedString = strings.ReplaceAll(cleanedString, "  ", " ")
		summaryLog.Output = cleanedString
		endTime := time.Now()
		summaryLog.Outtime = endTime.Format(time.RFC3339)
		duration := endTime.Sub(startTime)
		summaryLog.DiffTime = duration.Milliseconds()
		summaryLog.Status = result.StatusCode

		go logger.ToSummaryLog(summaryLog)
		return body, nil
	case err := <-serviceError:
		return nil, err
	case <-ctx.Done():
		return nil, fmt.Errorf("call %s:%s timeout %dms", cfg.System, cfg.Name, cfg.Timeout)
	}
}

func ParseString(data interface{}) string {
	b, _ := json.Marshal(data)

	return string(b)
}
