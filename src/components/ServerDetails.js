import React, { useState, useEffect, useContext, useRef } from 'react';
import { useParams } from 'react-router-dom';
import axios from '../utils/axiosInstance'; // Use the Axios instance with auth
import './ServerDetails.css';
import { AuthContext } from '../contexts/AuthContext';

function ServerDetails() {
  const { id } = useParams();
  const { auth } = useContext(AuthContext);
  const [server, setServer] = useState(null);
  const [status, setStatus] = useState('');
  const [ram, setRam] = useState('2G');
  const [port, setPort] = useState('25565');
  const [command, setCommand] = useState('');
  const [output, setOutput] = useState('');
  const ws = useRef(null);

  useEffect(() => {
    fetchServerDetails();
    // Establish WebSocket connection
    if (auth.token) {
      connectWebSocket();
    }
    // Cleanup on unmount
    return () => {
      if (ws.current) {
        ws.current.close();
      }
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [id, auth]);

  const fetchServerDetails = async () => {
    try {
      const response = await axios.get(`/servers/${id}`);
      setServer(response.data);
      setStatus(response.data.status);
    } catch (error) {
      console.error('Error fetching server details:', error);
    }
  };

  const connectWebSocket = () => {
    const token = localStorage.getItem('token');
    ws.current = new WebSocket(`ws://localhost:8080/api/v1/servers/${id}/output/ws?token=${token}`);

    ws.current.onopen = () => {
      console.log('WebSocket connection established');
    };

    ws.current.onmessage = (event) => {
      setOutput((prev) => prev + event.data + '\n');
    };

    ws.current.onerror = (error) => {
      console.error('WebSocket error:', error);
    };

    ws.current.onclose = () => {
      console.log('WebSocket connection closed');
    };
  };

  const handleStart = async () => {
    try {
      await axios.post(`/servers/${id}/start`);
      setStatus('Running');
    } catch (error) {
      console.error('Error starting server:', error);
      setStatus('Failed to start server');
    }
  };

  const handleStop = async () => {
    try {
      await axios.post(`/servers/${id}/stop`);
      setStatus('Stopped');
    } catch (error) {
      console.error('Error stopping server:', error);
      setStatus('Failed to stop server');
    }
  };

  const handleCommand = async () => {
    try {
      await axios.post(`/servers/${id}/command`, { command });
      setCommand('');
    } catch (error) {
      console.error('Error sending command:', error);
    }
  };

  return (
    <div className="server-details-container">
      {server && (
        <>
          <h1>{server.name}</h1>
          <p>Status: {status}</p>
          <div className="server-info">
            <p>Server ID: {server.id}</p>
            <p>Path: {server.path}</p>
            <p>Running: {server.is_running ? 'Yes' : 'No'}</p>
          </div>

          <div className="server-controls">
            <button onClick={handleStart} disabled={status === 'Running'}>Start Server</button>
            <button onClick={handleStop} disabled={status === 'Stopped'}>Stop Server</button>
          </div>

          <div className="command-section">
            <h2>Send Command</h2>
            <input
              type="text"
              value={command}
              onChange={(e) => setCommand(e.target.value)}
              placeholder="Enter command"
            />
            <button onClick={handleCommand}>Send</button>
          </div>

          <div className="server-output">
            <h2>Server Output:</h2>
            <pre>
              {output}
            </pre>
          </div>
        </>
      )}
    </div>
  );
}

export default ServerDetails;
