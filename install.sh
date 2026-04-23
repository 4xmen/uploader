#!/bin/bash

# Ensure script is run as root
if [ "$EUID" -ne 0 ]; then
  echo "Please run as root using sudo."
  exit
fi

APP_NAME="xmen-uploader"
APP_BIN="xmen-uploader_linux_amd64"
INSTALL_DIR="/opt/$APP_NAME"
USER="xmen-uploader"
GROUP="xmen-uploader"

# Check if binary exists in current directory
if [ ! -f "$APP_BIN" ]; then
    echo "Error: Binary $APP_BIN not found in current directory."
    echo "Please build the project for Linux first using 'make build-linux'"
    exit 1
fi

echo "Creating user and group '$USER'..."
groupadd -f $GROUP
id -u $USER &>/dev/null || useradd -r -s /bin/false -g $GROUP -d $INSTALL_DIR $USER

echo "Setting up installation directory ($INSTALL_DIR)..."
mkdir -p $INSTALL_DIR
mkdir -p $INSTALL_DIR/uploads

echo "Copying application files..."
cp $APP_BIN $INSTALL_DIR/

# Create .env if it doesn't exist
ENV_FILE="$INSTALL_DIR/.env"
if [ ! -f "$ENV_FILE" ]; then
    echo "Creating default .env file..."
    cat <<EOF > $ENV_FILE
PORT=8089
MAX_UPLOAD_SIZE_MB=50
MAX_ALLOWED_LIFETIME_DAYS=1
ADMIN_SECRET=13651385
DATABASE_PATH=uploader.db
STORAGE_PATH=./uploads
EOF
else
    echo ".env file already exists. Skipping creation."
fi

echo "Setting ownership and permissions..."
chown -R $USER:$GROUP $INSTALL_DIR
chmod +x $INSTALL_DIR/$APP_BIN
# Ensure only the app user can read the .env file (for security)
chmod 600 $ENV_FILE

echo "Creating systemd service..."
SERVICE_FILE="/etc/systemd/system/$APP_NAME.service"
cat <<EOF > $SERVICE_FILE
[Unit]
Description=XMen Uploader Service
After=network.target

[Service]
Type=simple
User=$USER
Group=$GROUP
WorkingDirectory=$INSTALL_DIR
ExecStart=$INSTALL_DIR/$APP_BIN
Restart=on-failure
RestartSec=5
LimitNOFILE=65535

[Install]
WantedBy=multi-user.target
EOF

echo "Reloading systemd daemon..."
systemctl daemon-reload

echo "Enabling and starting $APP_NAME service..."
systemctl enable $APP_NAME
systemctl restart $APP_NAME

echo "======================================"
echo "Installation complete!"
echo "The application is installed in $INSTALL_DIR"
echo "It runs as the restricted user: $USER"
echo ""
echo "You can check the status using:"
echo "sudo systemctl status $APP_NAME"
echo ""
echo "You can view logs using:"
echo "sudo journalctl -fu $APP_NAME"
echo "======================================"
