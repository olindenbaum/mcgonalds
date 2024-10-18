import React, { useState, useEffect } from 'react';
import { Link } from 'react-router-dom';
import axios from 'axios';
import './ServerList.css';

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
    <div className="server-list-container">
      <h1>Minecraft Servers</h1>
      <div className="server-cards">
        {servers.map((server) => (
          <div key={server.id} className="server-card">
            <h2>{server.name}</h2>
            <p>Status: <span className={server.status === 'Running' ? 'status-running' : 'status-stopped'}>{server.status}</span></p>
            <Link to={`/server/${server.id}`} className="details-link">View Details</Link>
          </div>
        ))}
      </div>
    </div>
  );
}

export default ServerList;
