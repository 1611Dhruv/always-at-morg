import React, { useState } from 'react';
import './App.css';

function App() {
  const [copied, setCopied] = useState(false);
  const [activeSection, setActiveSection] = useState('home');

  const installCommand = "curl -fsSL https://web.always-at-mor.big/install.sh | bash";

  const copyToClipboard = () => {
    navigator.clipboard.writeText(installCommand);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  return (
    <div className="App">
      {/* Navigation */}
      <nav className="nav">
        <div className="nav-logo">Always at Morg</div>
        <div className="nav-links">
          <button onClick={() => setActiveSection('home')} className={activeSection === 'home' ? 'active' : ''}>
            Home
          </button>
          <button onClick={() => setActiveSection('features')} className={activeSection === 'features' ? 'active' : ''}>
            Features
          </button>
          <button onClick={() => setActiveSection('install')} className={activeSection === 'install' ? 'active' : ''}>
            Install
          </button>
        </div>
      </nav>

      {/* Hero Section */}
      {activeSection === 'home' && (
        <section className="hero">
          <div className="hero-content">
            <div className="badge">UW Madison</div>
            <h1 className="hero-title">
              Always at <span className="title-highlight">Morg</span>
            </h1>
            <p className="hero-subtitle">
              A multiplayer terminal game connecting UW Madison students
              <br />
              Hang out in virtual Morgridge Hall from anywhere
            </p>

            <div className="cta-buttons">
              <button className="btn-primary" onClick={() => setActiveSection('install')}>
                Get Started
              </button>
              <button className="btn-secondary" onClick={() => window.open('https://github.com/1611Dhruv/always-at-morg', '_blank')}>
                View on GitHub
              </button>
            </div>

            <div className="terminal-preview">
              <div className="terminal-header">
                <span className="dot red"></span>
                <span className="dot yellow"></span>
                <span className="dot green"></span>
                <span className="terminal-title">always-at-morg</span>
              </div>
              <div className="terminal-body">
                <div className="terminal-line typing">
                  <span className="prompt">$</span> morg
                </div>
                <div className="terminal-line typing" style={{animationDelay: '0.5s'}}>
                  <span className="text-muted">Connecting to Morgridge Hall...</span>
                </div>
                <div className="terminal-line typing" style={{animationDelay: '1s'}}>
                  <span className="text-success">Connected!</span>
                </div>
                <div className="terminal-line typing" style={{animationDelay: '1.5s'}}>
                  <span className="text-highlight">Welcome to Always at Morg!</span>
                </div>
              </div>
            </div>
          </div>
        </section>
      )}

      {/* Features Section */}
      {activeSection === 'features' && (
        <section className="features">
          <h2 className="section-title">Why Always at Morg?</h2>

          <div className="features-grid">
            <div className="feature-card">
              <i className="fas fa-building feature-icon"></i>
              <h3>Explore Morgridge</h3>
              <p>Navigate through a virtual recreation of Morgridge Hall with your custom 3x3 avatar</p>
            </div>

            <div className="feature-card">
              <i className="fas fa-comments feature-icon"></i>
              <h3>Stay Connected</h3>
              <p>Global chat, room chat, and private messages to connect with fellow Badgers</p>
            </div>

            <div className="feature-card">
              <i className="fas fa-gamepad feature-icon"></i>
              <h3>Play Together</h3>
              <p>Treasure hunts and interactive games right in your terminal</p>
            </div>

            <div className="feature-card">
              <i className="fas fa-users feature-icon"></i>
              <h3>Real-time Multiplayer</h3>
              <p>See other students move around in real-time as you explore</p>
            </div>

            <div className="feature-card">
              <i className="fas fa-bolt feature-icon"></i>
              <h3>Lightning Fast</h3>
              <p>Built with Go and runs entirely in your terminal - lightweight and efficient</p>
            </div>

            <div className="feature-card">
              <i className="fas fa-palette feature-icon"></i>
              <h3>Customize</h3>
              <p>Create your own unique avatar and make it yours</p>
            </div>
          </div>

          <div className="controls-section">
            <h3 className="subsection-title">Controls</h3>
            <div className="controls-grid">
              <div className="control-item">
                <kbd>W A S D</kbd>
                <span>Move around</span>
              </div>
              <div className="control-item">
                <kbd>Enter</kbd>
                <span>Start chatting</span>
              </div>
              <div className="control-item">
                <kbd>G</kbd>
                <span>Global chat</span>
              </div>
              <div className="control-item">
                <kbd>O</kbd>
                <span>Room chat</span>
              </div>
              <div className="control-item">
                <kbd>P</kbd>
                <span>Private chat</span>
              </div>
              <div className="control-item">
                <kbd>Esc</kbd>
                <span>Exit chat</span>
              </div>
            </div>
          </div>
        </section>
      )}

      {/* Install Section */}
      {activeSection === 'install' && (
        <section className="install">
          <h2 className="section-title">Get Started</h2>

          <div className="install-steps">
            <div className="step">
              <div className="step-number">1</div>
              <div className="step-content">
                <h3>Install the game</h3>
                <p>Run this command in your terminal:</p>
                <div className="command-box" onClick={copyToClipboard}>
                  <code>{installCommand}</code>
                  <button className={`copy-btn ${copied ? 'copied' : ''}`}>
                    {copied ? 'Copied!' : 'Copy'}
                  </button>
                </div>
                <p className="text-muted">
                  Supports macOS, Linux, and Windows (WSL)
                </p>
              </div>
            </div>

            <div className="step">
              <div className="step-number">2</div>
              <div className="step-content">
                <h3>Run the game</h3>
                <div className="command-box">
                  <code>morg</code>
                </div>
                <p className="text-muted">
                  The game will automatically connect to the server
                </p>
              </div>
            </div>

            <div className="step">
              <div className="step-number">3</div>
              <div className="step-content">
                <h3>Create your avatar</h3>
                <p>Choose your colors and start exploring Morgridge Hall!</p>
              </div>
            </div>
          </div>

          <div className="platform-support">
            <h3 className="subsection-title">Platform Support</h3>
            <div className="platforms">
              <div className="platform">
                <i className="fab fa-apple platform-icon"></i>
                <span>macOS</span>
                <i className="fas fa-check text-success"></i>
              </div>
              <div className="platform">
                <i className="fab fa-linux platform-icon"></i>
                <span>Linux</span>
                <i className="fas fa-check text-success"></i>
              </div>
              <div className="platform">
                <i className="fab fa-windows platform-icon"></i>
                <span>Windows</span>
                <i className="fas fa-check text-success"></i>
              </div>
            </div>
          </div>
        </section>
      )}

      {/* Footer */}
      <footer className="footer">
        <p>Made with <i className="fas fa-heart"></i> for <span className="uw-badge">UW Madison</span> students</p>
        <p className="footer-links">
          <a href="https://github.com/1611Dhruv/always-at-morg" target="_blank" rel="noopener noreferrer">
            GitHub
          </a>
          {' • '}
          <a href="https://github.com/1611Dhruv/always-at-morg/issues" target="_blank" rel="noopener noreferrer">
            Report Issues
          </a>
        </p>
        <p className="badger">On, Wisconsin!</p>
      </footer>
    </div>
  );
}

export default App;
