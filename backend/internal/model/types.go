package model

import "time"

type User struct {
	ID        string
	BaseURL   string
	APIKey    string
	CreatedAt time.Time
}

type Story struct {
	ID             string    `json:"id"`
	Slug           string    `json:"slug"`
	Title          string    `json:"title"`
	Description    string    `json:"description"`
	CoverURL       string    `json:"coverUrl"`
	SystemPrompt   string    `json:"systemPrompt"`
	Scenario       string    `json:"scenario,omitempty"`
	StylePrompt    string    `json:"stylePrompt,omitempty"`
	OpeningMessage string    `json:"openingMessage,omitempty"`
	LikeCount      int       `json:"likeCount"`
	Liked          bool      `json:"liked"`
	CreatedAt      time.Time `json:"-"`
	UpdatedAt      time.Time `json:"-"`
}

type Character struct {
	ID              string `json:"id"`
	StoryID         string `json:"storyId"`
	Name            string `json:"name"`
	Description     string `json:"description"`
	Personality     string `json:"personality"`
	ExampleDialogue string `json:"exampleDialogue"`
	Priority        int    `json:"priority"`
}

type WorldInfo struct {
	ID       string   `json:"id"`
	StoryID  string   `json:"storyId"`
	Keywords []string `json:"keywords"`
	Content  string   `json:"content"`
	Priority int      `json:"priority"`
	Enabled  bool     `json:"enabled"`
}

type ChatSession struct {
	ID           string    `json:"chatSessionId"`
	UserID       string    `json:"-"`
	StoryID      string    `json:"-"`
	Title        string    `json:"title"`
	Summary      string    `json:"summary"`
	MessageCount int       `json:"messageCount,omitempty"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

type Message struct {
	ID            string    `json:"id"`
	ChatSessionID string    `json:"-"`
	Role          string    `json:"role"`
	Content       string    `json:"content"`
	TokenEstimate int       `json:"tokenEstimate"`
	CreatedAt     time.Time `json:"createdAt"`
}
