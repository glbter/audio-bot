package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
)

//import (
//	"bytes"
//	"encoding/json"
//	"fmt"
//	"io"
//	"net/http"
//	"strconv"
//	"sync"
//)
//
//var (
//	localStorage []VoiceMeme
//	localKeys map[string]string
//	mutx sync.Mutex
//	counter int
//)

//func init() {
//	localKeys = make(map[string]string)
//}

type Client struct {
	url string
}

type Meme struct {
	Id   string `json:"telegramFileId"`
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
	Memes  Memes
	Amount int
}

type VoiceMeme struct {
	Id     string `json:"id"`
	Name   string `json:"name"`
	FileId string `json:"telegramFileId"`
}

//public class AudioFileDto
//{
//	public Guid Id { get; set; }
//	public Guid TelegramFileId { get; set; }
//	public string Name { get; set; }
//	public DateTime CreatedOn { get; set; }
//}

// url + AudioFileController/
func NewClient(url string) *Client {
	return &Client{
		url: url + "api/AudioFile/",
	}
}

//public class AudioFileFilter
//{
//	public string Query { get; set; }
//	public int Skip { get; set; }
//	public int Take { get; set; }
//}

// find memes in backend by providing query
func (c *Client) FindMeme(params string, page string) (MemeResponse, error) {
	var memes Memes
	//for _, meme := range localStorage {
	//	memes = append(memes, Meme{Id: meme.Id, Name: meme.Name})
	//}

	//return MemeResponse{memes, len(memes)}, nil

	take := 10
	p, err := strconv.Atoi(page)
	if err != nil {
		return MemeResponse{}, fmt.Errorf("convert page to int: %w", err)
	}
	skip := (p - 1) * take

	url := fmt.Sprintf("%vGetByQuery?skip=%v&take=%v&query=%v", c.url, skip, take, params)
	resp, err := http.Get(url) //TODO: specify url
	if err != nil {
		return MemeResponse{}, fmt.Errorf("find meme: get resource: %w", err)
	}

	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return MemeResponse{}, fmt.Errorf("find meme: read body: %w", err)
	}

	if err := json.Unmarshal(body, &memes); err != nil {
		return MemeResponse{}, fmt.Errorf("find meme: unmarshal: %w", err)
	}

	res := MemeResponse{Memes: memes, Amount: len(memes)}
	return res, nil
}

func (c *Client) GetMeme(id string) (VoiceMeme, error) {
	//mutx.Lock()
	//defer mutx.Unlock()
	//n, err := strconv.Atoi(id)
	//if err != nil {
	//	return VoiceMeme{}, fmt.Errorf("ne ok: %w", err)
	//}
	//
	//for i, meme := range localStorage {
	//	if i == n {
	//		return meme, nil
	//	}
	//}
	//
	//return VoiceMeme{}, fmt.Errorf("ne ok: not found")
	resp, err := http.Get(c.url + "GetById?id=" + id) //TODO: specify url
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

func (c *Client) AddMeme(m Meme) error {
	//mutx.Lock()
	//defer mutx.Unlock()
	//id := strconv.Itoa(counter)
	//v := VoiceMeme{
	//	Id: id,
	//	FileId: m.Id,
	//	Name: m.Name,
	//}
	//localStorage = append(localStorage, v)
	//localKeys[id] = m.Name
	//counter += 1
	//
	//return nil
	body, err := json.Marshal(m)
	if err != nil {
		return fmt.Errorf("marshal body: %w", err)
	}

	resp, err := http.Post(c.url+"Create", "application/json", bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("post request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("create meme error")
	}

	return nil
}
