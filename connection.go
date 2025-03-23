package main

import (
	"log"
	"os"
	"os/exec"
	"strings"
)

func main() {
	// Extract MySQL connection details from environment variables
	mysqlUser := os.Getenv("MYSQL_USER")
	mysqlPass := os.Getenv("MYSQL_PASSWORD")
	mysqlHost := os.Getenv("MYSQL_HOST")
	mysqlPort := os.Getenv("MYSQL_PORT")
	databases := strings.Split(os.Getenv("DATABASES"), ",")

	if mysqlUser == "" || mysqlPass == "" || mysqlHost == "" || mysqlPort == "" || len(databases) == 0 {
		log.Fatal("Required environment variables are missing")
	}

	log.Printf("Testing connection to MySQL server: %s:%s", mysqlHost, mysqlPort)
	log.Printf("Using credentials for user: %s", mysqlUser)
	log.Printf("Databases to test: %v", databases)

	// First test connectivity with a simple SQL query
	testConnectivity(mysqlUser, mysqlPass, mysqlHost, mysqlPort)

	// Then check each database
	for _, dbName := range databases {
		testDatabase(mysqlUser, mysqlPass, mysqlHost, mysqlPort, dbName)
	}
}

func testConnectivity(user, pass, host, port string) {
	log.Println("=== TESTING BASIC CONNECTIVITY ===")
	
	// Use --connect-timeout to prevent long hangs
	cmd := exec.Command("mysql",
		"-u", user,
		"-p"+pass,
		"-h", host,
		"-P", port,
		"--connect-timeout=10",
		"-e", "SELECT 1 AS connection_test")

	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("❌ Connection test failed: %v", err)
		log.Printf("Output: %s", string(output))
		return
	}

	log.Printf("✅ Connection test passed: %s", string(output))
}

func testDatabase(user, pass, host, port, dbName string) {
	log.Printf("=== TESTING DATABASE: %s ===", dbName)
	
	// Test if we can access the database
	cmd := exec.Command("mysql",
		"-u", user,
		"-p"+pass,
		"-h", host,
		"-P", port,
		"--connect-timeout=10",
		dbName,
		"-e", "SHOW TABLES")

	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("❌ Database access test failed for %s: %v", dbName, err)
		log.Printf("Output: %s", string(output))
		return
	}

	log.Printf("✅ Successfully accessed database %s", dbName)
	log.Printf("Tables found: %s", string(output))

	// Try a small mysqldump test
	log.Printf("Testing mysqldump for %s", dbName)
	dumpCmd := exec.Command("mysqldump",
		"-u", user,
		"-p"+pass,
		"-h", host,
		"-P", port,
		"--column-statistics=0",
		"--no-data", // Only dump structure to keep it fast
		dbName)

	dumpOutput, err := dumpCmd.CombinedOutput()
	if err != nil {
		log.Printf("❌ mysqldump test failed for %s: %v", dbName, err)
		log.Printf("Output: %s", string(dumpOutput))
		return
	}

	log.Printf("✅ mysqldump test successful for %s", dbName)
	log.Printf("Output size: %d bytes", len(dumpOutput))
}
