import React, { useState, useEffect } from 'react';
import { useParams } from 'react-router-dom';
import axios from 'axios';

function ServerDetails() {
  const { id } = useParams();
  const [server, setServer] = useState(null);
  const [status, setStatus] = useState('');

  useEffect(() => {
    fetchServerDetails();
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
      await axios.post(`/api/v1/servers/${id}/start`);
      setStatus('Server started');
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

  if (!server) {
    return <div>Loading...</div>;
  }

  return (
    <div>
      <h1>{server.name}</h1>
      <p>Path: {server.path}</p>
      <p>JAR File: {server.jar_file.name} - {server.jar_file.version}</p>
      {server.mod_pack && (
        <p>Mod Pack: {server.mod_pack.name} - {server.mod_pack.version}</p>
      )}
      <p>Execution Command: {server.server_config.executable_command}</p>
      <button onClick={handleStart}>Start Server</button>
      <button onClick={handleStop}>Stop Server</button>
      <p>Status: {status}</p>
    </div>
  );
}

export default ServerDetails;