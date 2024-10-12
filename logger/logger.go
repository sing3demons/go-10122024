package logger

import (
	"context"
	"encoding/json"
	"os"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var Logger *zap.Logger = nil

type SummaryLogKey struct{}

func NewLogger() *zap.Logger {
	encoderConfig := map[string]string{
		"messageKey": "msg",
	}
	data, _ := json.Marshal(encoderConfig)
	var encCfg zapcore.EncoderConfig
	if err := json.Unmarshal(data, &encCfg); err != nil {
		return nil
	}

	// add the encoder config and rotator to create a new zap logger
	w := zapcore.AddSync(os.Stdout)
	core := zapcore.NewCore(
		zapcore.NewConsoleEncoder(encCfg),
		w,
		zap.InfoLevel)

	log := zap.New(core)
	Logger = log

	return log
}

type InvokeKey struct{}

func SetInvoke(ctx context.Context, invoke string) context.Context {
	return context.WithValue(ctx, InvokeKey{}, invoke)
}

func GetInvoke(ctx context.Context) string {
	invoke, _ := ctx.Value(InvokeKey{}).(string)

	return invoke
}

type Summary struct {
	LogTime    time.Time `json:"log_time"`
	Hostname   string    `json:"hostname"`
	Appname    string    `json:"appname"`
	Instance   string    `json:"instance"`
	LogName    string    `json:"log_name"`
	Intime     string    `json:"in_time"`
	Outtime    string    `json:"out_time"`
	DiffTime   int64     `json:"diff_time"`
	Ssid       string    `json:"ssid"`
	Invoke     string    `json:"invoke"`
	AuditLogId string    `json:"audit_log_id"`
	MobileNo   string    `json:"mobile_no"`
	Input      string    `json:"input"`
	Output     string    `json:"output"`
	Status     int       `json:"status"`
	ResultCode string    `json:"result_code"`
	Command    string    `json:"command"`
	MenuId     string    `json:"menu_id"`
	Channel    string    `json:"channel"`
}
type MessageLog struct {
	Topic   string `json:"topic"`
	Message string `json:"messages"`
}

func ToSummaryLog(newSummaryLog Summary) {
	startDate, _ := time.Parse(time.RFC3339, newSummaryLog.Intime)
	endTime := time.Now()
	newSummaryLog.LogName = "SUMMARY"
	newSummaryLog.LogTime = time.Now()
	newSummaryLog.Outtime = endTime.Format(time.RFC3339)
	newSummaryLog.DiffTime = endTime.Sub(startDate).Milliseconds()

	if len(newSummaryLog.Input) > 2000 {
		newSummaryLog.Input = newSummaryLog.Input[0:2000]
	}
	if len(newSummaryLog.Output) > 2000 {
		newSummaryLog.Output = newSummaryLog.Output[0:2000]
	}
	jsonBytes, _ := json.Marshal(newSummaryLog)

	// // Convert json to string:
	// jsonString := string(jsonBytes)

	// payloads := MessageLog{
	// 	Topic:   "ssb_summary",
	// 	Message: strings.ReplaceAll(jsonString, "|", ":"),
	// }

	// summaryBytes, _ := json.Marshal(payloads)

	summaryString := string(jsonBytes)

	Logger.Info(summaryString)
}
