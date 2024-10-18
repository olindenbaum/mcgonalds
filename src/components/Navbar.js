import React from 'react';
import { Link, useLocation } from 'react-router-dom';
import { FaList, FaPlus } from 'react-icons/fa';
import './Navbar.css';

function Navbar() {
  const location = useLocation();

  return (
    <nav className="navbar">
      <h2 className="navbar-brand">McGonald's</h2>
      <ul className="navbar-links">
        <li className={location.pathname === '/' ? 'active' : ''}>
          <Link to="/"><FaList className="icon" /> Server List</Link>
        </li>
        <li className={location.pathname === '/create' ? 'active' : ''}>
          <Link to="/create"><FaPlus className="icon" /> Create Server</Link>
        </li>
      </ul>
    </nav>
  );
}

export default Navbar;
