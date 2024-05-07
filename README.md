# Bad Siri

voice command prompt application that converts spoken language into text, processes it through a large language model (LLM), and returns the processed response. This project uses Python for speech recognition and Go for API interactions.

## Components

### Speech Recognition Module
This module uses the microphone to capture audio input and converts it into text.
- **Packages**:
    - `speech_recognition`: Main library for recognizing speech.
    - `PyAudio`: Used for microphone input in conjunction with `speech_recognition`.

### API Client Module
This module sends the processed text to a GCP VM running the LLM and retrieves the response.
- **Packages**:
    - `net/http`: Standard library in Go for HTTP client and server implementations.
    - `encoding/json`: For encoding and decoding JSON, used to format requests and parse responses.

### VM with LLM
Hosts the server that processes the input text using a large language model and returns generated text.
- **Packages**:
    - `transformers`: Provides the models and utilities for handling LLMs like tokenization and embedding.
    - `flask`: To set up an API server that can accept requests and send responses.

### Output
The response from the LLM is displayed in the terminal.
- **Technology**: Python
- **Function**: Uses basic print statements to display the output.

## Workflow Diagram

```plaintext
+------------+      +-------------+      +------------+      +------------------+      +-----------+      +-------------+
|            |      |             |      |            |      |                  |      |           |      |             |
| Input Text | ---> | Tokenization| ---> | Embeddings | ---> | Attention        | ---> | Decoding  | ---> | Output Text |
|            |      |             |      |            |      | Mechanisms       |      |           |      |             |
+------------+      +-------------+      +------------+      +------------------+      +-----------+      +-------------+
```

## Getting Started

### Prerequisites
- Python 3.x installed with `pip`
- Go 1.x installed
- Access to Google Cloud Platform for deploying a VM

### Setup and Installation
1. **Python Environment Setup**
    - Install the required Python libraries:
      ```bash
      pip install speech_recognition pyaudio transformers flask
      ```
2. **Go Environment Setup**
    - Set up the Go workspace and ensure Go is properly installed.

3. **VM Setup**
    - Deploy a GCP VM.
    - Install Python and required libraries like `transformers` and `flask`.

### Running the Application
1. Start the speech recognition module in Python.
2. Run the Go API client to handle the text input and output.
3. Ensure the LLM server on the GCP VM is operational and can receive requests.