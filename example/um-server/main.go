package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/b00tkitism/wsc/proxy"
	"github.com/itsabgr/ge"
)

var dbFilePath = flag.String("db", "./database.db", "sqlite database")

func main() {
	flag.Parse()

	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug})))

	ctx, cancel := signal.NotifyContext(context.Background(), os.Kill, os.Interrupt)
	defer cancel()
	_ = ctx

	db := &Database{File: *dbFilePath}
	if err := db.Start(ctx); err != nil {
		ge.Throw(err)
	}
	defer db.Stop()

	// if err := db.CreateUser(ctx, "mobinyentoken", 3*1024*1024, 1*1024*1024*1024, time.Minute*20); err != nil {
	// 	fmt.Println("err in adding mobin user : ", err)
	// }

	if err := db.ChargeUserService(ctx, "mobinyentoken", 10*1024*1024, 150*1024*1024*1024, time.Hour*24*10); err != nil {
		fmt.Println("failed to charge service : ", err)
	} else {
		fmt.Println("service charged!")
	}

	// if id, rate, err := (&CustomAuth{db}).Authenticate(ctx, "mobinyentoken"); err != nil {
	// 	fmt.Println("failed to authorize the token : ", err)
	// } else {
	// 	fmt.Println("user id is : ", id, rate)
	// 	if err := (&CustomAuth{db}).ReportUsage(ctx, id, 1024*3); err != nil {
	// 		fmt.Println("failed to update traffic : ", err)
	// 	} else {
	// 		fmt.Println("updated users traffic!")
	// 	}
	// }

	pro := proxy.NewProxy(&CustomAuth{DB: db}, 60, time.Second*10, 1*1e6*1e4)

	server := &http.Server{
		Addr:    ":4040",
		Handler: pro,
	}

	go func() {
		slog.Info("running server on : ", slog.String("addr", server.Addr))
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("server error", "err", err)
			cancel()
		}
	}()

	<-ctx.Done()
	slog.Info("shutdown signal received")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), time.Second*5)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		slog.Error("shutdown failed", "err", err)
	} else {
		slog.Info("server gracefully stopped")
	}
}
