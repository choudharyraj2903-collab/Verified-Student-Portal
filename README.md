# Verified Student Portal

A security-first platform for students to request and manage verification of their Points of Responsibility (PORs) and other academic records.

## Project Structure
- `backend/`: Go-based API server with PostgreSQL.
- `frontend/`: Next.js frontend application.

## Prerequisites
- **Go** (1.21 or later)
- **Node.js** (v18 or later)
- **PostgreSQL**
- **Git**

## Backend Setup
1. Navigate to the backend directory:
   ```bash
   cd backend
   ```
2. Copy the example environment file:
   ```bash
   cp .env.example .env
   ```
3. Configure your database, mail, and JWT settings in `.env`.
4. Generate RSA keys for JWT (RS256):
   ```bash
   mkdir -p backend/keys
   # Generate private key
   openssl genrsa -out backend/keys/private.pem 2048
   # Extract public key
   openssl rsa -in backend/keys/private.pem -pubout -out backend/keys/public.pem
   ```
   *Note: `private.pem` is ignored by git for security. `public.pem` can be committed if needed, but ensure you generate your own pair for local development.*

5. Install dependencies:
   ```bash
   go mod download
   ```
6. Run the server (migrations will run automatically):
   ```bash
   go run main.go
   ```
   The API will be available at `http://localhost:8080`.

## Frontend Setup
1. Navigate to the frontend directory:
   ```bash
   cd frontend
   ```
2. Install dependencies:
   ```bash
   npm install
   ```
3. Configure your frontend environment variables in `.env.local` if necessary (e.g., `NEXT_PUBLIC_API_URL`).
4. Start the development server:
   ```bash
   npm run dev
   ```
   The application will be available at `http://localhost:3000`.

## Design Principles
For a detailed look at the system architecture, database design, and security enforcement, please refer to [DESIGN.md](./DESIGN.md).
