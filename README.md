# XMen Uploader

A modern, fast, and minimal file uploader built with Go and vanilla web technologies. Features an Apple-like UI with drag-and-drop support, download limits, and automated file expiration.

## Features

- **No Login Required**: Simple drag-and-drop file sharing.
- **Configurable Limits**: Set maximum file upload sizes and maximum lifetime across the server.
- **Download Limits**: Users can specify how many times a file can be downloaded before it expires.
- **Lifetime Expiration**: Users can set an expiration time (in hours). Background tasks automatically clean up expired files.
- **Admin Dashboard**: View total active files, downloads, and storage usage.
- **Secure**: Built-in rate limiting (global and per-upload) and path traversal protections.
- **Modern UI**: Apple-style glassmorphism, responsive design, and automatic dark mode support.

## Getting Started

### Prerequisites

- Go 1.21 or higher installed on your system.

### Installation

1. Clone the repository or navigate to the project folder:
   ```bash
   cd xmen-uploader
   ```

2. Download dependencies:
   ```bash
   go mod download
   ```

3. Configure your environment. The application will use a `.env` file in the root directory if one exists, or fallback to system environment variables. You can customize the `.env` file:
   ```env
   PORT=8080
   MAX_UPLOAD_SIZE_MB=50
   MAX_ALLOWED_LIFETIME_DAYS=7
   ADMIN_SECRET=xmenadmin
   DATABASE_PATH=uploader.db
   STORAGE_PATH=./uploads
   ALLOWED_ORIGINS=*
   ```

4. Run the application:
   ```bash
   go run main.go
   ```

5. Open your browser and navigate to `http://localhost:8080`

### Admin Dashboard

Navigate to `http://localhost:8080/admin` and use the `ADMIN_SECRET` defined in your `.env` to view server statistics.

## Security Features

- **Rate Limiting**: Uses `httprate` to restrict global API requests (100 req/min per IP) and strict limits on the upload route (10 uploads/min per IP).
- **Directory Traversal Protection**: Uses strict filename parsing (`filepath.Base`) to ensure files are only served from the intended `uploads/` directory.
- **File Expiration Enforcement**: Both the metadata API and the download stream strictly enforce expiration timestamps and download counts, returning HTTP 410 (Gone) immediately.
