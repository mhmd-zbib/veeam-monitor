package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/smtp"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// Configuration for the application
type Config struct {
	VeeamPowerShellModule string   `json:"veeamPowerShellModule"`
	VeeamServerAddress    string   `json:"veeamServerAddress"`
	CheckIntervalMinutes  int      `json:"checkIntervalMinutes"`
	SMTPServer            string   `json:"smtpServer"`
	SMTPPort              int      `json:"smtpPort"`
	EmailFrom             string   `json:"emailFrom"`
	EmailTo               []string `json:"emailTo"`
	EmailPassword         string   `json:"emailPassword"`
	MonitorFailedJobs     bool     `json:"monitorFailedJobs"`
	MonitorWarningJobs    bool     `json:"monitorWarningJobs"`
	MonitorRunningJobs    bool     `json:"monitorRunningJobs"`
	LongRunningThreshold  int      `json:"longRunningThreshold"` // In minutes
}

// Represents a Veeam job status
type JobStatus struct {
	Name        string
	Status      string
	StartTime   string
	EndTime     string
	Description string
	Duration    string
}

func main() {
	// Define command-line arguments
	veeamServer := flag.String("veeamserver", "", "Veeam server address")
	emailFrom := flag.String("from", "", "Sender email address")
	emailPassword := flag.String("password", "", "Sender email password")
	emailTo := flag.String("to", "", "Recipient email address")
	smtpServer := flag.String("smtp", "", "SMTP server address")
	configFile := flag.String("config", "config.json", "Path to configuration file")
	
	// Parse command-line flags
	flag.Parse()
	
	// Set up logging
	logFile, err := setupLogging()
	if err != nil {
		log.Printf("Error setting up logging: %v. Will log to console only.\n", err)
	} else {
		defer logFile.Close()
	}

	// Load configuration from file
	config, err := loadConfig(*configFile)
	if err != nil {
		log.Printf("Error loading configuration: %v\n", err)
		log.Println("Will use default values and command-line parameters")
		// Create default config if file loading failed
		config = &Config{
			VeeamPowerShellModule: "Veeam.Backup.PowerShell",
			CheckIntervalMinutes:  15,
			SMTPPort:              25,
			MonitorFailedJobs:     true,
			LongRunningThreshold:  120,
		}
	}

	// Override config with command-line parameters if provided
	if *veeamServer != "" {
		config.VeeamServerAddress = *veeamServer
		log.Printf("Using Veeam server from command line: %s\n", config.VeeamServerAddress)
	}
	
	if *emailFrom != "" {
		config.EmailFrom = *emailFrom
		log.Printf("Using sender email from command line: %s\n", config.EmailFrom)
	}
	
	if *emailPassword != "" {
		config.EmailPassword = *emailPassword
		log.Println("Using email password from command line")
	}
	
	if *emailTo != "" {
		config.EmailTo = []string{*emailTo}
		log.Printf("Using recipient email from command line: %s\n", config.EmailTo[0])
	}
	
	if *smtpServer != "" {
		config.SMTPServer = *smtpServer
		log.Printf("Using SMTP server from command line: %s\n", config.SMTPServer)
	}

	// Validate essential configuration
	if config.VeeamServerAddress == "" {
		log.Println("Warning: No Veeam server address specified")
	}
	
	if config.EmailFrom == "" || len(config.EmailTo) == 0 || config.SMTPServer == "" {
		log.Println("Warning: Email configuration incomplete. Notifications will not be sent.")
	}

	log.Println("Starting Veeam backup monitoring service")

	// Main monitoring loop
	for {
		log.Println("Checking Veeam backup job statuses...")
		
		// Monitor different job types based on configuration
		var problematicJobs []JobStatus
		
		if config.MonitorFailedJobs {
			failedJobs, err := getJobsByStatus(config, "Failed")
			if err != nil {
				log.Printf("Error checking failed jobs: %v\n", err)
			} else {
				log.Printf("Found %d failed jobs\n", len(failedJobs))
				problematicJobs = append(problematicJobs, failedJobs...)
			}
		}
		
		if config.MonitorWarningJobs {
			warningJobs, err := getJobsByStatus(config, "Warning")
			if err != nil {
				log.Printf("Error checking warning jobs: %v\n", err)
			} else {
				log.Printf("Found %d warning jobs\n", len(warningJobs))
				problematicJobs = append(problematicJobs, warningJobs...)
			}
		}
		
		if config.MonitorRunningJobs {
			longRunningJobs, err := getLongRunningJobs(config)
			if err != nil {
				log.Printf("Error checking long-running jobs: %v\n", err)
			} else {
				log.Printf("Found %d long-running jobs\n", len(longRunningJobs))
				problematicJobs = append(problematicJobs, longRunningJobs...)
			}
		}
		
		// Send email notifications if there are problematic jobs
		if len(problematicJobs) > 0 {
			if err := sendEmailAlert(problematicJobs, config); err != nil {
				log.Printf("Error sending email alert: %v\n", err)
			} else {
				log.Println("Email alert sent successfully")
			}
		} else {
			log.Println("No problematic jobs found")
		}

		// Sleep until next check
		log.Printf("Sleeping for %d minutes until next check\n", config.CheckIntervalMinutes)
		time.Sleep(time.Duration(config.CheckIntervalMinutes) * time.Minute)
	}
}

