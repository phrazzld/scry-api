# Explore document processing capabilities with the Gemini API

The Gemini API supports PDF input, including long documents (up to 3600 pages). Gemini models process PDFs with native vision, and are therefore able to understand both text and image contents inside documents. With native PDF vision support, Gemini models are able to:

- Analyze diagrams, charts, and tables inside documents.
- Extract information into structured output formats.
- Answer questions about visual and text contents in documents.
- Summarize documents.
- Transcribe document content (e.g. to HTML) preserving layouts and formatting, for use in downstream applications (such as in RAG pipelines).

This tutorial demonstrates some possible ways to use the Gemini API with PDF documents. All output is text-only.

## Before you begin

Before calling the Gemini API, ensure you have your SDK of choice installed, and a Gemini API key configured and ready to use.

## Prompting with PDFs

This guide demonstrates how to upload and process PDFs using the File API or by including them as inline data.

### Technical details

Gemini 1.5 Pro and 1.5 Flash support a maximum of 3,600 document pages. Document pages must be in one of the following text data MIME types:

- PDF - application/pdf
- JavaScript - application/x-javascript, text/javascript
- Python - application/x-python, text/x-python
- TXT - text/plain
- HTML - text/html
- CSS - text/css
- Markdown - text/md
- CSV - text/csv
- XML - text/xml
- RTF - text/rtf

Each document page is equivalent to 258 tokens.

While there are no specific limits to the number of pixels in a document besides the model's context window, larger pages are scaled down to a maximum resolution of 3072x3072 while preserving their original aspect ratio, while smaller pages are scaled up to 768x768 pixels. There is no cost reduction for pages at lower sizes, other than bandwidth, or performance improvement for pages at higher resolution.

For best results:

- Rotate pages to the correct orientation before uploading.
- Avoid blurry pages.
- If using a single page, place the text prompt after the page.

## PDF input

For PDF payloads under 20MB, you can choose between uploading base64 encoded documents or directly uploading locally stored files.

### As inline data

You can process PDF documents directly from URLs. Here's a code snippet showing how to do this:

```go
package main

import (
    "context"
    "fmt"
    "io"
    "log"
    "net/http"
    "os"

    "github.com/google/generative-ai-go/genai"
    "google.golang.org/api/option"
)

func main() {
    ctx := context.Background()
    // Access your API key as an environment variable
    client, err := genai.NewClient(ctx, option.WithAPIKey(os.Getenv("GEMINI_API_KEY")))
    if err != nil {
        log.Fatal(err)
    }
    defer client.Close()

    model := client.GenerativeModel("gemini-1.5-flash")

    // Download the pdf.
    pdfResp, err := http.Get("https://discovery.ucl.ac.uk/id/eprint/10089234/1/343019_3_art_0_py4t4l_convrt.pdf")
    if err != nil {
        panic(err)
    }
    defer pdfResp.Body.Close()

    pdfBytes, err := io.ReadAll(pdfResp.Body)
    if err != nil {
        panic(err)
    }

    // Create the request.
    req := []genai.Part{
        genai.Blob{MIMEType: "application/pdf", Data: pdfBytes},

        genai.Text("Summarize this document"),
    }

    // Generate content.
    resp, err := model.GenerateContent(ctx, req...)
    if err != nil {
        panic(err)
    }

    // Handle the response of generated text.
    for _, c := range resp.Candidates {
        if c.Content != nil {
            fmt.Println(*c.Content)
        }
    }
}
```

### Locally stored PDFs

For locally stored PDFs, you can use the following approach:

