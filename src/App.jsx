import { useState, useEffect } from 'react'
import './App.css'

function App() {
  // State for session keys
  const [sessionKeys, setSessionKeys] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [copiedId, setCopiedId] = useState(null);
  const [showNotification, setShowNotification] = useState(false);
  const [notificationMessage, setNotificationMessage] = useState('');

  // State for new session key input
  const [newSessionKey, setNewSessionKey] = useState('');

  // State for showing/hiding the add form
  const [showAddForm, setShowAddForm] = useState(false);

  // API URL - use the current hostname to work on both localhost and network
  const API_URL = `http://${window.location.hostname}:3001`;

  // Function to truncate long session keys
  const truncateSessionKey = (key) => {
    if (key.length <= 25) return key;
    return `${key.substring(0, 12)}...${key.substring(key.length - 12)}`;
  };

  // Function to copy session key to clipboard
  const copyToClipboard = (id, key) => {
    navigator.clipboard.writeText(key)
      .then(() => {
        setCopiedId(id);
        setTimeout(() => setCopiedId(null), 2000);
      })
      .catch(err => {
        console.error('Failed to copy text: ', err);
      });
  };

  // Function to copy API URL to clipboard
  const copyApiUrl = () => {
    // Get the current hostname (works for both localhost and network IP)
    const hostname = window.location.hostname;
    // Use the current port or default to 3001 if not available
    const port = window.location.port || '3001';
/*    const apiUrl = `http://${hostname}:${port}/v1/chat/completions`;*/
    const apiUrl = `http://${hostname}:5000/v1/chat/completions`;

    navigator.clipboard.writeText(apiUrl)
      .then(() => {
        setNotificationMessage(`API URL copied! Default password: xierlove`);
        setShowNotification(true);
        setTimeout(() => setShowNotification(false), 3000);
      })
      .catch(err => {
        console.error('Failed to copy API URL: ', err);
        alert('Failed to copy API URL. Please try again.');
      });
  };

  // Load session keys from API
  useEffect(() => {
    const fetchSessionKeys = async () => {
      try {
        setLoading(true);
        // Always try to use the API first
        try {
          const response = await fetch(`${API_URL}/api/sessionKeys`);
          if (response.ok) {
            const data = await response.json();
            setSessionKeys(data.sessionKeys || []);
            setError(null);
            return;
          }
        } catch (error) {
          console.warn('API not available, falling back to local JSON file:', error.message);
        }

        // Fallback to local JSON file
        const response = await fetch('/data/sessionKeys.json');
        if (!response.ok) {
          throw new Error(`Failed to fetch session keys: ${response.status}`);
        }
        const data = await response.json();
        setSessionKeys(data.sessionKeys || []);
        setError(null);
      } catch (err) {
        console.error('Error loading session keys:', err);
        setError('Failed to load session keys. Please try again later.');
        setSessionKeys([]);
      } finally {
        setLoading(false);
      }
    };

    fetchSessionKeys();
  }, [API_URL]);

  // Function to save session keys to API
  const saveSessionKeys = async (updatedKeys) => {
    try {
      // Always try to use the API first
      try {
        const response = await fetch(`${API_URL}/api/sessionKeys`, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ sessionKeys: updatedKeys })
        });

        if (response.ok) {
          const data = await response.json();
          setSessionKeys(data.sessionKeys || updatedKeys);
          return;
        }
      } catch (error) {
        console.warn('API not available, using local state only:', error.message);
      }

      // Fallback to just updating the state
      console.log('Saving session keys (local only):', { sessionKeys: updatedKeys });
      setSessionKeys(updatedKeys);
    } catch (err) {
      console.error('Error saving session keys:', err);
      alert('Failed to save changes. Please try again.');
    }
  };

  // Function to handle redirection to fuclaude.com with the selected session key
  const handleUseKey = (sessionKey) => {
    window.open(`https://demo.fuclaude.com/login_token?session_key=${sessionKey}`, '_blank');
  };

  // Function to delete a session key
  const handleDeleteKey = (id) => {
    const updatedKeys = sessionKeys.filter(item => item.id !== id);
    saveSessionKeys(updatedKeys);
  };

  // Function to add a new session key
  const handleAddKey = (e) => {
    e.preventDefault();
    if (newSessionKey.trim()) {
      const newId = sessionKeys.length > 0 ? Math.max(...sessionKeys.map(item => item.id)) + 1 : 1;
      const updatedKeys = [...sessionKeys, { id: newId, key: newSessionKey.trim() }];
      saveSessionKeys(updatedKeys);
      setNewSessionKey('');
      setShowAddForm(false);
    }
  };

  return (
    <div className="app-container">
      <div className="window">
        <div className="title-bar">
          <div className="title-bar-controls">
            <button className="control close"></button>
            <button className="control minimize"></button>
            <button className="control maximize"></button>
          </div>
          <div className="title-bar-text">Session Key Management</div>
        </div>

        <div className="window-content">
          <div className="toolbar">
            <button
              className="toolbar-button copy-api-button"
              onClick={copyApiUrl}
              aria-label="Copy API URL"
              disabled={loading}
            >
              Copy API URL
            </button>
            <div className="toolbar-spacer"></div>
            <button
              className="toolbar-button add-button"
              onClick={() => setShowAddForm(!showAddForm)}
              aria-label="Add Session Key"
              disabled={loading}
            >
              {showAddForm ? 'Cancel' : '+ Add Session Key'}
            </button>
          </div>

          {showNotification && (
            <div className="notification">
              {notificationMessage}
            </div>
          )}

          {showAddForm && (
            <div className="add-form-container">
              <form onSubmit={handleAddKey} className="add-form">
                <input
                  type="text"
                  value={newSessionKey}
                  onChange={(e) => setNewSessionKey(e.target.value)}
                  placeholder="Enter new session key"
                  className="session-key-input"
                />
                <button type="submit" className="add-submit-button">Add</button>
              </form>
            </div>
          )}

          <div className="session-keys-list">
            {loading ? (
              <div className="loading-state">
                <div className="spinner"></div>
                <p>Loading session keys...</p>
              </div>
            ) : error ? (
              <div className="error-state">
                <p className="error-message">{error}</p>
                <button
                  className="retry-button"
                  onClick={() => window.location.reload()}
                >
                  Retry
                </button>
              </div>
            ) : sessionKeys.length === 0 ? (
              <div className="empty-state">
                <p>No session keys available. Add one to get started.</p>
              </div>
            ) : (
              <div className="session-keys-table-container">
                <table className="session-keys-table">
                  <thead>
                    <tr>
                      <th className="id-column">ID</th>
                      <th>Session Key</th>
                      <th className="actions-column">Actions</th>
                    </tr>
                  </thead>
                  <tbody>
                    {sessionKeys.map((item) => (
                      <tr key={item.id}>
                        <td className="id-column">{item.id}</td>
                        <td className="session-key-cell">
                          <div className="session-key-wrapper">
                            <span className="session-key">{truncateSessionKey(item.key)}</span>
                            <button
                              className={`copy-button ${copiedId === item.id ? 'copied' : ''}`}
                              onClick={() => copyToClipboard(item.id, item.key)}
                              title="Copy to clipboard"
                            >
                              {copiedId === item.id ? 'Copied!' : 'Copy'}
                            </button>
                          </div>
                        </td>
                        <td className="action-buttons">
                          <button
                            className="action-button use-button"
                            onClick={() => handleUseKey(item.key)}
                            title="Use this session key"
                          >
                            Use
                          </button>
                          <button
                            className="action-button delete-button"
                            onClick={() => handleDeleteKey(item.id)}
                            title="Delete this session key"
                          >
                            Delete
                          </button>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </div>
        </div>
      </div>
    </div>
  )
}

export default App
