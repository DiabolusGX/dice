// Copyright (c) 2022-present, DiceDB contributors
// All rights reserved. Licensed under the BSD 3-Clause License. See LICENSE file in the project root for full license information.

package wal

import (
	"fmt"
	"log/slog"
	sync "sync"
	"time"

	"github.com/dicedb/dice/internal/cmd"
)

type AbstractWAL interface {
	LogCommand([]byte) error
	Close() error
	Init(t time.Time) error
	ForEachCommand(f func(c cmd.DiceDBCmd) error) error
}

var (
	ticker *time.Ticker
	stopCh chan struct{}
	mu     sync.Mutex
)

func init() {
	ticker = time.NewTicker(1 * time.Minute)
	stopCh = make(chan struct{})
}

func rotateWAL(wl AbstractWAL) {
	mu.Lock()
	defer mu.Unlock()

	if err := wl.Close(); err != nil {
		slog.Warn("error closing the WAL", slog.Any("error", err))
	}

	if err := wl.Init(time.Now()); err != nil {
		slog.Warn("error creating a new WAL", slog.Any("error", err))
	}
}

func periodicRotate(wl AbstractWAL) {
	for {
		select {
		case <-ticker.C:
			rotateWAL(wl)
		case <-stopCh:
			return
		}
	}
}

func InitBG(wl AbstractWAL) {
	go periodicRotate(wl)
}

func ShutdownBG() {
	close(stopCh)
	ticker.Stop()
}

func ReplayWAL(wl AbstractWAL) {
	err := wl.ForEachCommand(func(c cmd.DiceDBCmd) error {
		fmt.Println("replaying", c.Cmd, c.Args)
		return nil
	})

	if err != nil {
		slog.Warn("error replaying WAL", slog.Any("error", err))
	}
}
