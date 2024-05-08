# Project Architecture

## Overview

This document provides an overview of the architecture for the Bad Siri project. The project is designed to convert spoken language into text, process it through a large language model (LLM), and return the processed response.

## Components

### 1. Speech Recognition Module

- **Description**: This module uses the `speech_recognition` library in Python to capture audio input from the microphone and convert it into text.
- **Technologies Used**:
    - Python
    - speech_recognition
    - PyAudio

### 2. API Client Module

- **Description**: Written in Go, this module sends the text to a GCP VM that runs the LLM and retrieves the response.
- **Technologies Used**:
    - Go
    - net/http
    - encoding/json

### 3. LLM Server

- **Description**: A Python Flask application running on a GCP VM. It processes the input text using a large language model and returns the generated text.
- **Technologies Used**:
    - Python
    - Flask
    - Transformers from Hugging Face

### 4. Output Handling

- **Description**: The final output is displayed in the terminal. This simple process is managed within the speech recognition module.
- **Technologies Used**:
    - Python

## Data Flow Diagram

Include a data flow diagram here that illustrates how data moves through the system.

## Security Considerations

Discuss any security considerations here, such as how data is protected during transmission and any authentication measures.

## Scalability and Performance

Outline how the architecture supports scalability and what measures are in place to ensure performance under load.

## Future Enhancements

Discuss potential future enhancements to the architecture.

