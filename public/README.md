# Claude API Interface

This is a Python API interface that polls session keys from `data/sessionKeys.json` and implements a simple API compatible with the [claude2api](https://github.com/yushangxiao/claude2api) project.

## Features

- Polls `data/sessionKeys.json` for session keys
- Implements `/v1/models` endpoint
- Supports the model `claude-3-7-sonnet-20250219`
- Uses default password authentication: `xierlove`

## Setup

1. Install the required dependencies:

```bash
pip install -r requirements.txt
```

2. Make sure your `data/sessionKeys.json` file is properly formatted:

```json
{
  "sessionKeys": [
    {
      "id": 1,
      "key": "your-session-key-1"
    },
    {
      "id": 2,
      "key": "your-session-key-2"
    }
  ]
}
```

## Usage

1. Start the API server:

```bash
python app.py
```

2. The server will run on `http://localhost:5000`

3. Access the models endpoint:

```bash
curl -X GET http://localhost:5000/v1/models \
  -H "Authorization: Bearer $(echo -n 'xierlove' | sha256sum | awk '{print $1}')"
```

4. Check the health of the API:

```bash
curl -X GET http://localhost:5000/health
```

## Authentication

The API uses a simple authentication mechanism with the default password `xierlove`. The password is hashed using SHA-256 and sent as a Bearer token in the Authorization header.

Example:
```bash
# Generate the token (Linux/Mac)
TOKEN=$(echo -n 'xierlove' | shasum -a 256 | awk '{print $1}')

# Use the token
curl -X GET http://localhost:5000/v1/models \
  -H "Authorization: Bearer $TOKEN"
```
