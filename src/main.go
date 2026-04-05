package main

import (
	"context"
	"flag"
	"os"
	"plugin"
	"strconv"

	"github.com/bbernhard/signal-cli-rest-api/api"
	"github.com/bbernhard/signal-cli-rest-api/client"
	docs "github.com/bbernhard/signal-cli-rest-api/docs"
	"github.com/bbernhard/signal-cli-rest-api/storage"
	"github.com/bbernhard/signal-cli-rest-api/utils"
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// @title Signal Cli REST API
// @version 1.0
// @description This is the Signal Cli REST API documentation.

// @tag.name General
// @tag.description Some general endpoints.

// @tag.name Devices
// @tag.description Register and link Devices.

// @tag.name Accounts
// @tag.description List registered and linked accounts

// @tag.name Groups
// @tag.description Create, List and Delete Signal Groups.

// @tag.name Messages
// @tag.description Send and Receive Signal Messages.

// @tag.name Attachments
// @tag.description List and Delete Attachments.

// @tag.name Profiles
// @tag.description Update Profile.

// @tag.name Identities
// @tag.description List and Trust Identities.

// @tag.name Reactions
// @tag.description React to messages.

// @tag.name Receipts
// @tag.description Send receipts for messages.

// @tag.name Search
// @tag.description Search the Signal Service.

// @tag.name Sticker Packs
// @tag.description List and Install Sticker Packs

// @host localhost:8080
// @schemes http
// @BasePath /

func main() {
	signalCliConfig := flag.String("signal-cli-config", "/home/.local/share/signal-cli/", "Config directory where signal-cli config is stored")
	attachmentTmpDir := flag.String("attachment-tmp-dir", "/tmp/", "Attachment tmp directory")
	avatarTmpDir := flag.String("avatar-tmp-dir", "/tmp/", "Avatar tmp directory")
	flag.Parse()

	logLevel := utils.GetEnv("LOG_LEVEL", "")
	if logLevel != "" {
		err := utils.SetLogLevel(logLevel)
		if err != nil {
			log.Error("Couldn't set log level to '", logLevel, "'. Falling back to the info log level")
			utils.SetLogLevel("info")
		}
	}

	if utils.GetEnv("SWAGGER_USE_HTTPS_AS_PREFERRED_SCHEME", "false") == "false" {
		docs.SwaggerInfo.Schemes = []string{"http", "https"}
	} else {
		docs.SwaggerInfo.Schemes = []string{"https", "http"}
	}

	router := gin.New()
	router.Use(gin.LoggerWithConfig(gin.LoggerConfig{
		SkipPaths: []string{"/v1/health"},
	}))
	router.Use(gin.Recovery())

	port := utils.GetEnv("PORT", "8080")
	if _, err := strconv.Atoi(port); err != nil {
		log.Fatal("Invalid PORT ", port, " set. PORT needs to be a number")
	}

	defaultSwaggerIp := utils.GetEnv("HOST_IP", "127.0.0.1")
	swaggerIp := utils.GetEnv("SWAGGER_IP", defaultSwaggerIp)
	swaggerHost := utils.GetEnv("SWAGGER_HOST", swaggerIp+":"+port)
	docs.SwaggerInfo.Host = swaggerHost

	log.Info("Started Signal Messenger REST API")

	webhookUrl := utils.GetEnv("RECEIVE_WEBHOOK_URL", "")

	jsonRpc2ClientConfigPath := *signalCliConfig + "/jsonrpc2.yml"
	signalCliApiConfigPath := *signalCliConfig + "/api-config.yml"
	signalClient := client.NewSignalClient(*signalCliConfig, *attachmentTmpDir, *avatarTmpDir, jsonRpc2ClientConfigPath, signalCliApiConfigPath, webhookUrl)
	err := signalClient.Init(60)
	if err != nil {
		log.Fatal("Couldn't init Signal Client: ", err.Error())
	}

	var store *storage.Store
	if databaseURL := os.Getenv("DATABASE_URL"); databaseURL != "" {
		store, err = storage.New(context.Background(), databaseURL)
		if err != nil {
			log.Fatal("Couldn't connect to database: ", err.Error())
		}
		defer store.Close()
	} else {
		log.Info("DATABASE_URL not set — message storage disabled")
	}

	a := api.NewApi(signalClient, store)
	v1 := router.Group("/v1")
	{
		v1.GET("/about", a.About)
		v1.GET("/health", a.Health)
		v1.GET("/configuration", a.GetConfiguration)
		v1.POST("/configuration", a.SetConfiguration)
		v1.GET("/accounts", a.GetAccounts)
		v1.GET("/search", a.SearchForNumbers)

		attachments := v1.Group("/attachments")
		{
			attachments.GET("", a.GetAttachments)
			attachments.GET("/:id", a.ServeAttachment)
			attachments.DELETE("/:id", a.RemoveAttachment)
		}

		accounts := v1.Group("/accounts/:number")
		{
			// Registration
			accounts.POST("/register", a.RegisterNumber)
			accounts.POST("/register/verify/:token", a.VerifyRegisteredNumber)
			accounts.DELETE("", a.UnregisterNumber)

			// Account settings
			accounts.PUT("/settings", a.UpdateAccountSettings)
			accounts.GET("/trust-mode", a.GetTrustMode)
			accounts.PUT("/trust-mode", a.SetTrustMode)
			accounts.POST("/rate-limit-challenge", a.SubmitRateLimitChallenge)
			accounts.POST("/username", a.SetUsername)
			accounts.DELETE("/username", a.RemoveUsername)
			accounts.POST("/pin", a.SetPin)
			accounts.DELETE("/pin", a.RemovePin)
			accounts.DELETE("/local-data", a.DeleteLocalAccountData)

			// Device linking
			accounts.GET("/qr-code", a.GetQrCodeLink)
			accounts.GET("/qr-code/raw", a.GetQrCodeLinkUri)
			accounts.GET("/devices", a.ListDevices)
			accounts.POST("/devices", a.AddDevice)
			accounts.DELETE("/devices/:deviceId", a.RemoveDevice)

			// Messages
			accounts.POST("/messages", a.Send)
			accounts.GET("/messages", a.PollMessages)
			accounts.GET("/messages/stream", a.StreamMessages)
			accounts.GET("/messages/history", a.GetMessages)
			accounts.POST("/messages/remote-delete", a.RemoteDelete)

			// Reactions & receipts
			accounts.POST("/reactions", a.SendReaction)
			accounts.DELETE("/reactions", a.RemoveReaction)
			accounts.POST("/receipts", a.SendReceipt)
			accounts.POST("/typing", a.SendStartTyping)
			accounts.DELETE("/typing", a.SendStopTyping)

			// Groups
			groups := accounts.Group("/groups")
			{
				groups.GET("", a.GetGroups)
				groups.POST("", a.CreateGroup)
				groups.GET("/:groupId", a.GetGroup)
				groups.PUT("/:groupId", a.UpdateGroup)
				groups.DELETE("/:groupId", a.DeleteGroup)
				groups.GET("/:groupId/avatar", a.GetGroupAvatar)
				groups.POST("/:groupId/block", a.BlockGroup)
				groups.POST("/:groupId/join", a.JoinGroup)
				groups.POST("/:groupId/quit", a.QuitGroup)
				groups.POST("/:groupId/members", a.AddMembersToGroup)
				groups.DELETE("/:groupId/members", a.RemoveMembersFromGroup)
				groups.POST("/:groupId/admins", a.AddAdminsToGroup)
				groups.DELETE("/:groupId/admins", a.RemoveAdminsFromGroup)
				groups.POST("/:groupId/pinned-message", a.PinMessageInGroup)
				groups.DELETE("/:groupId/pinned-message", a.UnpinMessageInGroup)
			}

			// Contacts
			contacts := accounts.Group("/contacts")
			{
				contacts.GET("", a.ListContacts)
				contacts.PUT("", a.UpdateContact)
				contacts.POST("/sync", a.SendContacts)
				contacts.GET("/:uuid", a.ListContact)
				contacts.GET("/:uuid/avatar", a.GetProfileAvatar)
			}

			// Identities
			accounts.GET("/identities", a.ListIdentities)
			accounts.PUT("/identities/:id/trust", a.TrustIdentity)

			// Profile
			accounts.PUT("/profile", a.UpdateProfile)

			// Sticker packs
			accounts.GET("/sticker-packs", a.ListInstalledStickerPacks)
			accounts.POST("/sticker-packs", a.AddStickerPack)

			// Polls
			accounts.POST("/polls", a.CreatePoll)
			accounts.POST("/polls/vote", a.VoteInPoll)
			accounts.DELETE("/polls", a.ClosePoll)
		}

		if utils.GetEnv("ENABLE_PLUGINS", "false") == "true" {
			signalCliRestApiPluginSharedObjDir := utils.GetEnv("SIGNAL_CLI_REST_API_PLUGIN_SHARED_OBJ_DIR", "")
			sharedObj, err := plugin.Open(signalCliRestApiPluginSharedObjDir + "signal-cli-rest-api_plugin_loader.so")
			if err != nil {
				log.Fatal("Couldn't load shared object: ", err)
			}

			pluginHandlerSymbol, err := sharedObj.Lookup("PluginHandler")
			if err != nil {
				log.Fatal("Couldn't get PluginHandler: ", err)
			}

			pluginHandler, ok := pluginHandlerSymbol.(utils.PluginHandler)
			if !ok {
				log.Fatal("Couldn't cast PluginHandler")
			}

			plugins := v1.Group("/plugins")
			{
				pluginConfigs := utils.NewPluginConfigs()
				err := pluginConfigs.Load("/plugins")
				if err != nil {
					log.Fatal("Couldn't load plugin configs: ", err.Error())
				}

				for _, pluginConfig := range pluginConfigs.Configs {
					if pluginConfig.Method == "GET" {
						plugins.GET(pluginConfig.Endpoint, pluginHandler.ExecutePlugin(pluginConfig))
					} else if pluginConfig.Method == "POST" {
						plugins.POST(pluginConfig.Endpoint, pluginHandler.ExecutePlugin(pluginConfig))
					} else if pluginConfig.Method == "DELETE" {
						plugins.DELETE(pluginConfig.Endpoint, pluginHandler.ExecutePlugin(pluginConfig))
					} else if pluginConfig.Method == "PUT" {
						plugins.PUT(pluginConfig.Endpoint, pluginHandler.ExecutePlugin(pluginConfig))
					}
				}
			}
		}
	}

	protocol := "http"
	if utils.GetEnv("SWAGGER_USE_HTTPS_AS_PREFERRED_SCHEME", "false") == "true" {
		protocol = "https"
	}

	swaggerUrl := ginSwagger.URL(protocol + "://" + swaggerHost + "/swagger/doc.json")
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler, swaggerUrl))

	router.Run()
}
