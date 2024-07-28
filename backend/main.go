package main

import (
	"context"
	"fmt"
	"net"
	"signal0ne/api/routers"
	"signal0ne/cmd/config"
	"signal0ne/internal/controllers"
	"signal0ne/internal/tools"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func main() {
	var server = gin.Default()

	cfg := config.GetInstance()
	if cfg == nil {
		panic("CRITICAL: unable to load config")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	mongoConn, err := tools.InitMongoClient(ctx, cfg.MongoUri)
	if err != nil {
		panic(
			fmt.Sprintf("Failed to establish connectiom to %s, error: %s",
				strings.Split(cfg.MongoUri, "/")[2],
				err),
		)
	}
	defer mongoConn.Disconnect(ctx)

	namespacesCollection := mongoConn.Database("signalone").Collection("namespaces")
	workflowsCollection := mongoConn.Database("signalone").Collection("workflows")
	_ = mongoConn.Database("signalone").Collection("users")

	conn, err := net.DialTimeout("unix", cfg.IPCSocket, (15 * time.Second))
	if err != nil {
		panic(
			fmt.Sprintf("Failed to establish connectiom to %s, error: %s",
				cfg.IPCSocket,
				err),
		)
	} else {
		defer conn.Close()
	}

	tools.Initialize(ctx, namespacesCollection)

	corsConfig := cors.DefaultConfig()
	corsConfig.AllowOrigins = []string{"*"}
	corsConfig.AllowHeaders = []string{"*"}
	corsConfig.AllowCredentials = true

	server.Use(cors.New(corsConfig))

	routerApiGroup := server.Group("/api")
	routerApiGroup.GET("/ping", func(ctx *gin.Context) {
		ctx.JSON(200, gin.H{
			"message": "pong",
		})
	})

	mainController := controllers.NewMainController()
	namespaceController := controllers.NewNamespaceController()
	workflowController := controllers.NewWorkflowController(workflowsCollection, namespacesCollection, cfg.Server)
	mainRouter := routers.NewMainRouter(mainController, namespaceController, workflowController)
	mainRouter.RegisterRoutes(routerApiGroup)

	//==========REMOVE BEFORE RELEASE==========
	_, err = conn.Write([]byte("Hello I am Go!"))
	if err != nil {
		fmt.Printf("Failed to send data: %s", err)
	}

	// Receive response
	buffer := make([]byte, 1024)
	n, err := conn.Read(buffer)
	if err != nil {
		fmt.Printf("Failed to read response: %s", err)
	}

	fmt.Printf("%s\n", buffer[:n])
	//===================

	server.Run(":" + cfg.Server.ServerPort)
}
