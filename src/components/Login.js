import React, { useState, useContext } from 'react';
import axios from 'axios';
import { AuthContext } from '../contexts/AuthContext';
import { useNavigate } from 'react-router-dom';
import './Auth.css';

function Login() {
  const { login } = useContext(AuthContext);
  const navigate = useNavigate();
  
  const [form, setForm] = useState({ username: '', password: '' });
  const [error, setError] = useState('');
  
  const handleChange = (e) => {
    setForm({ ...form, [e.target.name]: e.target.value });
  };
  
  const handleSubmit = async (e) => {
    e.preventDefault();
    setError('');
    
    try {
      const response = await axios.post('/api/v1/login', form);
      login(response.data.token);
      navigate('/');
    } catch (err) {
      setError(err.response?.data?.message || 'Login failed');
    }
  };
  
  return (
    <div className="auth-container">
      <h2>Login</h2>
      {error && <div className="error-message">{error}</div>}
      <form onSubmit={handleSubmit} className="auth-form">
        <div className="form-group">
          <label>Username:</label>
          <input type="text" name="username" value={form.username} onChange={handleChange} required />
        </div>
        
        <div className="form-group">
          <label>Password:</label>
          <input type="password" name="password" value={form.password} onChange={handleChange} required />
        </div>
        
        <button type="submit">Login</button>
      </form>
    </div>
  );
}

export default Login;

