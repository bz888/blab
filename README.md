# Blab

Blab is a Terminal User Interface (TUI) built with Golang for Large Language Model (LLM) chat. It uses the `ollama` package to run large language models locally.

Uploading Screen Recording 2025-01-14 at 7.14.47 PM.mov…



## macOs
note: To use voice recognition, `onnxruntime` must be installed
- **ONNXRuntime**: Install onnxruntime using Homebrew.
  ```shell
  brew install onnxruntime
  ```
  ```shell
  export LIBRARY_PATH=/opt/homebrew/Cellar/onnxruntime/1.17.1/lib
  ```
- **Ollama**: Install `ollama` refer to: [ollama docs](https://github.com/ollama/ollama)

## Quickstart
Ensure `ollama` is running.
create `.env` 
```shell
API_KEY=GOOGLE_API_KEY
```

```shell
git clone https://github.com/bz888/blab.git
```
```shell
./blab
```


## Usage
flags:
- `-dev`: Enables the log console on startup. (example: `blab -dev`)
- `-logPath=<path>`: Directory path for logFile output. (example: `blab -logPath="./"`)

In-app:
- `/help`: Display this help message.
- `/bye`: Exit the application.
- `/debug`: Toggle the debug console.
- `/voice`: Activate voice input.
- `/models`: Select between local LLMs.
