import React from 'react';
import { Link } from 'react-router-dom';

function Navbar() {
  return (
    <nav>
      <ul>
        <li><Link to="/">Server List</Link></li>
        <li><Link to="/create">Create Server</Link></li>
      </ul>
    </nav>
  );
}

export default Navbar;