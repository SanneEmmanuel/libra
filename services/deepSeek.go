package services

import (
    "bytes"
    "io"
    "mime/multipart"
    "net/http"
    "os"
)

const deepSeekURL = "https://api.deepseek.com/analyze" // update this
const apiKey = "YOUR_DEEPSEEK_API_KEY"

func CallDeepSeekAPI(command string, file *os.File) ([]byte, error) {
    body := &bytes.Buffer{}
    writer := multipart.NewWriter(body)

    _ = writer.WriteField("command", command)

    part, err := writer.CreateFormFile("attachment", file.Name())
    if err != nil {
        return nil, err
    }

    _, err = io.Copy(part, file)
    if err != nil {
        return nil, err
    }

    writer.Close()

    req, err := http.NewRequest("POST", deepSeekURL, body)
    if err != nil {
        return nil, err
    }

    req.Header.Set("Authorization", "Bearer "+apiKey)
    req.Header.Set("Content-Type", writer.FormDataContentType())

    client := &http.Client{}
    resp, err := client.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    return io.ReadAll(resp.Body)
}
