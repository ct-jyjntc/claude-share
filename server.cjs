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
console.log('Server starting...');
console.log('Current directory:', __dirname);
console.log('SessionKeys.json path:', dataFilePath);

// Check if the file exists
try {
  const fileExists = fs.existsSync(dataFilePath);
  console.log('SessionKeys.json exists:', fileExists);

  // Check if the directory exists
  const dirPath = path.dirname(dataFilePath);
  const dirExists = fs.existsSync(dirPath);
  console.log('Directory exists:', dirExists);

  // Create directory if it doesn't exist
  if (!dirExists) {
    console.log('Creating directory:', dirPath);
    fs.mkdirpSync(dirPath);
    console.log('Directory created');
  }

  // If file doesn't exist, create an empty file
  if (!fileExists) {
    console.log('Creating empty sessionKeys.json file');
    fs.writeJsonSync(dataFilePath, { sessionKeys: [] }, { spaces: 2 });
    console.log('Empty sessionKeys.json file created');
  }
} catch (error) {
  console.error('Error checking file:', error);
}

// API Routes
// Get all session keys
app.get('/api/sessionKeys', async (req, res) => {
  try {
    console.log('Received request to get session keys');
    console.log('Reading from file:', dataFilePath);

    const data = await fs.readJson(dataFilePath);
    console.log('Successfully read sessionKeys.json:', data);

    res.json(data);
  } catch (error) {
    console.error('Error reading session keys:', error);
    console.error('Error details:', error.stack);
    res.status(500).json({ error: 'Failed to read session keys' });
  }
});

// Update session keys
app.post('/api/sessionKeys', async (req, res) => {
  try {
    console.log('Received request to update session keys:', req.body);
    const { sessionKeys } = req.body;

    if (!Array.isArray(sessionKeys)) {
      console.error('Invalid data format received:', req.body);
      return res.status(400).json({ error: 'Invalid data format' });
    }

    console.log('Writing to file:', dataFilePath);
    console.log('Data to write:', { sessionKeys });

    await fs.writeJson(dataFilePath, { sessionKeys }, { spaces: 2 });
    console.log('Successfully wrote to sessionKeys.json');

    res.json({ success: true, sessionKeys });
  } catch (error) {
    console.error('Error saving session keys:', error);
    console.error('Error details:', error.stack);
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
