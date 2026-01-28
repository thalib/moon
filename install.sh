#!/bin/bash
#
# Moon Installation Script
# Interactive installation script for Moon - Dynamic Headless Engine
#

set -e  # Exit on error

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Print functions
print_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

print_header() {
    echo ""
    echo -e "${GREEN}========================================${NC}"
    echo -e "${GREEN}$1${NC}"
    echo -e "${GREEN}========================================${NC}"
    echo ""
}

# Check for root privileges
check_root() {
    print_info "Checking for root privileges..."
    if [ "$EUID" -ne 0 ]; then
        print_error "This script must be run as root."
        echo ""
        echo "Please run this script with sudo:"
        echo "  sudo ./install.sh"
        echo ""
        exit 1
    fi
    print_success "Running with root privileges"
}

# Create moon user if it doesn't exist
create_user() {
    print_info "Checking for moon user..."
    if id "moon" &>/dev/null; then
        print_info "moon user already exists"
    else
        print_info "Creating moon system user..."
        useradd --system --no-create-home --shell /bin/false moon
        print_success "moon user created"
    fi
}

# Create required directories
create_directories() {
    print_info "Creating required directories..."
    
    # Create /var/lib/moon directory
    if [ ! -d "/var/lib/moon" ]; then
        mkdir -p /var/lib/moon
        chown moon:moon /var/lib/moon
        chmod 755 /var/lib/moon
        print_success "Created /var/lib/moon directory"
    else
        print_info "/var/lib/moon directory already exists"
        chown moon:moon /var/lib/moon
    fi
    
    # Create /opt/moon directory (working directory for systemd service)
    if [ ! -d "/opt/moon" ]; then
        mkdir -p /opt/moon
        chown moon:moon /opt/moon
        chmod 755 /opt/moon
        print_success "Created /opt/moon directory"
    else
        print_info "/opt/moon directory already exists"
        chown moon:moon /opt/moon
    fi
    
    # Create /var/log/moon directory (for log files)
    if [ ! -d "/var/log/moon" ]; then
        mkdir -p /var/log/moon
        chown moon:moon /var/log/moon
        chmod 755 /var/log/moon
        print_success "Created /var/log/moon directory"
    else
        print_info "/var/log/moon directory already exists"
        chown moon:moon /var/log/moon
    fi
}

# Check for moon binary in current directory
check_binary() {
    print_info "Checking for moon binary in current directory..."
    if [ ! -f "moon" ]; then
        print_error "moon binary not found in current directory."
        echo ""
        echo "Please build moon first (see docs/INSTALL.md for instructions)."
        echo ""
        echo "Build commands:"
        echo "  go build -o moon ./cmd/moon"
        echo ""
        echo "Or use Docker to build (run from project root directory):"
        echo '  sudo docker run --rm -v "$(pwd):/app" -v "$(pwd)/.gocache:/gocache" -w /app -e GOCACHE=/gocache golang:latest sh -c "go build -buildvcs=false -o moon ./cmd/moon"'
        echo ""
        exit 1
    fi
    print_success "moon binary found"
}

# Stop running moon service
stop_service() {
    print_info "Checking for running moon service..."
    if systemctl is-active --quiet moon.service 2>/dev/null; then
        print_warning "moon service is currently running"
        print_info "Stopping moon service..."
        systemctl stop moon.service
        print_success "moon service stopped"
    else
        print_info "moon service is not running"
    fi
}

# Prompt for file overwrite
prompt_overwrite() {
    local file=$1
    local response
    
    if [ -f "$file" ]; then
        print_warning "File $file already exists"
        while true; do
            read -t 30 -p "Do you want to overwrite it? [y/N]: " response || {
                echo ""
                print_warning "Timeout: defaulting to 'No'"
                return 1
            }
            case $response in
                [Yy]* )
                    return 0  # Yes, overwrite
                    ;;
                [Nn]* | "" )
                    return 1  # No, don't overwrite
                    ;;
                * )
                    echo "Please answer yes (y) or no (n)."
                    ;;
            esac
        done
    fi
    return 0  # File doesn't exist, proceed
}

# Copy moon binary
install_binary() {
    print_header "Installing Moon Binary"
    
    local dest="/usr/local/bin/moon"
    
    if prompt_overwrite "$dest"; then
        print_info "Copying moon binary to $dest..."
        cp moon "$dest"
        chmod +x "$dest"
        print_success "moon binary installed to $dest"
    else
        print_warning "Skipped installing moon binary"
    fi
}

