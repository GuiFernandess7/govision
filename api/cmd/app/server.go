package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"

	middlewares "govision/internal/middlewares"
	routes "govision/internal/routes"
	rabbitmqConn "govision/services/rabbitmq"

	file "govision/internal/modules/file"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lmicroseconds | log.Lshortfile)
	_ = godotenv.Load()

	port := os.Getenv("API_PORT")
	if port == "" {
		port = "8080"
	}

	publisher := rabbitmqConn.PublisherFactory()

	e := echo.New()
	e = middlewares.ApplySecurityMiddlewares(e)

	fileHandler := file.NewHandler(publisher)
	routes.InitRoutes(e, fileHandler)
	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      e,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	log.Println("Server Listening on port: port")
	if err := srv.ListenAndServe(); err != nil {
		log.Fatalln("error starting server: ", err)
	}
}
