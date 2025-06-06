package services

import (
    "bytes"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "strings"
)

const chatModel = "deepseek-chat"
const chatEndpoint = "https://api.deepseek.com/chat/completions"

type chatRequest struct {
    Model    string        `json:"model"`
    Messages []chatMessage `json:"messages"`
    Stream   bool          `json:"stream"`
}

type chatMessage struct {
    Role    string `json:"role"`
    Content string `json:"content"`
}

type chatResponse struct {
    Choices []struct {
        Message chatMessage `json:"message"`
    } `json:"choices"`
}

// LibraChat sends a chat request to DeepSeek with streaming option
func LibraChat(userText string, stream bool) (string, error) {
    requestBody := chatRequest{
        Model:  chatModel,
        Stream: stream,
        Messages: []chatMessage{
            {Role: "system", Content: "You are a helpful assistant."},
            {Role: "user", Content: userText},
        },
    }

    jsonData, err := json.Marshal(requestBody)
    if err != nil {
        return "", err
    }

    req, err := http.NewRequest("POST", chatEndpoint, bytes.NewBuffer(jsonData))
    if err != nil {
        return "", err
    }

    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("Authorization", "Bearer sk-280ae96e3d76417194c421ae83125498")

    client := &http.Client{}
    resp, err := client.Do(req)
    if err != nil {
        return "", err
    }
    defer resp.Body.Close()

    if stream {
        // Read streamed chunks and concatenate
        var responseBuilder strings.Builder
        buf := make([]byte, 1024)
        for {
            n, err := resp.Body.Read(buf)
            if n > 0 {
                responseBuilder.Write(buf[:n])
            }
            if err == io.EOF {
                break
            }
            if err != nil {
                return "", fmt.Errorf("stream read error: %v", err)
            }
        }
        return responseBuilder.String(), nil
    }

    // Normal response
    body, err := io.ReadAll(resp.Body)
    if err != nil {
        return "", err
    }

    var parsed chatResponse
    if err := json.Unmarshal(body, &parsed); err != nil {
        return "", fmt.Errorf("invalid JSON from LibraAI source: %v", err)
    }

    if len(parsed.Choices) == 0 {
        return "", fmt.Errorf("no response from LibraAI source")
    }

    return parsed.Choices[0].Message.Content, nil
}
