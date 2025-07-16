package main

import (
	"context"
	"github.com/b00tkitism/wsc/proxy"
	"log/slog"
	"net/http"
	"os"
	"time"
)

var _ proxy.Authenticator = &CustomAuth{}

type CustomAuth struct {
}

func (c *CustomAuth) Authenticate(ctx context.Context, auth string) (int64, error) {
	slog.Debug("authenticating : ", slog.String("auth", auth))
	return 1, nil
}

var usage int64 = 0

func (c *CustomAuth) ReportUsage(ctx context.Context, id int64, usedTraffic int64) error {
	usage += usedTraffic
	slog.Debug("updating user", slog.Int64("id", id), slog.Int64("used-traffic", usedTraffic), slog.Int64("usage", usage), slog.Float64("usage-MB", float64(usage)/1e6))
	return nil
}

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug})))
	pro := proxy.NewProxy(&CustomAuth{}, 30, time.Second*10, 1*1e6*1e4)
	slog.Info("running...")
	err := http.ListenAndServe(":4040", pro)
	if err != nil {
		panic(err)
	}
	slog.Info("run!")
}
