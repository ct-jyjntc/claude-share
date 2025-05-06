const express = require('express');
const cors = require('cors');
const fs = require('fs-extra');
const path = require('path');

const app = express();
const PORT = 3001;

// Middleware
app.use(cors());
app.use(express.json());
app.use(express.static('dist')); // Serve the built React app

// Data file path
const dataFilePath = path.join(__dirname, 'public', 'data', 'sessionKeys.json');

// API Routes
// Get all session keys
app.get('/api/sessionKeys', async (req, res) => {
  try {
    const data = await fs.readJson(dataFilePath);
    res.json(data);
  } catch (error) {
    console.error('Error reading session keys:', error);
    res.status(500).json({ error: 'Failed to read session keys' });
  }
});

// Update session keys
app.post('/api/sessionKeys', async (req, res) => {
  try {
    const { sessionKeys } = req.body;

    if (!Array.isArray(sessionKeys)) {
      return res.status(400).json({ error: 'Invalid data format' });
    }

    await fs.writeJson(dataFilePath, { sessionKeys }, { spaces: 2 });
    res.json({ success: true, sessionKeys });
  } catch (error) {
    console.error('Error saving session keys:', error);
    res.status(500).json({ error: 'Failed to save session keys' });
  }
});

// Mock OpenAI API endpoint for chat completions
app.post('/v1/chat/completions', (req, res) => {
  // Check for basic authentication
  const authHeader = req.headers.authorization;
  if (!authHeader || !authHeader.startsWith('Basic ')) {
    return res.status(401).json({ error: 'Authentication required' });
  }

  // Decode the Basic auth credentials
  const base64Credentials = authHeader.split(' ')[1];
  const credentials = Buffer.from(base64Credentials, 'base64').toString('utf-8');
  // Extract password, ignore username
  const [, password] = credentials.split(':');

  // Check if password matches the default
  if (password !== 'xierlove') {
    return res.status(401).json({ error: 'Invalid credentials' });
  }

  // Mock response for the chat completions API
  res.json({
    id: 'chatcmpl-' + Math.random().toString(36).substring(2, 12),
    object: 'chat.completion',
    created: Math.floor(Date.now() / 1000),
    model: 'claude-3-opus-20240229',
    choices: [
      {
        index: 0,
        message: {
          role: 'assistant',
          content: 'This is a mock response from the API. Your API URL is working correctly with the default password: xierlove'
        },
        finish_reason: 'stop'
      }
    ],
    usage: {
      prompt_tokens: 10,
      completion_tokens: 20,
      total_tokens: 30
    }
  });
});

// Start the server
app.listen(PORT, '0.0.0.0', () => {
  console.log(`Server running on port ${PORT}`);
  console.log(`Local: http://localhost:${PORT}`);

  // Get the local IP address to display
  const { networkInterfaces } = require('os');
  const nets = networkInterfaces();

  for (const name of Object.keys(nets)) {
    for (const net of nets[name]) {
      // Skip over non-IPv4 and internal (i.e. 127.0.0.1) addresses
      if (net.family === 'IPv4' && !net.internal) {
        console.log(`Network: http://${net.address}:${PORT}`);
      }
    }
  }
});
