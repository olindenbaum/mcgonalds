import React, { useState, useEffect } from 'react';
import { useParams } from 'react-router-dom';
import axios from 'axios';

function ServerDetails() {
  const { id } = useParams();
  const [server, setServer] = useState(null);
  const [status, setStatus] = useState('');
  const [ram, setRam] = useState('2G'); // Default RAM
  const [port, setPort] = useState('25565'); // Default Port
  const [output, setOutput] = useState('');

  useEffect(() => {
    fetchServerDetails();
    // Optionally, establish a WebSocket connection here for real-time output
  }, [id]);

  const fetchServerDetails = async () => {
    try {
      const response = await axios.get(`/api/v1/servers/${id}`);
      setServer(response.data);
    } catch (error) {
      console.error('Error fetching server details:', error);
    }
  };

  const handleStart = async () => {
    try {
      const payload = {
        ram,
        port,
      };
      await axios.post(`/api/v1/servers/${id}/start`, payload);
      setStatus('Server started');
      // Optionally, start listening to the output stream
      fetchServerOutput();
    } catch (error) {
      console.error('Error starting server:', error);
      setStatus('Failed to start server');
    }
  };

  const handleStop = async () => {
    try {
      await axios.post(`/api/v1/servers/${id}/stop`);
      setStatus('Server stopped');
    } catch (error) {
      console.error('Error stopping server:', error);
      setStatus('Failed to stop server');
    }
  };

  const fetchServerOutput = async () => {
    try {
      const response = await axios.get(`/api/v1/servers/${id}/output`);
      setOutput(response.data.output);
    } catch (error) {
      console.error('Error fetching server output:', error);
    }
  };

  return (
    <div>
      {server && (
        <>
          <h1>{server.name}</h1>
          <p>Server ID: {server.server_id}</p>
          <p>Path: {server.path}</p>
          <p>Running: {server.is_running ? 'Yes' : 'No'}</p>
          {server.config && (
            <>
              <h2>Server Configuration</h2>
              <p>Config ID: {server.config.id}</p>
              <p>Created At: {new Date(server.config.created_at).toLocaleString()}</p>
              <p>Updated At: {new Date(server.config.updated_at).toLocaleString()}</p>
              <p>Execution Command: {server.config.executable_command}</p>
              {server.config.jar_file && server.config.jar_file.id !== 0 && (
                <>
                  <h3>JAR File</h3>
                  <p>Name: {server.config.jar_file.name}</p>
                  <p>Version: {server.config.jar_file.version}</p>
                  <p>Path: {server.config.jar_file.path}</p>
                  <p>Is Common: {server.config.jar_file.is_common ? 'Yes' : 'No'}</p>
                </>
              )}
              {server.config.mod_pack_id && (
                <p>Mod Pack ID: {server.config.mod_pack_id}</p>
              )}
            </>
          )}
        </>
      )}

      <div>
        <label htmlFor="ram">RAM Allocation:</label>
        <input
          type="text"
          id="ram"
          value={ram}
          onChange={(e) => setRam(e.target.value)}
          placeholder="e.g., 2G, 4G"
        />
      </div>

      <div>
        <label htmlFor="port">Port:</label>
        <input
          type="number"
          id="port"
          value={port}
          onChange={(e) => setPort(e.target.value)}
          placeholder="e.g., 25565"
        />
      </div>

      <button onClick={handleStart}>Start Server</button>
      <button onClick={handleStop}>Stop Server</button>
      <p>Status: {status}</p>

      <h2>Server Output:</h2>
      <pre style={{ backgroundColor: '#f0f0f0', padding: '10px' }}>
        {output}
      </pre>
    </div>
  );
}

export default ServerDetails;
