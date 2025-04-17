# Gemini API quickstart

This quickstart shows you how to install your SDK of choice and then make your first Gemini API request.

## Install the Gemini API library

> Note: We're rolling out a new set of Gemini API libraries, the Google Gen AI SDK.

Using Go 1.20+, install the generative-ai-go package in your module directory using the go get command:

```go
go get github.com/google/generative-ai-go
```

## Make your first request

1. Get a Gemini API key in Google AI Studio

2. Use the generateContent method to send a request to the Gemini API.

```go
model := client.GenerativeModel("gemini-2.0-flash")
resp, err := model.GenerateContent(ctx, genai.Text("Explain how AI works in a few words"))
if err != nil {
    log.Fatal(err)
}

printResponse(resp)
```
