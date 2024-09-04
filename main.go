package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/gin-gonic/gin"
)

// MQTT configuration
var (
	mqttBroker   = "tcp://localhost:1883" // Change this to your MQTT broker address
	mqttTopic    = "file/bin"             // Topic to send the file to
	mqttClientID = "go_mqtt_client"
	mqttClient   mqtt.Client
)

// Initialize MQTT connection
func initMQTT() {
	opts := mqtt.NewClientOptions().AddBroker(mqttBroker).SetClientID(mqttClientID)
	mqttClient = mqtt.NewClient(opts)
	if token := mqttClient.Connect(); token.Wait() && token.Error() != nil {
		log.Fatalf("Failed to connect to MQTT broker: %v", token.Error())
	}
	fmt.Println("Connected to MQTT broker")
}

func main() {
	// Initialize MQTT
	initMQTT()

	// Initialize Gin router
	router := gin.Default()

	// Define the upload route
	router.POST("/upload", func(c *gin.Context) {
		// Retrieve the file from the form-data
		file, err := c.FormFile("file")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to get the file"})
			return
		}

		// Ensure file has a .bin extension
		if ext := file.Filename[len(file.Filename)-4:]; ext != ".bin" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Only .bin files are allowed"})
			return
		}

		// Save the file locally
		if err := c.SaveUploadedFile(file, file.Filename); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save the file"})
			return
		}

		// Read the file content
		fileContent, err := os.ReadFile(file.Filename)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read the file"})
			return
		}

		// Publish the file content to the MQTT topic
		token := mqttClient.Publish(mqttTopic, 0, false, fileContent)
		token.Wait()
		if token.Error() != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to publish to MQTT"})
			return
		}

		// Remove the saved file after publishing
		os.Remove(file.Filename)

		// Respond with success
		c.JSON(http.StatusOK, gin.H{"message": "File uploaded and published to MQTT"})
	})

	// Start the server
	router.Run(":8080")
}