// Setup logging to file and console
func setupLogging() (*os.File, error) {
	// Create logs directory if it doesn't exist
	if err := os.MkdirAll("logs", 0755); err != nil {
		return nil, err
	}

	// Create log file with timestamp in name
	timestamp := time.Now().Format("2006-01-02")
	logPath := filepath.Join("logs", fmt.Sprintf("veeam-monitor-%s.log", timestamp))
	
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}

	// Set up multi-writer for console and file logging
	multiWriter := os.Stdout
	
	// Set log output to both file and console
	log.SetOutput(multiWriter)
	
	return logFile, nil
}

// Load configuration from JSON file
func loadConfig(filePath string) (*Config, error) {
	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("error reading config file: %v", err)
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("error parsing config file: %v", err)
	}

	// Set defaults for any missing values
	if config.CheckIntervalMinutes < 1 {
		log.Println("Warning: Check interval is less than 1 minute, setting to default of 15 minutes")
		config.CheckIntervalMinutes = 15
	}
	
	if !config.MonitorFailedJobs && !config.MonitorWarningJobs && !config.MonitorRunningJobs {
		log.Println("Warning: No monitoring options enabled, enabling failed job monitoring by default")
		config.MonitorFailedJobs = true
	}
	
	if config.LongRunningThreshold < 1 {
		config.LongRunningThreshold = 120 // Default to 2 hours
		log.Println("Warning: Long running threshold not set, defaulting to 120 minutes")
	}

	return &config, nil
}

// Get jobs by status (Failed, Warning, etc.)
func getJobsByStatus(config *Config, status string) ([]JobStatus, error) {
	// PowerShell command to get jobs with specified status
	psCommand := fmt.Sprintf(`
		Import-Module %s
		if ("%s" -ne "") {
			$Server = Connect-VBRServer -Server %s
		}
		Get-VBRJob | Where-Object {$_.LastResult -eq "%s"} | Select-Object Name,LastResult,LastStart,LastEnd,Description | ConvertTo-Csv -NoTypeInformation
		if ("%s" -ne "") {
			Disconnect-VBRServer
		}
	`, config.VeeamPowerShellModule, config.VeeamServerAddress, config.VeeamServerAddress, status, config.VeeamServerAddress)

	// Execute PowerShell command
	cmd := exec.Command("powershell", "-Command", psCommand)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to execute PowerShell command for %s jobs: %v", status, err)
	}

	// Parse the CSV output
	return parseJobStatusOutput(string(output), status)
}

// Get long-running jobs
func getLongRunningJobs(config *Config) ([]JobStatus, error) {
	// PowerShell command to get currently running jobs
	psCommand := fmt.Sprintf(`
		Import-Module %s
		if ("%s" -ne "") {
			$Server = Connect-VBRServer -Server %s
		}
		$runningJobs = Get-VBRJob | Where-Object {$_.IsRunning -eq $true} | Select-Object Name,@{Name="Status";Expression={"Running"}},@{Name="StartTime";Expression={$_.FindLastSession().CreationTime}},@{Name="EndTime";Expression={"N/A"}},@{Name="Description";Expression={"Currently running"}},@{Name="Duration";Expression={((Get-Date) - $_.FindLastSession().CreationTime).TotalMinutes}}
		$longRunningJobs = $runningJobs | Where-Object {$_.Duration -gt %d}
		$longRunningJobs | ConvertTo-Csv -NoTypeInformation
		if ("%s" -ne "") {
			Disconnect-VBRServer
		}
	`, config.VeeamPowerShellModule, config.VeeamServerAddress, config.VeeamServerAddress, config.LongRunningThreshold, config.VeeamServerAddress)

	// Execute PowerShell command
	cmd := exec.Command("powershell", "-Command", psCommand)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to execute PowerShell command for long-running jobs: %v", err)
	}

	// Parse the CSV output
	jobs, err := parseJobStatusOutput(string(output), "Running")
	if err != nil {
		return nil, err
	}
	
	// Add duration information to job description
	for i := range jobs {
		jobs[i].Description = fmt.Sprintf("Long-running job (over %d minutes): %s", 
			config.LongRunningThreshold, jobs[i].Description)
	}
	
	return jobs, nil
}

