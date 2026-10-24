/*
 * Copyright (C) 2026  Henrique Almeida
 * This file is part of bibverse.
 *
 * bibverse is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * bibverse is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with bibverse.  If not, see <https://www.gnu.org/licenses/>.
 */

package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	// Roots for a host with no CA bundle, such as the scratch image, used only when the
	// system offers none.
	_ "golang.org/x/crypto/x509roots/fallback"

	"github.com/h3nc4/bibverse/internal/verse"
)

func main() {
	if err := run(); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "bibverse: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	client := &http.Client{Timeout: 15 * time.Second}
	text, err := verse.Fetch(ctx, client, verse.URL)
	if err != nil {
		return err
	}

	_, err = fmt.Println(text)
	return err
}
