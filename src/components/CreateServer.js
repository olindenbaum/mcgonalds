import React, { useState, useEffect, useContext } from 'react';
import axios from '../utils/axiosInstance';
import './CreateServer.css';
import { AuthContext } from '../contexts/AuthContext';

function CreateServer() {
  const { auth } = useContext(AuthContext);
  const [name, setName] = useState('');
  const [jarFile, setJarFile] = useState('');
  const [modPack, setModPack] = useState('');
  const [command, setCommand] = useState('');
  const [commonJars, setCommonJars] = useState([]);
  const [commonModPacks, setCommonModPacks] = useState([]);
  const [selectedJarFile, setSelectedJarFile] = useState(null);
  const [selectedModPack, setSelectedModPack] = useState(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');
  const [success, setSuccess] = useState('');

  useEffect(() => {
    fetchCommonAssets();
  }, []);

  const fetchCommonAssets = async () => {
    try {
      const jarsResponse = await axios.get('/jar-files?common=true');
      setCommonJars(jarsResponse.data);

      const modPacksResponse = await axios.get('/mod-packs?common=true');
      setCommonModPacks(modPacksResponse.data);
    } catch (error) {
      console.error('Error fetching common assets:', error);
      setError('Failed to load common assets.');
    }
  };

  const handleSubmit = async (e) => {
    e.preventDefault();
    setLoading(true);
    setError('');
    setSuccess('');

    try {
      const formData = new FormData();
      formData.append('name', name);
      formData.append('executable_command', command);
      if (selectedJarFile) {
        formData.append('jar_file', selectedJarFile);
      } else if (jarFile) {
        formData.append('jar_file_id', jarFile);
      }

      if (selectedModPack) {
        formData.append('mod_pack', selectedModPack);
      } else if (modPack) {
        formData.append('mod_pack_id', modPack);
      }

      const config = {
        headers: {
          'Content-Type': 'multipart/form-data',
        },
      };

      const response = await axios.post('/servers', formData, config);
      console.log('Server created:', response.data);
      setSuccess('Server created successfully!');
      // Reset form
      setName('');
      setJarFile('');
      setModPack('');
      setCommand('');
      setSelectedJarFile(null);
      setSelectedModPack(null);
    } catch (error) {
      console.error('Error creating server:', error);
      setError(error.response?.data?.message || 'Failed to create server.');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="create-server-container">
      <h1>Create Minecraft Server</h1>
      {error && <div className="error-message">{error}</div>}
      {success && <div className="success-message">{success}</div>}
      <form onSubmit={handleSubmit} className="create-server-form">
        <div className="form-group">
          <label htmlFor="name">Server Name:</label>
          <input
            type="text"
            id="name"
            value={name}
            onChange={(e) => setName(e.target.value)}
            required
            placeholder="Enter server name"
          />
        </div>

        <div className="form-group">
          <label htmlFor="jarFile">JAR File:</label>
          <select
            id="jarFile"
            value={jarFile}
            onChange={(e) => setJarFile(e.target.value)}
            disabled={selectedJarFile !== null}
            className="select-input"
          >
            <option value="">Select a common JAR file</option>
            {commonJars.map((jar) => (
              <option key={jar.id} value={jar.id}>
                {jar.name} - {jar.version}
              </option>
            ))}
          </select>
          <div className="file-input-container">
            <input
              type="file"
              accept=".jar"
              onChange={(e) => setSelectedJarFile(e.target.files[0])}
              className="file-input"
            />
            {selectedJarFile && <span>{selectedJarFile.name}</span>}
          </div>
        </div>

        <div className="form-group">
          <label htmlFor="modPack">Mod Pack:</label>
          <select
            id="modPack"
            value={modPack}
            onChange={(e) => setModPack(e.target.value)}
            disabled={selectedModPack !== null}
            className="select-input"
          >
            <option value="">Select a common Mod Pack (optional)</option>
            {commonModPacks.map((pack) => (
              <option key={pack.id} value={pack.id}>
                {pack.name} - {pack.version}
              </option>
            ))}
          </select>
          <div className="file-input-container">
            <input
              type="file"
              accept=".zip,.jar"
              onChange={(e) => setSelectedModPack(e.target.files[0])}
              className="file-input"
            />
            {selectedModPack && <span>{selectedModPack.name}</span>}
          </div>
        </div>

        <div className="form-group">
          <label htmlFor="command">Executable Command:</label>
          <input
            type="text"
            id="command"
            value={command}
            onChange={(e) => setCommand(e.target.value)}
            required
            placeholder="e.g., java -jar file.jar -Xmx2G -Xms2G"
          />
        </div>

        <button type="submit" disabled={loading} className="submit-button">
          {loading ? 'Creating...' : 'Create Server'}
        </button>
      </form>
    </div>
  );
}

export default CreateServer;
