# Contributing to KumoMTA UI

Thank you for your interest in contributing to KumoMTA UI! This document provides guidelines and instructions for contributing.

## Getting Started

### Prerequisites

- Go 1.22 or later
- Node.js 18+ and npm
- SQLite3
- Git

### Development Setup

1. **Clone the repository**
   ```bash
   git clone https://github.com/pulak-ranjan/kumomta-ui.git
   cd kumomta-ui
   ```

2. **Install Go dependencies**
   ```bash
   go mod tidy
   ```

3. **Install frontend dependencies**
   ```bash
   cd web
   npm install
   cd ..
   ```

4. **Run the backend (development)**
   ```bash
   export DB_DIR=./data
   go run ./cmd/server
   ```

5. **Run the frontend (development)**
   ```bash
   cd web
   npm run dev
   ```

6. **Access the panel**
   - Frontend: http://localhost:5173
   - API: http://localhost:9000/api

## Project Structure

```
kumomta-ui/
├── cmd/server/          # Application entry point
├── internal/
│   ├── api/             # HTTP handlers and routing
│   ├── core/            # Business logic (config generation, DKIM, etc.)
│   ├── models/          # Database models
│   └── store/           # Database operations
├── scripts/             # Installation and utility scripts
├── web/                 # React frontend
│   ├── src/
│   │   ├── pages/       # Page components
│   │   ├── api.js       # API client
│   │   └── AuthContext.jsx
│   └── ...
└── go.mod
```

## Code Style

### Go

- Follow standard Go formatting (`go fmt`)
- Use meaningful variable and function names
- Add comments for exported functions
- Handle errors explicitly
- Use the existing patterns in the codebase

### React/JavaScript

- Use functional components with hooks
- Follow the existing Tailwind CSS patterns
- Keep components focused and reasonably sized
- Use the `api.js` module for API calls

## Making Changes

### Branch Naming

- `feature/description` - New features
- `fix/description` - Bug fixes
- `docs/description` - Documentation updates
- `refactor/description` - Code refactoring

### Commit Messages

Use clear, descriptive commit messages:

```
feat: add bounce account bulk import
fix: token expiry check timezone issue
docs: update installation instructions
refactor: extract DNS helpers to separate module
```

### Pull Request Process

1. Create a feature branch from `main`
2. Make your changes
3. Test thoroughly
4. Update documentation if needed
5. Submit a pull request with a clear description

## Testing

### Manual Testing Checklist

Before submitting a PR, please verify:

- [ ] Backend compiles without errors (`go build ./cmd/server`)
- [ ] Frontend builds without errors (`cd web && npm run build`)
- [ ] New features work as expected
- [ ] Existing functionality is not broken
- [ ] API responses are correct
- [ ] UI displays properly

### Running the Full Stack

```bash
# Terminal 1: Backend
export DB_DIR=./data
go run ./cmd/server

# Terminal 2: Frontend
cd web
npm run dev
```

## Reporting Issues

When reporting issues, please include:

- Description of the problem
- Steps to reproduce
- Expected vs actual behavior
- Environment details (OS, Go version, Node version)
- Relevant logs or error messages

## Security Issues

If you discover a security vulnerability, please **do not** open a public issue. Instead, contact the maintainer directly.

## Questions?

Feel free to open an issue for questions or discussions about the project.

---

Thank you for contributing! 🎉
