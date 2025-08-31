package main

import (
	"context"
	"errors"
	"math"
	"strconv"
	"time"

	"github.com/b00tkitism/wsc/proxy"
)

var _ proxy.Authenticator = &CustomAuth{}

type CustomAuth struct {
	DB *Database
}

func (cauth *CustomAuth) Authenticate(ctx context.Context, auth string) (int64, int64, error) {
	id, rate, usedTraffic, totalTraffic, endTime, err := cauth.DB.FindUser(ctx, auth)
	if err != nil {
		return id, 0, errors.New("failed to find user '" + auth + "'. (Error: " + err.Error() + ")")
	}
	if time.Now().UnixNano() >= endTime {
		return id, 0, errors.New("user service time exceeded '" + auth + "'(" + strconv.Itoa(int(id)) + ") at '" + time.Unix(0, endTime).String() + "'")
	}
	if usedTraffic >= totalTraffic {
		usedTrafficStr := strconv.FormatFloat(float64(usedTraffic)/1024/1024, 'g', -1, 64) + "MB"
		totalTrafficStr := strconv.FormatFloat(float64(totalTraffic)/1024/1024, 'g', -1, 64) + "MB"
		remainedTrafficStr := strconv.FormatFloat(math.Max(float64(totalTraffic-usedTraffic)/1024/1024, 0), 'g', -1, 64) + "MB"
		return 0, 0, errors.New("user service traffic exceeded '" + auth + "'(" + strconv.Itoa(int(id)) + "). [used_traffic = " + usedTrafficStr + ", total_traffic = " + totalTrafficStr + ", remained = " + remainedTrafficStr + "]")
	}
	return id, rate, nil
}

func (cauth *CustomAuth) ReportUsage(ctx context.Context, id int64, usedTraffic int64) error {
	return cauth.DB.UpdateUser(ctx, id, usedTraffic)
}