```go
package genai

import (
    "context"
    "fmt"
    "log"
    "os"

    "github.com/google/generative-ai-go/genai"
    "google.golang.org/api/option"
)

func main() {
    ctx := context.Background()
    // Access your API key as an environment variable
    client, err := genai.NewClient(ctx, option.WithAPIKey(os.Getenv("GEMINI_API_KEY")))
    if err != nil {
        log.Fatal(err)
    }
    defer client.Close()

    model := client.GenerativeModel("gemini-1.5-flash")

    pdfBytes, err := os.ReadFile("/content/343019_3_art_0_py4t4l_convrt.pdf")
    if err != nil {
        log.Fatal(err)
    }

    // Create the request.
    req := []genai.Part{
        genai.Blob{MIMEType: "application/pdf", Data: pdfBytes},

        genai.Text("Summarize this document"),
    }

    // Generate content.
    resp, err := model.GenerateContent(ctx, req...)
    if err != nil {
        panic(err)
    }

    // Handle the response of generated text.
    for _, c := range resp.Candidates {
        if c.Content != nil {
            fmt.Println(*c.Content)
        }
    }
}
```

## Large PDFs

You can use the File API to upload a document of any size. Always use the File API when the total request size (including the files, text prompt, system instructions, etc.) is larger than 20 MB.

> Note: The File API lets you store up to 20 GB of files per project, with a per-file maximum size of 2 GB. Files are stored for 48 hours. They can be accessed in that period with your API key, but cannot be downloaded from the API. The File API is available at no cost in all regions where the Gemini API is available.

Call media.upload to upload a file using the File API. The following code uploads a document file and then uses the file in a call to models.generateContent.

### Large PDFs from URLs

Use the File API for large PDF files available from URLs, simplifying the process of uploading and processing these documents directly through their URLs:

```go
package main

import (
    "context"
    "fmt"
    "io"
    "log"
    "net/http"
    "os"

    "github.com/google/generative-ai-go/genai"
    "google.golang.org/api/option"
)

func main() {
    ctx := context.Background()
    // Access your API key as an environment variable
    client, err := genai.NewClient(ctx, option.WithAPIKey(os.Getenv("GEMINI_API_KEY")))
    if err != nil {
        log.Fatal(err)
    }
    defer client.Close()

    model := client.GenerativeModel("gemini-1.5-flash")

    // Create the file
    pdfPath := "A17_FlightPlan.pdf"
    pdfFile, err := os.Create(pdfPath)
    if err != nil {
        log.Fatal(err)
    }
    defer pdfFile.Close()

    // Download the pdf.
    pdfResp, err := http.Get("https://www.nasa.gov/wp-content/uploads/static/history/alsj/a17/A17_FlightPlan.pdf")
    if err != nil {
        log.Fatal(err)
    }
    defer pdfResp.Body.Close()

    // Save the file
    _, err = io.Copy(pdfFile, pdfResp.Body)
    if err != nil {
        log.Fatal(err)
    }

    file, err := client.UploadFileFromPath(ctx, pdfPath, nil)
    if err != nil {
        log.Fatal(err)
    }
    defer client.DeleteFile(ctx, file.Name)

    // Create the request.
    req := []genai.Part{
        genai.FileData{URI: file.URI},

        genai.Text("Summarize this document"),
    }

    // Generate content.
    resp, err := model.GenerateContent(ctx, req...)
    if err != nil {
        panic(err)
    }

    // Handle the response of generated text.
    for _, c := range resp.Candidates {
        if c.Content != nil {
            fmt.Println(*c.Content)
        }
    }
}
```

### Large PDFs stored locally

```go
file, err := client.UploadFileFromPath(ctx, filepath.Join(testDataDir, "test.pdf"), nil)
if err != nil {
	log.Fatal(err)
}
defer client.DeleteFile(ctx, file.Name)

model := client.GenerativeModel("gemini-1.5-flash")
resp, err := model.GenerateContent(ctx,
	genai.Text("Give me a summary of this pdf file."),
	genai.FileData{URI: file.URI})
if err != nil {
	log.Fatal(err)
}

printResponse(resp)
```

You can verify the API successfully stored the uploaded file and get its metadata by calling files.get. Only the name (and by extension, the uri) are unique.

