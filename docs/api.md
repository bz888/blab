# API Documentation

## Overview

This document outlines the API interactions for the Bad Siri project, specifically detailing the API client module written in Go that interacts with the LLM server.

## API Endpoints

### 1. Submit Text for Processing

- **URL**: `/process-text`
- **Method**: `POST`
- **Auth Required**: No
- **Data Constraints**:
  ```json
  {
    "text": "[plain text to be processed]"
  }
  ```