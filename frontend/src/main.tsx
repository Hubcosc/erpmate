import React from 'react'
import ReactDOM from 'react-dom/client'
// Base styles load first so component styles can override them.
import './styles/global.css'
import App from './App'

ReactDOM.createRoot(document.getElementById('root')!).render(
  <React.StrictMode>
    <App />
  </React.StrictMode>,
)
