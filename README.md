# Blab

Blab is a Terminal User Interface (TUI) built with Golang for Large Language Model (LLM) chat. It uses the `ollama` package to run large language models locally.

## macOs

- **FLAC**: Install FLAC using Homebrew.
  ```sh
  brew install flac
  ```
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
create `.env` under `app/cmd` 
```shell
API_KEY=GOOGLE_API_KEY
```

```shell
git clone https://github.com/bz888/blab.git
cd app
```
```shell
./blab
```


## Usage
flags:
- `-dev`: Enables the log console on startup. (example: `blab -dev`)

In-app:
- `/help`: Display this help message.
- `/bye`: Exit the application.
- `/debug`: Toggle the debug console.
- `/voice`: Activate voice input.
- `/models`: Select between local LLMs.