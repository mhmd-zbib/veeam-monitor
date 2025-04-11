# Installation script for Veeam Backup Monitor
# Must be run with administrator privileges

# Check if running as administrator
$isAdmin = ([Security.Principal.WindowsPrincipal] [Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
if (-not $isAdmin) {
    Write-Error "This script must be run as Administrator. Please restart with elevated privileges."
    exit 1
}

# Set the installation directory
$installDir = "$env:ProgramFiles\VeeamBackupMonitor"

# Create installation directory if it doesn't exist
if (-not (Test-Path $installDir)) {
    New-Item -ItemType Directory -Path $installDir | Out-Null
    Write-Host "Created installation directory: $installDir"
}

# Build the application
Write-Host "Building the application..."
try {
    go build -o "$installDir\veeam-monitor.exe"
    if (-not $?) {
        throw "Failed to build application"
    }
} catch {
    Write-Error "Failed to build the Go application: $_"
    exit 1
}

# Copy configuration file
Write-Host "Copying configuration file..."
Copy-Item -Path ".\config.json" -Destination "$installDir" -Force

# Create logs directory
if (-not (Test-Path "$installDir\logs")) {
    New-Item -ItemType Directory -Path "$installDir\logs" | Out-Null
    Write-Host "Created logs directory: $installDir\logs"
}

# Check if NSSM is available
$nssmPath = "$env:ProgramFiles\nssm\nssm.exe"
$installService = $true

if (-not (Test-Path $nssmPath)) {
    $installService = $false
    Write-Warning "NSSM not found at $nssmPath. The service will not be installed automatically."
    Write-Host "You can download NSSM from https://nssm.cc and install the service manually."
}

# Install as a service if NSSM is available
if ($installService) {
    Write-Host "Installing Windows service using NSSM..."
    
    # Remove existing service if it exists
    & $nssmPath stop VeeamBackupMonitor 2>$null
    & $nssmPath remove VeeamBackupMonitor confirm 2>$null
    
    # Install new service
    & $nssmPath install VeeamBackupMonitor "$installDir\veeam-monitor.exe"
    & $nssmPath set VeeamBackupMonitor AppDirectory "$installDir"
    & $nssmPath set VeeamBackupMonitor Description "Monitors Veeam Backup & Replication jobs and sends email alerts on failures"
    & $nssmPath set VeeamBackupMonitor Start SERVICE_AUTO_START
    
    # Start the service
    & $nssmPath start VeeamBackupMonitor
    
    if ($?) {
        Write-Host "Service installed and started successfully."
    } else {
        Write-Warning "Failed to install or start the service."
    }
}

Write-Host ""
Write-Host "Installation completed!"
Write-Host "Veeam Backup Monitor has been installed to: $installDir"
Write-Host ""
Write-Host "Don't forget to customize the configuration in $installDir\config.json"
Write-Host "You may need to allow the application through the Windows Firewall if it needs to send emails through a remote SMTP server."
if (-not $installService) {
    Write-Host ""
    Write-Host "To run the application, execute: $installDir\veeam-monitor.exe"
} 