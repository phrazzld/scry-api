# Text generation

The Gemini API can generate text output in response to various inputs, including text, images, video, and audio. This guide shows you how to generate text using text and image inputs. It also covers streaming, chat, and system instructions.

## Before you begin

Before calling the Gemini API, ensure you have your SDK of choice installed, and a Gemini API key configured and ready to use.

## Text input

The simplest way to generate text using the Gemini API is to provide the model with a single text-only input, as shown in this example:

```go
// import packages here

func main() {
  ctx := context.Background()
  client, err := genai.NewClient(ctx, option.WithAPIKey(os.Getenv("GEMINI_API_KEY")))
  if err != nil {
    log.Fatal(err)
  }
  defer client.Close()

  model := client.GenerativeModel("gemini-2.0-flash")
  resp, err := model.GenerateContent(ctx, genai.Text("How does AI work?"))
  if err != nil {
    log.Fatal(err)
  }
  printResponse(resp) // helper function for printing content parts
}
```

## Image input

The Gemini API supports multimodal inputs that combine text and media files. The following example shows how to generate text from text and image input:

```go
model := client.GenerativeModel("gemini-2.0-flash")

imgData, err := os.ReadFile(filepath.Join(testDataDir, "organ.jpg"))
if err != nil {
  log.Fatal(err)
}

resp, err := model.GenerateContent(ctx,
  genai.Text("Tell me about this instrument"),
  genai.ImageData("jpeg", imgData))
if err != nil {
  log.Fatal(err)
}

printResponse(resp)
```

## Streaming output

By default, the model returns a response after completing the entire text generation process. You can achieve faster interactions by using streaming to return instances of GenerateContentResponse as they're generated.

```go
model := client.GenerativeModel("gemini-1.5-flash")
iter := model.GenerateContentStream(ctx, genai.Text("Write a story about a magic backpack."))
for {
  resp, err := iter.Next()
  if err == iterator.Done {
    break
  }
  if err != nil {
    log.Fatal(err)
  }
  printResponse(resp)
}
```

## Multi-turn conversations

The Gemini SDK lets you collect multiple rounds of questions and responses into a chat. The chat format enables users to step incrementally toward answers and to get help with multipart problems. This SDK implementation of chat provides an interface to keep track of conversation history, but behind the scenes it uses the same generateContent method to create the response.

The following code example shows a basic chat implementation:

```go
model := client.GenerativeModel("gemini-1.5-flash")
cs := model.StartChat()

cs.History = []*genai.Content{
  {
    Parts: []genai.Part{
      genai.Text("Hello, I have 2 dogs in my house."),
    },
    Role: "user",
  },
  {
    Parts: []genai.Part{
      genai.Text("Great to meet you. What would you like to know?"),
    },
    Role: "model",
  },
}

res, err := cs.SendMessage(ctx, genai.Text("How many paws are in my house?"))
if err != nil {
  log.Fatal(err)
}
printResponse(res)
```

You can also use streaming with chat, as shown in the following example:

```go
model := client.GenerativeModel("gemini-1.5-flash")
cs := model.StartChat()

cs.History = []*genai.Content{
  {
    Parts: []genai.Part{
      genai.Text("Hello, I have 2 dogs in my house."),
    },
    Role: "user",
  },
  {
    Parts: []genai.Part{
      genai.Text("Great to meet you. What would you like to know?"),
    },
    Role: "model",
  },
}

iter := cs.SendMessageStream(ctx, genai.Text("How many paws are in my house?"))
for {
  resp, err := iter.Next()
  if err == iterator.Done {
    break
  }
  if err != nil {
    log.Fatal(err)
  }
  printResponse(resp)
}
```

## Configuration parameters

Every prompt you send to the model includes parameters that control how the model generates responses. You can configure these parameters, or let the model use the default options.

The following example shows how to configure model parameters:

```go
model := client.GenerativeModel("gemini-1.5-pro-latest")
model.SetTemperature(0.9)
model.SetTopP(0.5)
model.SetTopK(20)
model.SetMaxOutputTokens(100)
model.SystemInstruction = genai.NewUserContent(genai.Text("You are Yoda from Star Wars."))
model.ResponseMIMEType = "application/json"
resp, err := model.GenerateContent(ctx, genai.Text("What is the average size of a swallow?"))
if err != nil {
  log.Fatal(err)
}
printResponse(resp)
```

Here are some of the model parameters you can configure. (Naming conventions vary by programming language.)

- **stopSequences**: Specifies the set of character sequences (up to 5) that will stop output generation. If specified, the API will stop at the first appearance of a stop_sequence. The stop sequence won't be included as part of the response.
- **temperature**: Controls the randomness of the output. Use higher values for more creative responses, and lower values for more deterministic responses. Values can range from [0.0, 2.0].
- **maxOutputTokens**: Sets the maximum number of tokens to include in a candidate.
- **topP**: Changes how the model selects tokens for output. Tokens are selected from the most to least probable until the sum of their probabilities equals the topP value. The default topP value is 0.95.
- **topK**: Changes how the model selects tokens for output. A topK of 1 means the selected token is the most probable among all the tokens in the model's vocabulary, while a topK of 3 means that the next token is selected from among the 3 most probable using the temperature. Tokens are further filtered based on topP with the final token selected using temperature sampling.

## System instructions

System instructions let you steer the behavior of a model based on your specific use case. When you provide system instructions, you give the model additional context to help it understand the task and generate more customized responses. The model should adhere to the system instructions over the full interaction with the user, enabling you to specify product-level behavior separate from the prompts provided by end users.

You can set system instructions when you initialize your model:

```go
// import packages here

func main() {
  ctx := context.Background()
  client, err := genai.NewClient(ctx, option.WithAPIKey(os.Getenv("GEMINI_API_KEY")))
  if err != nil {
    log.Fatal(err)
  }
  defer client.Close()

  model := client.GenerativeModel("gemini-2.0-flash")
  model.SystemInstruction = &genai.Content{
    Parts: []genai.Part{genai.Text(`
      You are a cat. Your name is Neko.
    `)},
  }
  resp, err := model.GenerateContent(ctx, genai.Text("Hello there"))
  if err != nil {
    log.Fatal(err)
  }
  printResponse(resp) // helper function for printing content parts
}
```

Then, you can send requests to the model as usual.

## Supported models

The entire Gemini family of models supports text generation. To learn more about the models and their capabilities, see Models.

## Prompting tips

For basic text generation use cases, your prompt might not need to include any output examples, system instructions, or formatting information. This is a zero-shot approach. For some use cases, a one-shot or few-shot prompt might produce output that's more aligned with user expectations. In some cases, you might also want to provide system instructions to help the model understand the task or follow specific guidelines.
