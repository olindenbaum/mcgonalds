import React, { useState, useEffect } from 'react';
import { Link } from 'react-router-dom';
import axios from 'axios';

function ServerList() {
  const [servers, setServers] = useState([]);

  useEffect(() => {
    fetchServers();
    // Optionally, set up polling or WebSocket connections for real-time status updates
  }, []);

  const fetchServers = async () => {
    try {
      const response = await axios.get('/api/v1/servers');
      setServers(response.data);
    } catch (error) {
      console.error('Error fetching servers:', error);
    }
  };

  return (
    <div>
      <h1>Minecraft Servers</h1>
      <ul>
        {servers.map((server) => (
          <li key={server.id}>
            <Link to={`/server/${server.id}`}>{server.name}</Link> - Status: {server.status}
          </li>
        ))}
      </ul>
    </div>
  );
}

export default ServerList;