// Parse the CSV output from PowerShell
func parseJobStatusOutput(output string, status string) ([]JobStatus, error) {
	lines := strings.Split(output, "\n")
	if len(lines) < 2 {
		return []JobStatus{}, nil
	}

	var jobs []JobStatus
	// Skip header line and process data lines
	for i := 1; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}

		// Parse CSV line (this is simplified - use a proper CSV parser in production)
		fields := strings.Split(line, ",")
		if len(fields) >= 5 {
			job := JobStatus{
				Name:        strings.Trim(fields[0], "\""),
				Status:      strings.Trim(fields[1], "\""),
				StartTime:   strings.Trim(fields[2], "\""),
				EndTime:     strings.Trim(fields[3], "\""),
				Description: strings.Trim(fields[4], "\""),
			}
			
			// Add duration if available (for running jobs)
			if len(fields) >= 6 {
				job.Duration = strings.Trim(fields[5], "\"")
			}
			
			jobs = append(jobs, job)
		}
	}

	return jobs, nil
}

// Send email alert for problematic jobs
func sendEmailAlert(problematicJobs []JobStatus, config *Config) error {
	// Create email subject and body
	subject := fmt.Sprintf("ALERT: %d Veeam Backup Jobs Need Attention", len(problematicJobs))
	
	// Group jobs by status for better readability
	failedJobs := []JobStatus{}
	warningJobs := []JobStatus{}
	runningJobs := []JobStatus{}
	
	for _, job := range problematicJobs {
		switch job.Status {
		case "Failed":
			failedJobs = append(failedJobs, job)
		case "Warning":
			warningJobs = append(warningJobs, job)
		case "Running":
			runningJobs = append(runningJobs, job)
		}
	}
	
	// Build email body
	body := "Veeam Backup & Replication Job Status Report\n"
	body += "===========================================\n\n"
	
	if len(failedJobs) > 0 {
		body += fmt.Sprintf("FAILED JOBS (%d):\n", len(failedJobs))
		body += "--------------\n"
		for _, job := range failedJobs {
			body += fmt.Sprintf("Job: %s\nStatus: %s\nStart Time: %s\nEnd Time: %s\nDescription: %s\n\n",
				job.Name, job.Status, job.StartTime, job.EndTime, job.Description)
		}
		body += "\n"
	}
	
	if len(warningJobs) > 0 {
		body += fmt.Sprintf("WARNING JOBS (%d):\n", len(warningJobs))
		body += "----------------\n"
		for _, job := range warningJobs {
			body += fmt.Sprintf("Job: %s\nStatus: %s\nStart Time: %s\nEnd Time: %s\nDescription: %s\n\n",
				job.Name, job.Status, job.StartTime, job.EndTime, job.Description)
		}
		body += "\n"
	}
	
	if len(runningJobs) > 0 {
		body += fmt.Sprintf("LONG-RUNNING JOBS (%d):\n", len(runningJobs))
		body += "---------------------\n"
		for _, job := range runningJobs {
			durationText := ""
			if job.Duration != "" {
				durationMin, _ := strings.Split(job.Duration, ".")[0], strings.Split(job.Duration, ".")[1]
				durationText = fmt.Sprintf(" (Running for %s minutes)", durationMin)
			}
			
			body += fmt.Sprintf("Job: %s\nStatus: %s%s\nStart Time: %s\nDescription: %s\n\n",
				job.Name, job.Status, durationText, job.StartTime, job.Description)
		}
	}
	
	body += "\nThis is an automated message from the Veeam Backup Monitor.\n"

	// Prepare email message
	msg := fmt.Sprintf("From: %s\r\n"+
		"To: %s\r\n"+
		"Subject: %s\r\n"+
		"\r\n"+
		"%s", config.EmailFrom, strings.Join(config.EmailTo, ", "), subject, body)

	// Connect to SMTP server
	var auth smtp.Auth
	if config.EmailPassword != "" {
		auth = smtp.PlainAuth("", config.EmailFrom, config.EmailPassword, config.SMTPServer)
	}
	
	// Send the email
	err := smtp.SendMail(
		fmt.Sprintf("%s:%d", config.SMTPServer, config.SMTPPort),
		auth,
		config.EmailFrom,
		config.EmailTo,
		[]byte(msg),
	)
	
	return err
} 