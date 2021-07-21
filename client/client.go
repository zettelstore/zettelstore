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
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"zettelstore.de/z/api"
	"zettelstore.de/z/domain/id"
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
func (c *Client) newRequest(ctx context.Context, method string, ub *api.URLBuilder, body io.Reader) (*http.Request, error) {
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

func (c *Client) buildAndExecuteRequest(
	ctx context.Context, method string, ub *api.URLBuilder, body io.Reader, h http.Header) (*http.Response, error) {
	req, err := c.newRequest(ctx, method, ub, body)
	if err != nil {
		return nil, err
	}
	err = c.updateToken(ctx)
	if err != nil {
		return nil, err
	}
	for key, val := range h {
		req.Header[key] = append(req.Header[key], val...)
	}
	return c.executeRequest(req)
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
	req, err := c.newRequest(ctx, http.MethodPost, c.newURLBuilder('v'), strings.NewReader(authData.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return c.executeAuthRequest(req)
}

// RefreshToken updates the access token
func (c *Client) RefreshToken(ctx context.Context) error {
	req, err := c.newRequest(ctx, http.MethodPut, c.newURLBuilder('v'), nil)
	if err != nil {
		return err
	}
	return c.executeAuthRequest(req)
}

// CreateZettel creates a new zettel and returns its URL.
func (c *Client) CreateZettel(ctx context.Context, data *api.ZettelDataJSON) (id.Zid, error) {
	var buf bytes.Buffer
	if err := encodeZettelData(&buf, data); err != nil {
		return id.Invalid, err
	}
	ub := c.jsonZettelURLBuilder(nil)
	resp, err := c.buildAndExecuteRequest(ctx, http.MethodPost, ub, &buf, nil)
	if err != nil {
		return id.Invalid, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		return id.Invalid, errors.New(resp.Status)
	}
	dec := json.NewDecoder(resp.Body)
	var newZid api.ZidJSON
	err = dec.Decode(&newZid)
	if err != nil {
		return id.Invalid, err
	}
	zid, err := id.Parse(newZid.ID)
	if err != nil {
		return id.Invalid, err
	}
	return zid, nil
}

func encodeZettelData(buf *bytes.Buffer, data *api.ZettelDataJSON) error {
	enc := json.NewEncoder(buf)
	enc.SetEscapeHTML(false)
	return enc.Encode(&data)
}

// ListZettel returns a list of all Zettel.
func (c *Client) ListZettel(ctx context.Context, query url.Values) ([]api.ZettelJSON, error) {
	ub := c.jsonZettelURLBuilder(query)
	resp, err := c.buildAndExecuteRequest(ctx, http.MethodGet, ub, nil, nil)
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
	return zl.List, nil
}

// GetZettelJSON returns a zettel as a JSON struct.
func (c *Client) GetZettelJSON(ctx context.Context, zid id.Zid, query url.Values) (*api.ZettelDataJSON, error) {
	ub := c.jsonZettelURLBuilder(query).SetZid(zid)
	resp, err := c.buildAndExecuteRequest(ctx, http.MethodGet, ub, nil, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, errors.New(resp.Status)
	}
	dec := json.NewDecoder(resp.Body)
	var out api.ZettelDataJSON
	err = dec.Decode(&out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// GetEvaluatedZettel return a zettel in a defined encoding.
func (c *Client) GetEvaluatedZettel(ctx context.Context, zid id.Zid, enc api.EncodingEnum) (string, error) {
	ub := c.jsonZettelURLBuilder(nil).SetZid(zid)
	ub.AppendQuery(api.QueryKeyFormat, enc.String())
	ub.AppendQuery(api.QueryKeyPart, api.PartContent)
	resp, err := c.buildAndExecuteRequest(ctx, http.MethodGet, ub, nil, nil)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", errors.New(resp.Status)
	}
	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(content), nil
}

// UpdateZettel updates an existing zettel.
func (c *Client) UpdateZettel(ctx context.Context, zid id.Zid, data *api.ZettelDataJSON) error {
	var buf bytes.Buffer
	if err := encodeZettelData(&buf, data); err != nil {
		return err
	}
	ub := c.jsonZettelURLBuilder(nil).SetZid(zid)
	resp, err := c.buildAndExecuteRequest(ctx, http.MethodPut, ub, &buf, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		return errors.New(resp.Status)
	}
	return nil
}

// RenameZettel renames a zettel.
func (c *Client) RenameZettel(ctx context.Context, oldZid, newZid id.Zid) error {
	ub := c.jsonZettelURLBuilder(nil).SetZid(oldZid)
	h := http.Header{
		api.HeaderDestination: {c.jsonZettelURLBuilder(nil).SetZid(newZid).String()},
	}
	resp, err := c.buildAndExecuteRequest(ctx, api.MethodMove, ub, nil, h)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		return errors.New(resp.Status)
	}
	return nil
}

// DeleteZettel deletes a zettel with the given identifier.
func (c *Client) DeleteZettel(ctx context.Context, zid id.Zid) error {
	ub := c.jsonZettelURLBuilder(nil).SetZid(zid)
	resp, err := c.buildAndExecuteRequest(ctx, http.MethodDelete, ub, nil, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		return errors.New(resp.Status)
	}
	return nil
}

func (c *Client) jsonZettelURLBuilder(query url.Values) *api.URLBuilder {
	ub := c.newURLBuilder('z')
	for key, values := range query {
		if key == api.QueryKeyFormat {
			continue
		}
		for _, val := range values {
			ub.AppendQuery(key, val)
		}
	}
	return ub
}

// ListTags returns a map of all tags, together with the associated zettel containing this tag.
func (c *Client) ListTags(ctx context.Context) (map[string][]string, error) {
	err := c.updateToken(ctx)
	if err != nil {
		return nil, err
	}
	req, err := c.newRequest(ctx, http.MethodGet, c.newURLBuilder('t'), nil)
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
	return tl.Tags, nil
}

// ListRoles returns a list of all roles.
func (c *Client) ListRoles(ctx context.Context) ([]string, error) {
	err := c.updateToken(ctx)
	if err != nil {
		return nil, err
	}
	req, err := c.newRequest(ctx, http.MethodGet, c.newURLBuilder('r'), nil)
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
	var rl api.RoleListJSON
	err = dec.Decode(&rl)
	if err != nil {
		return nil, err
	}
	return rl.Roles, nil
}