# Copy configuration file
install_config() {
    print_header "Installing Configuration File"
    
    local src="samples/moon.conf"
    local dest="/etc/moon.conf"
    
    if [ ! -f "$src" ]; then
        print_error "Configuration file $src not found"
        exit 1
    fi
    
    if prompt_overwrite "$dest"; then
        print_info "Copying configuration file to $dest..."
        cp "$src" "$dest"
        chmod 644 "$dest"
        print_success "Configuration file installed to $dest"
        echo ""
        print_warning "IMPORTANT: Edit $dest and set your JWT_SECRET"
        print_warning "Or set MOON_JWT_SECRET environment variable in the service file"
    else
        print_warning "Skipped installing configuration file"
    fi
}

# Copy systemd service file
install_service() {
    print_header "Installing Systemd Service"
    
    local src="samples/moon.service"
    local dest="/etc/systemd/system/moon.service"
    
    if [ ! -f "$src" ]; then
        print_error "Service file $src not found"
        exit 1
    fi
    
    if prompt_overwrite "$dest"; then
        print_info "Copying service file to $dest..."
        cp "$src" "$dest"
        chmod 644 "$dest"
        print_success "Service file installed to $dest"
        echo ""
        print_warning "IMPORTANT: Set MOON_JWT_SECRET environment variable before starting the service"
        print_warning "Run: sudo systemctl edit moon.service"
        print_warning "Then add: Environment=\"MOON_JWT_SECRET=your-secret-here\""
    else
        print_warning "Skipped installing service file"
    fi
}

# Reload systemd and enable service
enable_service() {
    print_header "Configuring Systemd Service"
    
    print_info "Reloading systemd daemon..."
    systemctl daemon-reload
    print_success "Systemd daemon reloaded"
    
    print_info "Enabling moon service to start on boot..."
    systemctl enable moon.service
    print_success "moon service enabled"
}

# Start the service
start_service() {
    print_header "Starting Moon Service"
    
    print_info "Starting moon service..."
    systemctl start moon.service
    
    # Wait for the service to start with retry logic
    print_info "Waiting for service to start..."
    local max_attempts=10
    local attempt=1
    
    while [ $attempt -le $max_attempts ]; do
        if systemctl is-active --quiet moon.service; then
            print_success "moon service started successfully"
            return 0
        fi
        
        # Check if service is in failed state
        if systemctl is-failed --quiet moon.service; then
            print_error "moon service failed to start"
            print_warning "Check the status with: systemctl status moon.service"
            print_warning "Check logs with: journalctl -u moon.service -n 50"
            exit 1
        fi
        
        sleep 1
        attempt=$((attempt + 1))
    done
    
    print_error "Failed to start moon service (timed out after ${max_attempts} seconds)"
    print_warning "Check the status with: systemctl status moon.service"
    print_warning "Check logs with: journalctl -u moon.service -n 50"
    exit 1
}

# Show service status
show_status() {
    print_header "Moon Service Status"
    
    systemctl status moon.service --no-pager -l
}

# Print completion message
print_completion() {
    echo ""
    print_header "Installation Complete!"
    
    echo "Moon has been successfully installed and started."
    echo ""
    echo "Next steps:"
    echo "  1. Set the JWT secret: sudo systemctl edit moon.service"
    echo "     Add: Environment=\"MOON_JWT_SECRET=your-secret-here\""
    echo "  2. Edit /etc/moon.conf and configure your settings (optional)"
    echo "  3. Restart the service: sudo systemctl restart moon.service"
    echo "  4. Check logs: sudo journalctl -u moon.service -f"
    echo ""
    echo "Useful commands:"
    echo "  sudo systemctl status moon.service   # Check service status"
    echo "  sudo systemctl restart moon.service  # Restart service"
    echo "  sudo systemctl stop moon.service     # Stop service"
    echo "  sudo journalctl -u moon.service -f   # View live logs"
    echo ""
    print_success "Install complete."
    echo ""
}

# Main installation flow
main() {
    print_header "Moon Installation Script"
    echo "This script will install Moon - Dynamic Headless Engine"
    echo ""
    
    # Run all checks and installation steps
    check_root
    check_binary
    create_user
    create_directories
    stop_service
    install_binary
    install_config
    install_service
    enable_service
    start_service
    show_status
    print_completion
}

# Run main function
main
