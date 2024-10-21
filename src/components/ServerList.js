import React, { useState, useEffect, useContext } from 'react';
import { Link } from 'react-router-dom';
import axios from '../utils/axiosInstance'; // Updated import
import './ServerList.css';
import { AuthContext } from '../contexts/AuthContext';

function ServerList() {
  const [servers, setServers] = useState([]);
  const { auth } = useContext(AuthContext);

  useEffect(() => {
    fetchServers();
    // Optionally, set up polling or WebSocket connections for real-time status updates
  }, [auth]);

  const fetchServers = async () => {
    try {
      const response = await axios.get('/servers');
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
