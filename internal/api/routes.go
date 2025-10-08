package api

import (
	"database/sql"
	"html/template"
	"net/http"

	"castafiore-backend/internal/auth"
	"castafiore-backend/internal/config"
	"castafiore-backend/internal/subsonic"
	"castafiore-backend/internal/web"

	"github.com/gin-gonic/gin"
)

func SetupRoutes(router *gin.Engine, authService *auth.Service, db *sql.DB, cfg *config.Config) {
	// Configure HTML templates with custom functions
	funcMap := template.FuncMap{
		"add": func(a, b int) int { return a + b },
		"sub": func(a, b int) int { return a - b },
		"mul": func(a, b int) int { return a * b },
		"div": func(a, b int) int { return a / b },
		"mod": func(a, b int) int { return a % b },
	}

	tmpl := template.Must(template.New("").Funcs(funcMap).ParseGlob("web/templates/*"))
	router.SetHTMLTemplate(tmpl)

	// Create subsonic service
	subsonicService := subsonic.NewService(db, authService, cfg)

	// Create web controller
	webController := web.NewWebController(db, authService, cfg)

	// Authentication routes (no middleware)
	router.GET("/login", webController.LoginForm)
	router.POST("/login", webController.Login)
	router.GET("/logout", webController.Logout)

	// Redirect root to admin
	router.GET("/", func(c *gin.Context) {
		c.Redirect(http.StatusFound, "/admin")
	})

	// Static files
	router.Static("/static", "./static")

	// Web Admin Interface (with authentication middleware)
	admin := router.Group("/admin")
	admin.Use(webController.AuthMiddleware())
	{
		admin.GET("", webController.Dashboard)
		admin.GET("/", webController.Dashboard)
		admin.GET("/users", webController.Users)
		admin.GET("/users/create", webController.CreateUserForm)
		admin.POST("/users/create", webController.CreateUser)
		admin.GET("/users/:id/edit", webController.EditUserForm)
		admin.POST("/users/:id/edit", webController.EditUser)
		admin.POST("/users/:id/delete", webController.DeleteUser)
		admin.GET("/music", webController.MusicBrowser)
		admin.GET("/music/search", webController.SearchMusic)
		admin.GET("/settings", webController.Settings)
		admin.POST("/settings/update-music-path", webController.UpdateMusicPath)

		// API endpoints for admin interface
		admin.GET("/api/stats", webController.APIStats)
		admin.GET("/api/users", webController.APIUsers)
		admin.POST("/api/scan-library", webController.ScanLibrary)
		admin.GET("/api/scan-progress", webController.GetScanProgress)
		admin.GET("/api/library-stats", webController.GetLibraryStats)
	}

	// Subsonic API endpoints
	rest := router.Group("/rest")
	{
		// System endpoints (both with and without .view suffix)
		// Note: Subsonic spec says ping/getLicense don't require auth, but we enforce it for security
		rest.GET("/ping", subsonicService.AuthMiddleware(), subsonicService.Ping)
		rest.GET("/ping.view", subsonicService.AuthMiddleware(), subsonicService.Ping)
		rest.GET("/getLicense", subsonicService.AuthMiddleware(), subsonicService.GetLicense)
		rest.GET("/getLicense.view", subsonicService.AuthMiddleware(), subsonicService.GetLicense)
		rest.GET("/getMusicFolders", subsonicService.AuthMiddleware(), subsonicService.GetMusicFolders)
		rest.GET("/getMusicFolders.view", subsonicService.AuthMiddleware(), subsonicService.GetMusicFolders)

		// Browsing endpoints (both with and without .view suffix)
		rest.GET("/getIndexes", subsonicService.AuthMiddleware(), subsonicService.GetIndexes)
		rest.GET("/getIndexes.view", subsonicService.AuthMiddleware(), subsonicService.GetIndexes)
		rest.GET("/getMusicDirectory", subsonicService.AuthMiddleware(), subsonicService.GetMusicDirectory)
		rest.GET("/getMusicDirectory.view", subsonicService.AuthMiddleware(), subsonicService.GetMusicDirectory)
		rest.GET("/getGenres", subsonicService.AuthMiddleware(), subsonicService.GetGenres)
		rest.GET("/getGenres.view", subsonicService.AuthMiddleware(), subsonicService.GetGenres)
		rest.GET("/getArtists", subsonicService.AuthMiddleware(), subsonicService.GetArtists)
		rest.GET("/getArtists.view", subsonicService.AuthMiddleware(), subsonicService.GetArtists)
		rest.GET("/getArtist", subsonicService.AuthMiddleware(), subsonicService.GetArtist)
		rest.GET("/getArtist.view", subsonicService.AuthMiddleware(), subsonicService.GetArtist)
		rest.GET("/getArtistInfo2", subsonicService.AuthMiddleware(), subsonicService.GetArtistInfo2)
		rest.GET("/getArtistInfo2.view", subsonicService.AuthMiddleware(), subsonicService.GetArtistInfo2)
		rest.GET("/getAlbum", subsonicService.AuthMiddleware(), subsonicService.GetAlbum)
		rest.GET("/getAlbum.view", subsonicService.AuthMiddleware(), subsonicService.GetAlbum)
		rest.GET("/getSong", subsonicService.AuthMiddleware(), subsonicService.GetSong)
		rest.GET("/getSong.view", subsonicService.AuthMiddleware(), subsonicService.GetSong)

		// Album/song lists (both with and without .view suffix)
		rest.GET("/getAlbumList", subsonicService.AuthMiddleware(), subsonicService.GetAlbumList)
		rest.GET("/getAlbumList.view", subsonicService.AuthMiddleware(), subsonicService.GetAlbumList)
		rest.GET("/getAlbumList2", subsonicService.AuthMiddleware(), subsonicService.GetAlbumList2)
		rest.GET("/getAlbumList2.view", subsonicService.AuthMiddleware(), subsonicService.GetAlbumList2)
		rest.GET("/getRandomSongs", subsonicService.AuthMiddleware(), subsonicService.GetRandomSongs)
		rest.GET("/getRandomSongs.view", subsonicService.AuthMiddleware(), subsonicService.GetRandomSongs)
		rest.GET("/getTopSongs", subsonicService.AuthMiddleware(), subsonicService.GetTopSongs)
		rest.GET("/getTopSongs.view", subsonicService.AuthMiddleware(), subsonicService.GetTopSongs)
		rest.GET("/getSongsByGenre", subsonicService.AuthMiddleware(), subsonicService.GetSongsByGenre)
		rest.GET("/getSongsByGenre.view", subsonicService.AuthMiddleware(), subsonicService.GetSongsByGenre)
		rest.GET("/getSimilarSongs2", subsonicService.AuthMiddleware(), subsonicService.GetSimilarSongs2)
		rest.GET("/getSimilarSongs2.view", subsonicService.AuthMiddleware(), subsonicService.GetSimilarSongs2)
		rest.GET("/getNowPlaying", subsonicService.AuthMiddleware(), subsonicService.GetNowPlaying)
		rest.GET("/getNowPlaying.view", subsonicService.AuthMiddleware(), subsonicService.GetNowPlaying)
		rest.GET("/getStarred", subsonicService.AuthMiddleware(), subsonicService.GetStarred)
		rest.GET("/getStarred.view", subsonicService.AuthMiddleware(), subsonicService.GetStarred)
		rest.GET("/getStarred2", subsonicService.AuthMiddleware(), subsonicService.GetStarred2)
		rest.GET("/getStarred2.view", subsonicService.AuthMiddleware(), subsonicService.GetStarred2)

		// Searching (both with and without .view suffix)
		rest.GET("/search2", subsonicService.AuthMiddleware(), subsonicService.Search2)
		rest.GET("/search2.view", subsonicService.AuthMiddleware(), subsonicService.Search2)
		rest.GET("/search3", subsonicService.AuthMiddleware(), subsonicService.Search3)
		rest.GET("/search3.view", subsonicService.AuthMiddleware(), subsonicService.Search3)
		// rest.GET("/testSong", subsonicService.AuthMiddleware(), subsonicService.TestSong) // TODO: Implement TestSong method

		// Playlists (both with and without .view suffix)
		rest.GET("/getPlaylists", subsonicService.AuthMiddleware(), subsonicService.GetPlaylists)
		rest.GET("/getPlaylists.view", subsonicService.AuthMiddleware(), subsonicService.GetPlaylists)
		rest.GET("/getPlaylist", subsonicService.AuthMiddleware(), subsonicService.GetPlaylist)
		rest.GET("/getPlaylist.view", subsonicService.AuthMiddleware(), subsonicService.GetPlaylist)
		rest.GET("/createPlaylist", subsonicService.AuthMiddleware(), subsonicService.CreatePlaylist)
		rest.GET("/createPlaylist.view", subsonicService.AuthMiddleware(), subsonicService.CreatePlaylist)
		rest.GET("/updatePlaylist", subsonicService.AuthMiddleware(), subsonicService.UpdatePlaylist)
		rest.GET("/updatePlaylist.view", subsonicService.AuthMiddleware(), subsonicService.UpdatePlaylist)
		rest.GET("/deletePlaylist", subsonicService.AuthMiddleware(), subsonicService.DeletePlaylist)
		rest.GET("/deletePlaylist.view", subsonicService.AuthMiddleware(), subsonicService.DeletePlaylist)

		// Media retrieval (both with and without .view suffix)
		rest.GET("/stream", subsonicService.AuthMiddleware(), subsonicService.Stream)
		rest.GET("/stream.view", subsonicService.AuthMiddleware(), subsonicService.Stream)
		rest.GET("/download", subsonicService.AuthMiddleware(), subsonicService.Download)
		rest.GET("/download.view", subsonicService.AuthMiddleware(), subsonicService.Download)
		rest.GET("/getCoverArt", subsonicService.AuthMiddleware(), subsonicService.GetCoverArt)
		rest.GET("/getCoverArt.view", subsonicService.AuthMiddleware(), subsonicService.GetCoverArt)
		rest.GET("/getLyrics", subsonicService.AuthMiddleware(), subsonicService.GetLyrics)
		rest.GET("/getLyrics.view", subsonicService.AuthMiddleware(), subsonicService.GetLyrics)
		rest.GET("/getAvatar", subsonicService.AuthMiddleware(), subsonicService.GetAvatar)
		rest.GET("/getAvatar.view", subsonicService.AuthMiddleware(), subsonicService.GetAvatar)

		// User management (both with and without .view suffix)
		rest.GET("/getUser", subsonicService.AuthMiddleware(), subsonicService.GetUser)
		rest.GET("/getUser.view", subsonicService.AuthMiddleware(), subsonicService.GetUser)
		rest.GET("/getUsers", subsonicService.AuthMiddleware(), subsonicService.GetUsers)
		rest.GET("/getUsers.view", subsonicService.AuthMiddleware(), subsonicService.GetUsers)
		rest.GET("/createUser", subsonicService.AuthMiddleware(), subsonicService.CreateUser)
		rest.GET("/createUser.view", subsonicService.AuthMiddleware(), subsonicService.CreateUser)
		rest.GET("/updateUser", subsonicService.AuthMiddleware(), subsonicService.UpdateUser)
		rest.GET("/updateUser.view", subsonicService.AuthMiddleware(), subsonicService.UpdateUser)
		rest.GET("/deleteUser", subsonicService.AuthMiddleware(), subsonicService.DeleteUser)
		rest.GET("/deleteUser.view", subsonicService.AuthMiddleware(), subsonicService.DeleteUser)
		rest.GET("/changePassword", subsonicService.AuthMiddleware(), subsonicService.ChangePassword)
		rest.GET("/changePassword.view", subsonicService.AuthMiddleware(), subsonicService.ChangePassword)

		// Rating and favorites (both with and without .view suffix)
		rest.GET("/star", subsonicService.AuthMiddleware(), subsonicService.Star)
		rest.GET("/star.view", subsonicService.AuthMiddleware(), subsonicService.Star)
		rest.GET("/unstar", subsonicService.AuthMiddleware(), subsonicService.Unstar)
		rest.GET("/unstar.view", subsonicService.AuthMiddleware(), subsonicService.Unstar)
		rest.GET("/setRating", subsonicService.AuthMiddleware(), subsonicService.SetRating)
		rest.GET("/setRating.view", subsonicService.AuthMiddleware(), subsonicService.SetRating)
		rest.GET("/scrobble", subsonicService.AuthMiddleware(), subsonicService.Scrobble)
		rest.GET("/scrobble.view", subsonicService.AuthMiddleware(), subsonicService.Scrobble)
		rest.GET("/setNowPlaying", subsonicService.AuthMiddleware(), subsonicService.SetNowPlaying)
		rest.GET("/setNowPlaying.view", subsonicService.AuthMiddleware(), subsonicService.SetNowPlaying)
	}

	// Health check endpoint
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "healthy",
			"service": "castafiore-backend",
		})
	})

	// Debug routes endpoint
	router.GET("/debug/routes", func(c *gin.Context) {
		routes := router.Routes()
		c.JSON(http.StatusOK, gin.H{
			"total_routes": len(routes),
			"routes":       routes,
		})
	})

	// API info endpoint
	router.GET("/api", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"service":      "Castafiore Backend",
			"version":      "1.0.0",
			"subsonic_api": "1.16.1",
			"endpoints": gin.H{
				"health":   "/health",
				"subsonic": "/rest/*",
				"admin":    "/admin",
			},
		})
	})
}
