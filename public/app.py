import json
import os
import time
import threading
import hashlib
import logging
import uuid
import requests
import random
from flask import Flask, request, jsonify, Response, stream_with_context
from flask_cors import CORS

# Configure logging
logging.basicConfig(level=logging.INFO, format='%(asctime)s - %(name)s - %(levelname)s - %(message)s')
logger = logging.getLogger(__name__)

app = Flask(__name__)
CORS(app)

# Global variables
SESSION_KEYS_FILE = os.path.join(os.path.dirname(os.path.abspath(__file__)), "data", "sessionKeys.json")
DEFAULT_PASSWORD = "xierlove"
session_keys = []
last_modified_time = 0

# Function to load session keys from file
def load_session_keys():
    global session_keys, last_modified_time
    
    try:
        current_modified_time = os.path.getmtime(SESSION_KEYS_FILE)
        
        # Only reload if the file has been modified
        if current_modified_time > last_modified_time:
            with open(SESSION_KEYS_FILE, 'r') as f:
                data = json.load(f)
                session_keys = data.get("sessionKeys", [])
                last_modified_time = current_modified_time
                logger.info(f"Loaded {len(session_keys)} session keys from file")
    except Exception as e:
        logger.error(f"Error loading session keys: {str(e)}")

# Function to poll session keys periodically
def poll_session_keys():
    while True:
        load_session_keys()
        time.sleep(60)  # Check every minute

# Authentication middleware
def authenticate():
    auth_header = request.headers.get('Authorization')
    
    if not auth_header or not auth_header.startswith('Bearer '):
        return False
    
    token = auth_header.split(' ')[1]
    hashed_password = hashlib.sha256(DEFAULT_PASSWORD.encode()).hexdigest()
    
    return token == hashed_password

# Start polling thread
threading.Thread(target=poll_session_keys, daemon=True).start()

# Load keys initially
load_session_keys()

# Function to get a session key using round-robin
def get_session_key():
    if not session_keys:
        return None
    
    # Simple round-robin selection
    key_index = int(time.time()) % len(session_keys)
    return session_keys[key_index].get("key")

# Function to get a random session key
def get_random_session_key():
    if not session_keys:
        return None
    
    # Random selection
    return random.choice(session_keys).get("key")

# Function to call Claude API using claude2api
def call_claude_api(session_key, messages, model, temperature=0.7, max_tokens=4096, stream=False):
    try:
        # Prepare the API endpoint URL for claude2api
        # Assuming claude2api is running locally on port 8080
        # In a real-world scenario, you would configure this
        claude2api_url = "http://localhost:8080/v1/chat/completions"
        
        # Prepare headers with the correct API key for claude2api
        # claude2api expects the raw API key, not a hashed value
        headers = {
            "Content-Type": "application/json",
            "Authorization": "Bearer xierlove"
        }
        
        # Prepare the request payload
        payload = {
            "model": model,
            "messages": messages,
            "temperature": temperature,
            "max_tokens": max_tokens,
            "stream": stream
        }
        
        # Make the request to claude2api
        if stream:
            response = requests.post(claude2api_url, json=payload, headers=headers, stream=True)
            response.raise_for_status()
            return response
        else:
            response = requests.post(claude2api_url, json=payload, headers=headers)
            response.raise_for_status()
            return response.json()
    
    except Exception as e:
        logger.error(f"Error calling Claude API via claude2api: {str(e)}")
        raise e

@app.route('/v1/models', methods=['GET'])
def get_models():
    if not authenticate():
        return jsonify({"error": "Unauthorized"}), 401
    
    models = {
        "object": "list",
        "data": [
            {
                "id": "claude-3-7-sonnet-20250219",
                "object": "model",
                "created": int(time.time()),
                "owned_by": "anthropic"
            }
        ]
    }
    
    return jsonify(models)

@app.route('/v1/chat/completions', methods=['POST'])
def chat_completions():
    if not authenticate():
        return jsonify({"error": "Unauthorized"}), 401
    
    # Get a session key using round-robin
    session_key = get_session_key()
    if not session_key:
        return jsonify({"error": "No session keys available"}), 503
    
    # Parse request data
    data = request.json
    model = data.get("model", "claude-3-7-sonnet-20250219")
    messages = data.get("messages", [])
    stream = data.get("stream", False)
    temperature = data.get("temperature", 0.7)
    max_tokens = data.get("max_tokens", 4096)
    
    # Log the request
    logger.info(f"Chat completion request: model={model}, stream={stream}, messages_count={len(messages)}")
    
    # Handle streaming responses
    if stream:
        def generate():
            try:
                # Call Claude API via claude2api
                response = call_claude_api(session_key, messages, model, temperature, max_tokens, stream=True)
                
                # Simply pass through the streaming response from claude2api
                for chunk in response.iter_lines():
                    if chunk:
                        yield chunk.decode('utf-8') + '\n\n'
            
            except Exception as e:
                logger.error(f"Error in streaming response: {str(e)}")
                error_msg = {"error": {"message": str(e), "type": "api_error"}}
                yield f"data: {json.dumps(error_msg)}\n\n"
            
        return Response(stream_with_context(generate()), content_type='text/event-stream')
    else:
        # Non-streaming response
        try:
            # Call Claude API via claude2api
            response = call_claude_api(session_key, messages, model, temperature, max_tokens, stream=False)
            
            # Simply return the response from claude2api
            return jsonify(response)
        except Exception as e:
            logger.error(f"Error in non-streaming response: {str(e)}")
            return jsonify({"error": {"message": str(e), "type": "api_error"}}), 500

@app.route('/health', methods=['GET'])
def health_check():
    return jsonify({"status": "healthy", "session_keys_count": len(session_keys)})

if __name__ == '__main__':
    app.run(host='0.0.0.0', port=5000, debug=False)
