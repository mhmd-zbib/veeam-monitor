@echo off
REM Veeam Backup Monitor Runner
REM This script launches the Veeam Backup Monitor with command-line parameters
REM Replace the values below with your actual settings

SET VEEAM_SERVER=veeam-server.example.com
SET EMAIL_FROM=alerts@example.com
SET EMAIL_PASSWORD=yourpassword
SET EMAIL_TO=admin@example.com
SET SMTP_SERVER=smtp.example.com

REM Launch the monitor
veeam-monitor.exe -veeamserver "%VEEAM_SERVER%" -from "%EMAIL_FROM%" -password "%EMAIL_PASSWORD%" -to "%EMAIL_TO%" -smtp "%SMTP_SERVER%"

REM If you want to use a custom config file instead
REM veeam-monitor.exe -config "custom-config.json" 