```go
file, err := client.UploadFileFromPath(ctx, filepath.Join(testDataDir, "personWorkingOnComputer.jpg"), nil)
if err != nil {
	log.Fatal(err)
}
defer client.DeleteFile(ctx, file.Name)

gotFile, err := client.GetFile(ctx, file.Name)
if err != nil {
	log.Fatal(err)
}
fmt.Println("Got file:", gotFile.Name)

model := client.GenerativeModel("gemini-1.5-flash")
resp, err := model.GenerateContent(ctx,
	genai.FileData{URI: file.URI},
	genai.Text("Describe this image"))
if err != nil {
	log.Fatal(err)
}

printResponse(resp)
```

## Multiple PDFs

The Gemini API is capable of processing multiple PDF documents in a single request, as long as the combined size of the documents and the text prompt stays within the model's context window.

```go
package main

import (
    "context"
    "fmt"
    "io"
    "log"
    "net/http"
    "os"

    "github.com/google/generative-ai-go/genai"
    "google.golang.org/api/option"
)

func main() {
    ctx := context.Background()
    // Access your API key as an environment variable
    client, err := genai.NewClient(ctx, option.WithAPIKey(os.Getenv("GEMINI_API_KEY")))
    if err != nil {
        log.Fatal(err)
    }
    defer client.Close()

    model := client.GenerativeModel("gemini-1.5-flash")

    docUrl1 := "https://arxiv.org/pdf/2312.11805"
    docUrl2 := "https://arxiv.org/pdf/2403.05530"

    // Create the file
    doc1Path := "doc1.pdf"
    doc1File, err := os.Create(doc1Path)
    if err != nil {
        log.Fatal(err)
    }
    defer doc1File.Close()

    doc2Path := "doc2.pdf"
    doc2File, err := os.Create(doc2Path)
    if err != nil {
        log.Fatal(err)
    }
    defer doc2File.Close()

    doc1Resp, err := http.Get(docUrl1)
    if err != nil {
        log.Fatal(err)
    }
    defer doc1Resp.Body.Close()

    doc2Resp, err := http.Get(docUrl2)
    if err != nil {
        log.Fatal(err)
    }
    defer doc2Resp.Body.Close()

    // Save the file
    _, err = io.Copy(doc1File, doc1Resp.Body)
    if err != nil {
        log.Fatal(err)
    }

    _, err = io.Copy(doc2File, doc2Resp.Body)
    if err != nil {
        log.Fatal(err)
    }

    doc1, err := client.UploadFileFromPath(ctx, doc1Path, nil)
    if err != nil {
        log.Fatal(err)
    }
    defer client.DeleteFile(ctx, doc1.Name)

    doc2, err := client.UploadFileFromPath(ctx, doc2Path, nil)
    if err != nil {
        log.Fatal(err)
    }
    defer client.DeleteFile(ctx, doc2.Name)

    // Create the request.
    req := []genai.Part{
        genai.FileData{URI: doc1.URI},
        genai.FileData{URI: doc2.URI},

        genai.Text("What is the difference between each of the main benchmarks between these two papers? Output these in a table."),
    }

    // Generate content.
    resp, err := model.GenerateContent(ctx, req...)
    if err != nil {
        panic(err)
    }

    // Handle the response of generated text.
    for _, c := range resp.Candidates {
        if c.Content != nil {
            fmt.Println(*c.Content)
        }
    }
}
```

## List files

You can list all files uploaded using the File API and their URIs using files.list.

```go
iter := client.ListFiles(ctx)
for {
	ifile, err := iter.Next()
	if err == iterator.Done {
		break
	}
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(ifile.Name)
}
```

## Delete files

Files uploaded using the File API are automatically deleted after 2 days. You can also manually delete them using files.delete.

