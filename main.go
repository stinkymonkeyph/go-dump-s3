package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

var (
	databases  = strings.Split(os.Getenv("DATABASES"), ",") // List of databases from environment variable
	bucketName = os.Getenv("S3_BUCKET")                     // S3 bucket name from environment variable
	region     = os.Getenv("S3_REGION")                     // S3 region from environment variable
	prefix     = os.Getenv("S3_PREFIX")                     // S3 prefix from environment variable
	discordURL = os.Getenv("DISCORD_URL")                   // Discord webhook URL from environment variable
	mysqlUser  = os.Getenv("MYSQL_USER")                    // MySQL username from environment variable
	mysqlPass  = os.Getenv("MYSQL_PASSWORD")                // MySQL password from environment variable
	mysqlHost  = os.Getenv("MYSQL_HOST")                    // MySQL host from environment variable
	mysqlPort  = os.Getenv("MYSQL_PORT")                    // MySQL port from environment variable
)

func main() {
	if bucketName == "" || region == "" || discordURL == "" || len(databases) == 0 ||
		mysqlUser == "" || mysqlPass == "" || mysqlHost == "" || mysqlPort == "" {
		errorMsg := "Required environment variables are missing or DATABASES list is empty"
		log.Println(errorMsg)
		return
	}

	// Create a session for AWS S3
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(region)},
	)
	if err != nil {
		errorMsg := fmt.Sprintf("Failed to create AWS session: %v", err)
		log.Println(errorMsg)
		sendDiscordNotification(errorMsg, nil, "")
		return
	}

	svc := s3.New(sess)

	for _, dbName := range databases {
		fileName := generateBackupFileName(dbName)
		log.Printf("Starting backup process for database: %s", dbName)
		
		err := backupAndUploadDatabase(svc, dbName, fileName)
		if err != nil {
			errorMsg := fmt.Sprintf("❌ Backup failed for -> %s (file: %s)", dbName, fileName)
			log.Printf("%s: %v", errorMsg, err)
			sendDiscordNotification(errorMsg, err, fileName)
		} else {
			successMsg := fmt.Sprintf("✅ Backup successful for -> %s (file: %s)", dbName, fileName)
			log.Println(successMsg)
			sendDiscordNotification(successMsg, nil, fileName)
		}
	}
}

func generateBackupFileName(dbName string) string {
	timestamp := time.Now().Format("20060102-150405")
	return fmt.Sprintf("%s-%s.sql", dbName, timestamp)
}

func backupAndUploadDatabase(svc *s3.S3, dbName string, fileName string) error {
	log.Printf("Creating temporary directory for backup of %s", dbName)
	// Create a temporary directory
	tempDir, err := ioutil.TempDir("", "backup")
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir) // Clean up temp directory

	filePath := filepath.Join(tempDir, fileName)

	// Backup the database to a file
	log.Printf("Executing mysqldump for %s", dbName)
	cmd := exec.Command("mysqldump",
    "-u", mysqlUser,
    "-p"+mysqlPass,
    "-h", mysqlHost,
    "-P", mysqlPort,
    dbName)
		
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	
	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("failed to execute mysqldump: %w, stderr: %s", err, stderr.String())
	}

	if stdout.Len() == 0 {
		return fmt.Errorf("mysqldump produced empty output")
	}

	// Write the dump to a file in the temp directory
	log.Printf("Writing backup to file: %s", filePath)
	err = os.WriteFile(filePath, stdout.Bytes(), 0644)
	if err != nil {
		return fmt.Errorf("failed to write backup file: %w", err)
	}

	// Check file size to make sure something was written
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return fmt.Errorf("failed to stat backup file: %w", err)
	}
	
	log.Printf("Backup file size: %d bytes", fileInfo.Size())
	if fileInfo.Size() == 0 {
		return fmt.Errorf("backup file is empty")
	}

	// Upload the file to S3
	log.Printf("Uploading backup to S3: %s", fileName)
	err = uploadToS3(svc, fileName, filePath)
	if err != nil {
		return fmt.Errorf("failed to upload backup to S3: %w", err)
	}
	
	log.Printf("Backup and upload completed successfully for %s", dbName)
	// Temp directory and file will be deleted automatically with defer
	return nil
}

func uploadToS3(svc *s3.S3, fileName, filePath string) error {
	log.Printf("Opening file for S3 upload: %s", filePath)
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Apply the S3 prefix if it exists
	s3Key := fileName
	if prefix != "" {
		s3Key = fmt.Sprintf("%s/%s", strings.TrimSuffix(prefix, "/"), fileName)
	}

	log.Printf("Putting object to S3 bucket: %s, key: %s", bucketName, s3Key)
	_, err = svc.PutObject(&s3.PutObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(s3Key),
		Body:   file,
	})
	if err != nil {
		return fmt.Errorf("failed to upload file to S3: %w", err)
	}

	log.Printf("S3 upload completed for: %s", s3Key)
	return nil
}

func sendDiscordNotification(message string, err error, fileName string) {
	content := message
	if err != nil {
		content = fmt.Sprintf("%s: %v", message, err)
	}

	log.Printf("Sending Discord notification: %s", content)
	
	// Use Discord webhook to send the message
	resp, err := http.PostForm(discordURL, url.Values{"content": {content}})
	if err != nil {
		log.Printf("Failed to send Discord notification: %v", err)
		return
	}
	defer resp.Body.Close()

	// Handle the 204 No Content response as success
	if resp.StatusCode == http.StatusNoContent {
		log.Println("Discord notification sent successfully")
	} else if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		log.Printf("Failed to send Discord notification, status: %s, response: %s", resp.Status, string(body))
	}
}
