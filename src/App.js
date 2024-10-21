import React from 'react';
import { BrowserRouter as Router, Route, Routes, Navigate } from 'react-router-dom';
import Navbar from './components/Navbar';
import ServerList from './components/ServerList';
import CreateServer from './components/CreateServer';
import ServerDetails from './components/ServerDetails';
import Login from './components/Login';
import Signup from './components/Signup';
import { AuthProvider, AuthContext } from './contexts/AuthContext';

function App() {
  return (
    <AuthProvider>
      <Router>
        <div className="App">
          <Navbar />
          <Routes>
            <Route path="/" element={<ProtectedRoute><ServerList /></ProtectedRoute>} />
            <Route path="/create" element={<ProtectedRoute><CreateServer /></ProtectedRoute>} />
            <Route path="/server/:id" element={<ProtectedRoute><ServerDetails /></ProtectedRoute>} />
            <Route path="/login" element={<Login />} />
            <Route path="/signup" element={<Signup />} />
          </Routes>
        </div>
      </Router>
    </AuthProvider>
  );
}

const ProtectedRoute = ({ children }) => {
  const { auth } = React.useContext(AuthContext);
  return auth.token ? children : <Navigate to="/login" />;
};

export default App;
