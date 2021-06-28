//-----------------------------------------------------------------------------
// Copyright (c) 2021 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package client provides a client for accessing the Zettelstore via its API.
package client

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"net/url"
	"time"

	"zettelstore.de/z/api"
)

// Client contains all data to execute requests.
type Client struct {
	baseURL   string
	username  string
	password  string
	token     string
	tokenType string
	expires   time.Time
}

// NewClient create a new client.
func NewClient(baseURL string) *Client {
	c := Client{baseURL: baseURL}
	return &c
}

func (c *Client) newRequest(ctx context.Context, method, path string, body io.Reader) (*http.Request, error) {
	return http.NewRequestWithContext(ctx, method, c.baseURL+path, body)
}

// SetAuth sets authentication data.
func (c *Client) SetAuth(username, password string) {
	c.username = username
	c.password = password
}

func (c *Client) handleAuthResponse(resp *http.Response, err error) error {
	if err != nil {
		if resp != nil {
			resp.Body.Close()
		}
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return errors.New(resp.Status)
	}
	dec := json.NewDecoder(resp.Body)
	var tinfo api.AuthJSON
	err = dec.Decode(&tinfo)
	if err != nil {
		return err
	}
	c.token = tinfo.Token
	c.tokenType = tinfo.Type
	c.expires = time.Now().Add(time.Duration(tinfo.Expires*10/9) * time.Second)
	return nil
}

func (c *Client) updateToken(ctx context.Context) error {
	if c.username == "" {
		return nil
	}
	if time.Now().After(c.expires) {
		// New token
		resp, err := http.PostForm(
			c.baseURL+"/a", url.Values{"username": {c.username}, "password": {c.password}})
		return c.handleAuthResponse(resp, err)
	}
	// Refresh token
	req, err := c.newRequest(ctx, http.MethodPut, "/a", nil)
	if err != nil {
		return err
	}
	req.Header.Add("Authorization", c.tokenType+" "+c.token)
	client := http.Client{}
	resp, err := client.Do(req)
	return c.handleAuthResponse(resp, err)
}

// ListZettel returns a list of all Zettel.
func (c *Client) ListZettel(ctx context.Context) string {
	err := c.updateToken(ctx)
	if err != nil {
		log.Println("ERR", err)
	}
	return ""
}