```go
file, err := client.UploadFileFromPath(ctx, filepath.Join(testDataDir, "personWorkingOnComputer.jpg"), nil)
if err != nil {
	log.Fatal(err)
}
defer client.DeleteFile(ctx, file.Name)

gotFile, err := client.GetFile(ctx, file.Name)
if err != nil {
	log.Fatal(err)
}
fmt.Println("Got file:", gotFile.Name)

model := client.GenerativeModel("gemini-1.5-flash")
resp, err := model.GenerateContent(ctx,
	genai.FileData{URI: file.URI},
	genai.Text("Describe this image"))
if err != nil {
	log.Fatal(err)
}

printResponse(resp)
```

## Context caching with PDFs

```go
package main

import (
    "context"
    "fmt"
    "io"
    "log"
    "net/http"
    "os"

    "github.com/google/generative-ai-go/genai"
    "google.golang.org/api/option"
)

func main() {
    ctx := context.Background()
    // Access your API key as an environment variable
    client, err := genai.NewClient(ctx, option.WithAPIKey(os.Getenv("GEMINI_API_KEY")))
    if err != nil {
        log.Fatal(err)
    }
    defer client.Close()

    // Create the file
    pdfPath := "A17_FlightPlan.pdf"
    pdfFile, err := os.Create(pdfPath)
    if err != nil {
        log.Fatal(err)
    }
    defer pdfFile.Close()

    // Download the pdf.
    pdfResp, err := http.Get("https://www.nasa.gov/wp-content/uploads/static/history/alsj/a17/A17_FlightPlan.pdf")
    if err != nil {
        log.Fatal(err)
    }
    defer pdfResp.Body.Close()

    // Save the file
    _, err = io.Copy(pdfFile, pdfResp.Body)
    if err != nil {
        log.Fatal(err)
    }

    file, err := client.UploadFileFromPath(ctx, pdfPath, nil)
    if err != nil {
        log.Fatal(err)
    }
    defer client.DeleteFile(ctx, file.Name)

    fd := genai.FileData{URI: file.URI}

    argcc := &genai.CachedContent{
        Model:             "gemini-1.5-flash-001",
        SystemInstruction: genai.NewUserContent(genai.Text("You are an expert analyzing transcripts.")),
        Contents:          []*genai.Content{genai.NewUserContent(fd)},
    }
    cc, err := client.CreateCachedContent(ctx, argcc)
    if err != nil {
        log.Fatal(err)
    }
    defer client.DeleteCachedContent(ctx, cc.Name)

    // Create the request.
    req := []genai.Part{
        genai.Text("Please summarize this transcript"),
    }

    model := client.GenerativeModelFromCachedContent(cc)

    // Generate content.
    resp, err := model.GenerateContent(ctx, req...)
    if err != nil {
        panic(err)
    }

    // Handle the response of generated text.
    for _, c := range resp.Candidates {
        if c.Content != nil {
            fmt.Println(*c.Content)
        }
    }
}
```

## List caches

It's not possible to retrieve or view cached content, but you can retrieve cache metadata (name, model, display_name, usage_metadata, create_time, update_time, and expire_time).

```go
fmt.Println("My caches:")
iter := client.ListCachedContents(ctx)
for {
  cc, err := iter.Next()
  if err == iterator.Done {
    break
  }
  if err != nil {
    log.Fatal(err)
  }
  fmt.Println("   ", cc.Name)
}
```

## Update a cache

You can set a new ttl or expire_time for a cache. Changing anything else about the cache isn't supported.

```go
newExpireTime := cc.Expiration.ExpireTime.Add(2 * time.Hour)
_, err = client.UpdateCachedContent(ctx, cc, &genai.CachedContentToUpdate{
  Expiration: &genai.ExpireTimeOrTTL{ExpireTime: newExpireTime}})
if err != nil {
  log.Fatal(err)
}
```

## Delete a cache

The caching service provides a delete operation for manually removing content from the cache.

```go
defer client.DeleteCachedContent(ctx, cc.Name)
```
