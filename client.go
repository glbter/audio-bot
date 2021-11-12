package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type Client struct {
	url string
}

type Meme struct {
	Id   string `json:"id"`
	Name string `json:"name"`
}

type Memes []Meme

func (memes Memes) ToNames() []string {
	names := make([]string, 0, len(memes))
	for _, m := range memes {
		names = append(names, m.Name)
	}

	return names
}

func (memes Memes) ToIds() []string {
	names := make([]string, 0, len(memes))
	for _, m := range memes {
		names = append(names, m.Id)
	}

	return names
}

type MemeResponse struct {
	Memes  Memes `json:"memes"`
	Amount int   `json:"amount"`
}

type VoiceMeme struct {
	Id     string `json:"id"`
	Name   string `json:"name"`
	FileId string `json:"file_id"`
}

func NewClient(url string) *Client {
	return &Client{
		url: url,
	}
}

// find memes in backend by providing query
func (c *Client) FindMeme(params string) (MemeResponse, error) {
	resp, err := http.Get(c.url + "params=" + params) //TODO: specify url
	if err != nil {
		return MemeResponse{}, fmt.Errorf("find meme: get resource: %w", err)
	}

	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return MemeResponse{}, fmt.Errorf("find meme: read body: %w", err)
	}

	var memeResp MemeResponse
	if err := json.Unmarshal(body, &memeResp); err != nil {
		return MemeResponse{}, fmt.Errorf("find meme: unmarshal: %w", err)
	}

	return memeResp, nil
}

func (c *Client) GetMeme(id string) (VoiceMeme, error) {
	resp, err := http.Get(c.url + "/" + id) //TODO: specify url
	if err != nil {
		return VoiceMeme{}, fmt.Errorf("get meme: get resource: %w", err)
	}

	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return VoiceMeme{}, fmt.Errorf("get meme: read body: %w", err)
	}

	var memeResp VoiceMeme
	if err := json.Unmarshal(body, &memeResp); err != nil {
		return VoiceMeme{}, fmt.Errorf("get meme: unmarshal: %w", err)
	}

	return memeResp, nil
}
