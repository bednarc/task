package main

import (
	"os"
)

func main() {
	a := App{}

	dbUsername := os.Getenv("MONGO_ROOT_USERNAME")
	dbPassword := os.Getenv("MONGO_ROOT_PASSWORD")
	dbPortNumber := os.Getenv("MONGO_PORT_NUMBER")
	appPortNumber := os.Getenv("APP_PORT_NUMBER")

	c := Config{
		dbName:     "task",
		dbUsername: dbUsername,
		dbPassword: dbPassword,
		dbPort:     dbPortNumber,
	}
	a.Initialize(c)

	a.Run(":" + appPortNumber)
}
