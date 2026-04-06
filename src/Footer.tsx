export function Footer() {
  return (
    <footer className="footer glass-panel">
      <h3 className="footer-title">Author</h3>
      <div className="author-container">
        <img src="https://github.com/hakanolgun.png" alt="Hakan Olgun" className="author-avatar" />
        <div className="author-details">
          <span className="author-name">Hakan Olgun</span>
          <div className="social-links">
            <a href="https://github.com/hakanolgun" target="_blank" rel="noreferrer" title="GitHub">
              <svg
                width="18"
                height="18"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                strokeWidth="2"
                strokeLinecap="round"
                strokeLinejoin="round">
                <path d="M9 19c-5 1.5-5-2.5-7-3m14 6v-3.87a3.37 3.37 0 0 0-.94-2.61c3.14-.35 6.44-1.54 6.44-7A5.44 5.44 0 0 0 20 4.77 5.07 5.07 0 0 0 19.91 1S18.73.65 16 2.48a13.38 13.38 0 0 0-7 0C6.27.65 5.09 1 5.09 1A5.07 5.07 0 0 0 5 4.77a5.44 5.44 0 0 0-1.5 3.78c0 5.42 3.3 6.61 6.44 7A3.37 3.37 0 0 0 9 18.13V22"></path>
              </svg>
            </a>
            <a
              href="https://linkedin.com/in/hknlgn"
              target="_blank"
              rel="noreferrer"
              title="LinkedIn">
              <svg
                width="18"
                height="18"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                strokeWidth="2"
                strokeLinecap="round"
                strokeLinejoin="round">
                <path d="M16 8a6 6 0 0 1 6 6v7h-4v-7a2 2 0 0 0-2-2 2 2 0 0 0-2 2v7h-4v-7a6 6 0 0 1 6-6z"></path>
                <rect x="2" y="9" width="4" height="12"></rect>
                <circle cx="4" cy="4" r="2"></circle>
              </svg>
            </a>
            <a href="https://x.com/kpt_hkn" target="_blank" rel="noreferrer" title="X (Twitter)">
              <svg
                width="18"
                height="18"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                strokeWidth="2"
                strokeLinecap="round"
                strokeLinejoin="round">
                <path d="M4 4l11.733 16h4.267l-11.733-16z"></path>
                <path d="M4 20l6.768-6.768m2.46-2.46l6.772-6.772"></path>
              </svg>
            </a>
            <a href="https://social.vivaldi.net/@mathrandom" target="_blank" rel="noreferrer" title="Mastodon" className="mastodon-icon">
              <svg
                width="18"
                height="18"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                strokeWidth="2"
                strokeLinecap="round"
                strokeLinejoin="round">
                <path d="M21.58 13.913c-.245 3.074-2.81 5.9-6.31 6.551-2.146.398-4.4.453-6.195.148-1.52-.259-2.85-.92-2.85-.92s-.027-1.393-.049-2.924c1.238.384 2.658.544 4.148.514 2.805-.054 4.978-.507 5.679-2.079.034-.078.066-.159.096-.24.593-1.68-.041-3.666-.041-3.666-4.636 1.139-8.497.106-9.67-2.613-.245-.583-.34-1.282-.365-2.008-.045-1.196-.062-2.52.016-3.834.135-2.261 1.258-4.502 3.619-5.496C11.597 2.456 14.542 2.5 14.542 2.5h.063s2.946-.044 4.256.848c2.361.993 3.484 3.235 3.619 5.496.113 1.956.12 4.17.065 6.069h-.963z" />
                <path d="M16.51 10.74v.85h-1.57v-.85c0-1.07-.63-1.6-1.89-1.6-1.37 0-2.06.67-2.06 2.01v4.4h-1.54v-4.4c0-1.34-.69-2.01-2.06-2.01-1.26 0-1.89.53-1.89 1.6v.85H3.93v-1.63c0-1.57.85-2.6 2.53-3.1.55-.16 1.16-.24 1.83-.24 1.25 0 2.21.36 2.89 1.07.67-.71 1.63-1.07 2.89-1.07.67 0 1.28.08 1.83.24 1.68.5 2.53 1.53 2.53 3.1v1.63z" />
              </svg>
            </a>
          </div>
        </div>
      </div>
    </footer>
  );
}
