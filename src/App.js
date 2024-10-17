import React from 'react';
import { BrowserRouter as Router, Route, Routes } from 'react-router-dom';
import Navbar from './components/Navbar';
import ServerList from './components/ServerList';
import CreateServer from './components/CreateServer';
import ServerDetails from './components/ServerDetails';

function App() {
  return (
    <Router>
      <div className="App">
        <Navbar />
        <Routes>
          <Route path="/" element={<ServerList />} />
          <Route path="/create" element={<CreateServer />} />
          <Route path="/server/:id" element={<ServerDetails />} />
        </Routes>
      </div>
    </Router>
  );
}

export default App;