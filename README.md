# Veeam Backup Monitor

A lightweight, robust Go application for monitoring Veeam Backup & Replication jobs and sending email notifications when issues occur.

## Features

- Monitors Veeam backup jobs for:
  - Failed jobs
  - Warning-state jobs 
  - Long-running tasks exceeding a defined threshold
- Sends detailed email notifications via local mail server
- Configurable check intervals
- Comprehensive logging
- Command-line parameter support for quick configuration

## Requirements

- Go 1.16 or higher
- Windows Server with Veeam Backup & Replication installed
- Veeam PowerShell module (typically installed with Veeam)
- Local or remote SMTP server for sending emails

## Installation

1. Clone or download this repository
2. Configure the `config.json` file with your settings
3. Build the application:

```
cd veeam-monitor
go build
```

Alternatively, you can use the provided PowerShell installation script:

```
.\install.ps1
```

## Usage

### Basic Usage

Run the executable with the default configuration:

```
.\veeam-monitor.exe
```

### Command-line Parameters

You can override configuration settings with command-line parameters:

```
.\veeam-monitor.exe -veeamserver "veeam-server.example.com" -from "alerts@example.com" -password "emailpassword" -to "admin@example.com" -smtp "smtp.example.com"
```

Available parameters:

- `-veeamserver`: Veeam server address
- `-from`: Sender email address
- `-password`: Sender email password
- `-to`: Recipient email address
- `-smtp`: SMTP server address
- `-config`: Path to configuration file (default: "config.json")

Parameters specified on the command line will override those in the config file.

## Configuration

Edit the `config.json` file to customize the monitoring settings:

```json
{
    "veeamPowerShellModule": "Veeam.Backup.PowerShell",
    "veeamServerAddress": "localhost",
    "checkIntervalMinutes": 15,
    "smtpServer": "localhost",
    "smtpPort": 25,
    "emailFrom": "veeam-monitor@example.com",
    "emailTo": ["admin@example.com"],
    "emailPassword": "",
    "monitorFailedJobs": true,
    "monitorWarningJobs": true,
    "monitorRunningJobs": true,
    "longRunningThreshold": 120
}
```

Configuration options:

- `veeamPowerShellModule`: Name of the Veeam PowerShell module (usually "Veeam.Backup.PowerShell")
- `veeamServerAddress`: Hostname or IP address of the Veeam Backup & Replication server
- `checkIntervalMinutes`: How often to check for problems (in minutes)
- `smtpServer`: SMTP server address
- `smtpPort`: SMTP server port
- `emailFrom`: Sender email address
- `emailTo`: List of recipient email addresses
- `emailPassword`: Password for SMTP authentication (if required)
- `monitorFailedJobs`: Set to true to monitor failed jobs
- `monitorWarningJobs`: Set to true to monitor jobs with warnings
- `monitorRunningJobs`: Set to true to monitor long-running jobs
- `longRunningThreshold`: Threshold in minutes for considering a job as "long-running"

## Running as a Service

To run the application as a Windows service, you can use NSSM (Non-Sucking Service Manager):

1. Download NSSM from [nssm.cc](https://nssm.cc/)
2. Install the service:

```
nssm install VeeamBackupMonitor c:\path\to\veeam-monitor.exe
nssm set VeeamBackupMonitor AppDirectory c:\path\to\veeam-monitor
```

3. Start the service:

```
nssm start VeeamBackupMonitor
```

The installation script will attempt to do this automatically if NSSM is installed.

## Logs

Logs are stored in the `logs` directory with daily rotation. Each log includes:

- Job status checks
- Error information
- Email notification status

## Extending the Application

To monitor additional aspects of Veeam jobs:

1. Modify the PowerShell commands in the monitoring functions
2. Add additional filters or checks based on your requirements
3. Customize the email notification format in `sendEmailAlert()` function

## Troubleshooting

If you encounter issues:

1. Check the log files for detailed error messages
2. Verify that the Veeam PowerShell module is installed and accessible
3. Test SMTP connectivity independently
4. Ensure the application has appropriate permissions to access Veeam

## License

This project is open source and available under the MIT License. 