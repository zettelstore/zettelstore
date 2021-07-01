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
	"net/http"
	"net/url"
	"strings"
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
	if !strings.HasSuffix(baseURL, "/") {
		baseURL += "/"
	}
	c := Client{baseURL: baseURL}
	return &c
}

func (c *Client) newURLBuilder(key byte) *api.URLBuilder {
	return api.NewURLBuilder(c.baseURL, key)
}
func (c *Client) newListRequest(ctx context.Context, method string, ub *api.URLBuilder, body io.Reader) (*http.Request, error) {
	return http.NewRequestWithContext(ctx, method, ub.String(), body)
}

func (c *Client) executeRequest(req *http.Request) (*http.Response, error) {
	if c.token != "" {
		req.Header.Add("Authorization", c.tokenType+" "+c.token)
	}
	client := http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		if resp != nil && resp.Body != nil {
			resp.Body.Close()
		}
		return nil, err
	}
	return resp, err
}

// SetAuth sets authentication data.
func (c *Client) SetAuth(username, password string) {
	c.username = username
	c.password = password
	c.token = ""
	c.tokenType = ""
	c.expires = time.Time{}
}

func (c *Client) executeAuthRequest(req *http.Request) error {
	resp, err := c.executeRequest(req)
	if err != nil {
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
		return c.Authenticate(ctx)
	}
	return c.RefreshToken(ctx)
}

// Authenticate sets a new token by sending user name and password.
func (c *Client) Authenticate(ctx context.Context) error {
	authData := url.Values{"username": {c.username}, "password": {c.password}}
	req, err := c.newListRequest(ctx, http.MethodPost, c.newURLBuilder('a'), strings.NewReader(authData.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return c.executeAuthRequest(req)
}

// RefreshToken updates the access token
func (c *Client) RefreshToken(ctx context.Context) error {
	req, err := c.newListRequest(ctx, http.MethodPut, c.newURLBuilder('a'), nil)
	if err != nil {
		return err
	}
	return c.executeAuthRequest(req)
}

// ListZettel returns a list of all Zettel.
func (c *Client) ListZettel(ctx context.Context) (*api.ZettelListJSON, error) {
	req, err := c.newListRequest(ctx, http.MethodGet, c.newURLBuilder('z'), nil)
	if err != nil {
		return nil, err
	}
	err = c.updateToken(ctx)
	if err != nil {
		return nil, err
	}
	resp, err := c.executeRequest(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, errors.New(resp.Status)
	}
	dec := json.NewDecoder(resp.Body)
	var zl api.ZettelListJSON
	err = dec.Decode(&zl)
	if err != nil {
		return nil, err
	}
	return &zl, nil
}

// ListTags returns a list of all tags, together with the associated zettel containing this tag.
func (c *Client) ListTags(ctx context.Context) (*api.TagListJSON, error) {
	err := c.updateToken(ctx)
	if err != nil {
		return nil, err
	}
	req, err := c.newListRequest(ctx, http.MethodGet, c.newURLBuilder('t'), nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.executeRequest(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, errors.New(resp.Status)
	}
	dec := json.NewDecoder(resp.Body)
	var tl api.TagListJSON
	err = dec.Decode(&tl)
	if err != nil {
		return nil, err
	}
	return &tl, nil
}
