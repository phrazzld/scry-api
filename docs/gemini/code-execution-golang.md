# Code Execution with the Gemini API

The Gemini API code execution feature enables the model to generate and run Python code and learn iteratively from the results until it arrives at a final output. You can use this code execution capability to build applications that benefit from code-based reasoning and that produce text output. For example, you could use code execution in an application that solves equations or processes text.

**Note**: Gemini is only able to execute code in Python. You can still ask Gemini to generate code in another language, but the model can't use the code execution tool to run it.

Code execution is available in both AI Studio and the Gemini API. In AI Studio, you can enable code execution in the right panel under Tools. The Gemini API provides code execution as a tool, similar to function calling. After you add code execution as a tool, the model decides when to use it.

The code execution environment includes the following libraries: `altair`, `chess`, `cv2`, `matplotlib`, `mpmath`, `numpy`, `pandas`, `pdfminer`, `reportlab`, `seaborn`, `sklearn`, `statsmodels`, `striprtf`, `sympy`, and `tabulate`. You can't install your own libraries.

**Note**: Only `matplotlib` is supported for graph rendering using code execution.

## Before You Begin
Before calling the Gemini API, ensure you have your SDK of choice installed, and a Gemini API key configured and ready to use.

## Get Started with Code Execution

### Enable Code Execution on the Model
You can enable code execution on the model, as shown in the following example:

```go
package main

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
    client, err := genai.NewClient(ctx, option.WithAPIKey(os.Getenv("API_KEY")))
    if err != nil {
        log.Fatal(err)
    }
    defer client.Close()

    model := client.GenerativeModel("gemini-1.5-pro")
    // To enable code execution, set the `CodeExecution` tool.
    model.Tools = []*genai.Tool{
        {CodeExecution: &genai.CodeExecution{}},
    }
    resp, err := model.GenerateContent(ctx, genai.Text(`
        What is the sum of the first 50 prime numbers?
        Generate and run code for the calculation, and make sure you get all 50.
        `))
    if err != nil {
        log.Fatal(err)
    }
    // The model will generate code to solve the problem, which is returned in an
    // `ExecutableCode` part. Itершен

    // will also run that code and use the result,
    // which is returned in a `CodeExecutionResult` part.
    printResponse(resp)
}

func printResponse(resp *genai.GenerateContentResponse) {
    for _, cand := range resp.Candidates {
        if cand.Content != nil {
            for _, part := range cand.Content.Parts {
                fmt.Println(part)
            }
        }
    }
    fmt.Println("---")
}
```

The output might look something like this:

```
Thoughts:
I need to write a program that identifies if a number is prime. Then I need to
keep track of how many primes I've found, adding them as I go, and stop when I
get to 50.

&{ExecutableCodePython
def is_prime(n):
    """Returns True if n is a prime number, False otherwise."""
    if n <= 1:
        return False
    for i in range(2, int(n**0.5) + 1):
        if n % i == 0:
            return False
    return True

count = 0
sum_primes = 0
i = 2

while count < 50:
    if is_prime(i):
        sum_primes += i
        count += 1
    i += 1

print(f"The sum of the first 50 prime numbers is: {sum_primes}")
}

&{CodeExecutionResultOutcomeOK The sum of the first 50 prime numbers is: 5117
}

Findings: The sum of the first 50 prime numbers is 5117.

---
```

Calling `client.GenerativeModel` is inexpensive, so you can create as many model instances as you want and configure the `CodeExecution` tool as needed.

### Use Code Execution in Chat
You can also use code execution as part of a chat, as shown in the following example:

```go
ctx := context.Background()
client, err := genai.NewClient(ctx, option.WithAPIKey(os.Getenv("API_KEY")))
if err != nil {
  log.Fatal(err)
}
defer client.Close()

model := client.GenerativeModel("gemini-1.5-pro")
// To enable code execution, set the `CodeExecution` tool.
model.Tools = []*genai.Tool{
  {CodeExecution: &genai.CodeExecution{}},
}

cs := model.StartChat()
res, err := cs.SendMessage(ctx, genai.Text(`
  What is the sum of the first 50 prime numbers?
  Generate and run code for the calculation, and make sure you get all 50.
`))
if err != nil {
  log.Fatal(err)
}

// do something with `res`
```

### Input/Output (I/O)
Starting with Gemini 2.0 Flash, code execution supports file input and graph output. Using these new input and output capabilities, you can upload CSV and text files, ask questions about the files, and have Matplotlib graphs generated as part of the response.

#### I/O Pricing
When using code execution I/O, you're charged for input tokens and output tokens:
- **Input tokens**:
  - User prompt
- **Output tokens**:
  - Code generated by the model
  - Code execution output in the code environment
  - Summary generated by the model

#### I/O Details
When you're working with code execution I/O, be aware of the following technical details:
- The maximum runtime of the code environment is 30 seconds.
- If the code environment generates an error, the model may decide to regenerate the code output. This can happen up to 5 times.
- The maximum file input size is limited by the model token window. In AI Studio, using Gemini Flash 2.0, the maximum input file size is 1 million tokens (roughly 2MB for text files of the supported input types). If you upload a file that's too large, AI Studio won't let you send it.

| Feature                     | Single Turn                                   | Bidirectional (Multimodal Live API)         |
|-----------------------------|-----------------------------------------------|--------------------------------------------|
| **Models supported**        | All Gemini 2.0 models                         | Only Flash experimental models             |
| **File input types**        | .png, .jpeg, .csv, .xml, .cpp, .java, .py, .js, .ts | .png, .jpeg, .csv, .xml, .cpp, .java, .py, .js, .ts |
| **Plotting libraries**      | Matplotlib                                    | Matplotlib                                 |
| **Multi-tool use**          | No                                            | Yes                                        |

## Billing
There's no additional charge for enabling code execution from the Gemini API. You'll be billed at the current rate of input and output tokens based on the Gemini model you're using.

Here are a few other things to know about billing for code execution:
- You're only billed once for the input tokens you pass to the model, and you're billed for the final output tokens returned to you by the model.
- Tokens representing generated code are counted as output tokens. Generated code can include text and multimodal output like images.
- Code execution results are also counted as output tokens.

The billing model works as follows:
- You're billed at the current rate of input and output tokens based on the Gemini model you're using.
- If Gemini uses code execution when generating your response, the original prompt, the generated code, and the result of the executed code are labeled **intermediate tokens** and are billed as input tokens.
- Gemini then generates a summary and returns the generated code, the result of the executed code, and the final summary. These are billed as output tokens.
- The Gemini API includes an intermediate token count in the API response, so you know why you're getting additional input tokens beyond your initial prompt.

## Limitations
- The model can only generate and execute code. It can't return other artifacts like media files.
- In some cases, enabling code execution can lead to regressions in other areas of model output (for example, writing a story).
- There is some variation in the ability of the different models to use code execution successfully.
