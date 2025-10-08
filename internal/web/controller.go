package web

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"castafiore-backend/internal/auth"
	"castafiore-backend/internal/config"
	"castafiore-backend/internal/library"

	"github.com/gin-gonic/gin"
)

type WebController struct {
	db               *sql.DB
	auth             *auth.Service
	config           *config.Config
	scanner          *library.Scanner
	optimizedScanner *library.OptimizedScanner
}

type DashboardData struct {
	TotalUsers   int
	TotalArtists int
	TotalAlbums  int
	TotalSongs   int
	MusicPath    string
	RecentUsers  []RecentUser
}

type RecentUser struct {
	Username         string
	Email            string
	SubscriptionPlan string
	CreatedAt        string
}

type User struct {
	ID                   int    `json:"id"`
	Username             string `json:"username"`
	Email                string `json:"email"`
	SubscriptionPlan     string `json:"subscription_plan"`
	MaxConcurrentStreams int    `json:"max_concurrent_streams"`
	MaxDownloadsPerDay   int    `json:"max_downloads_per_day"`
	IsAdmin              bool   `json:"is_admin"`
	IsActive             bool   `json:"is_active"`
	CreatedAt            string `json:"created_at"`
}

// Estructuras para la biblioteca de música
type Artist struct {
	ID         int    `json:"id"`
	Name       string `json:"name"`
	AlbumCount int    `json:"album_count"`
	SongCount  int    `json:"song_count"`
}

type Album struct {
	ID         int    `json:"id"`
	Name       string `json:"name"`
	ArtistID   int    `json:"artist_id"`
	ArtistName string `json:"artist_name"`
	Year       int    `json:"year"`
	SongCount  int    `json:"song_count"`
	Duration   int    `json:"duration"`
}

type Song struct {
	ID          int    `json:"id"`
	Title       string `json:"title"`
	ArtistID    int    `json:"artist_id"`
	ArtistName  string `json:"artist_name"`
	AlbumID     int    `json:"album_id"`
	AlbumName   string `json:"album_name"`
	Track       int    `json:"track"`
	Year        int    `json:"year"`
	Genre       string `json:"genre"`
	Duration    int    `json:"duration"`
	Size        int64  `json:"size"`
	Suffix      string `json:"suffix"`
	Path        string `json:"path"`
	BitRate     int    `json:"bit_rate"`
	ContentType string `json:"content_type"`
}

type LibraryData struct {
	Section      string   `json:"section"`
	Artists      []Artist `json:"artists,omitempty"`
	Albums       []Album  `json:"albums,omitempty"`
	Songs        []Song   `json:"songs,omitempty"`
	TotalPages   int      `json:"total_pages"`
	CurrentPage  int      `json:"current_page"`
	PageSize     int      `json:"page_size"`
	TotalRecords int      `json:"total_records"`
	ArtistFilter string   `json:"artist_filter,omitempty"`
}

func NewWebController(db *sql.DB, authService *auth.Service, cfg *config.Config) *WebController {
	// Initialize scanners
	scanner := library.NewScanner(db)
	optimizedScanner := library.NewOptimizedScanner(db)

	controller := &WebController{
		db:               db,
		auth:             authService,
		config:           cfg,
		scanner:          scanner,
		optimizedScanner: optimizedScanner,
	}

	// Cargar el directorio de música persistido si existe
	if persistedPath := controller.loadMusicPathFromFile(); persistedPath != cfg.MusicPath {
		controller.config.MusicPath = persistedPath
		os.Setenv("MUSIC_PATH", persistedPath)
	}

	return controller
}

// Login página
func (w *WebController) LoginForm(c *gin.Context) {
	c.HTML(http.StatusOK, "login.html", gin.H{
		"title": "Iniciar Sesión",
	})
}

// Login POST
func (w *WebController) Login(c *gin.Context) {
	username := c.PostForm("username")
	password := c.PostForm("password")

	if username == "" || password == "" {
		c.HTML(http.StatusBadRequest, "login.html", gin.H{
			"title": "Iniciar Sesión",
			"error": "Username y password son requeridos",
		})
		return
	}

	// Validar credenciales
	user, err := w.auth.ValidateAdminAuth(w.db, username, password)
	if err != nil {
		c.HTML(http.StatusUnauthorized, "login.html", gin.H{
			"title": "Iniciar Sesión",
			"error": "Credenciales inválidas",
		})
		return
	}

	// Verificar que el usuario sea admin
	var isAdmin bool
	err = w.db.QueryRow("SELECT is_admin FROM users WHERE id = $1", user.ID).Scan(&isAdmin)
	if err != nil || !isAdmin {
		c.HTML(http.StatusUnauthorized, "login.html", gin.H{
			"title": "Iniciar Sesión",
			"error": "Acceso denegado: Se requieren permisos de administrador",
		})
		return
	}

	// Generar JWT token
	token, err := w.auth.GenerateJWT(user)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "login.html", gin.H{
			"title": "Iniciar Sesión",
			"error": "Error al generar token de sesión",
		})
		return
	}

	// Establecer cookie de sesión
	c.SetCookie("auth_token", token, 3600*24, "/", "", false, true) // HttpOnly cookie

	// Redirigir al dashboard
	c.Redirect(http.StatusFound, "/admin")
}

// Logout
func (w *WebController) Logout(c *gin.Context) {
	// Eliminar cookie
	c.SetCookie("auth_token", "", -1, "/", "", false, true)
	c.Redirect(http.StatusFound, "/login")
}

// Middleware de autenticación
func (w *WebController) AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Obtener token de cookie
		token, err := c.Cookie("auth_token")
		if err != nil {
			c.Redirect(http.StatusFound, "/login")
			c.Abort()
			return
		}

		// Validar token
		claims, err := w.auth.ValidateJWT(token)
		if err != nil {
			c.SetCookie("auth_token", "", -1, "/", "", false, true) // Eliminar cookie inválida
			c.Redirect(http.StatusFound, "/login")
			c.Abort()
			return
		}

		// Verificar que el usuario sea admin
		var isAdmin bool
		err = w.db.QueryRow("SELECT is_admin FROM users WHERE id = $1", claims.UserID).Scan(&isAdmin)
		if err != nil || !isAdmin {
			c.SetCookie("auth_token", "", -1, "/", "", false, true)
			c.Redirect(http.StatusFound, "/login")
			c.Abort()
			return
		}

		// Agregar información del usuario al contexto
		c.Set("user_id", claims.UserID)
		c.Set("username", claims.Username)
		c.Next()
	}
}

// AuthMiddleware with debug logging
func (w *WebController) AuthMiddlewareDebug() gin.HandlerFunc {
	return func(c *gin.Context) {
		log.Printf("Debug Auth: Processing request for %s", c.Request.URL.Path)

		// Obtener token de cookie
		token, err := c.Cookie("auth_token")
		if err != nil {
			log.Printf("Debug Auth: No auth token found: %v", err)
			c.Redirect(http.StatusFound, "/login")
			c.Abort()
			return
		}

		// Validar token
		claims, err := w.auth.ValidateJWT(token)
		if err != nil {
			log.Printf("Debug Auth: Invalid JWT: %v", err)
			c.SetCookie("auth_token", "", -1, "/", "", false, true)
			c.Redirect(http.StatusFound, "/login")
			c.Abort()
			return
		}

		// Verificar que el usuario sea admin
		var isAdmin bool
		err = w.db.QueryRow("SELECT is_admin FROM users WHERE id = $1", claims.UserID).Scan(&isAdmin)
		if err != nil {
			log.Printf("Debug Auth: Database error checking admin status: %v", err)
			c.SetCookie("auth_token", "", -1, "/", "", false, true)
			c.Redirect(http.StatusFound, "/login")
			c.Abort()
			return
		}

		if !isAdmin {
			log.Printf("Debug Auth: User %s (ID: %d) is not admin", claims.Username, claims.UserID)
			c.SetCookie("auth_token", "", -1, "/", "", false, true)
			c.Redirect(http.StatusFound, "/login")
			c.Abort()
			return
		}

		log.Printf("Debug Auth: User %s (ID: %d) authenticated successfully", claims.Username, claims.UserID)

		// Agregar información del usuario al contexto
		c.Set("user_id", claims.UserID)
		c.Set("username", claims.Username)
		c.Next()
	}
}

// Dashboard principal
func (w *WebController) Dashboard(c *gin.Context) {
	data := w.getDashboardData()
	w.renderPage(c, "Dashboard", "dashboard", gin.H{
		"data": data,
	})
}

// Página de usuarios
func (w *WebController) Users(c *gin.Context) {
	users := w.getAllUsers()
	w.renderPage(c, "Gestión de Usuarios", "users", gin.H{
		"users": users,
	})
}

// Crear usuario (formulario)
func (w *WebController) CreateUserForm(c *gin.Context) {
	w.renderPage(c, "Crear Usuario", "create_user", nil)
}

// Crear usuario (POST)
func (w *WebController) CreateUser(c *gin.Context) {
	username := c.PostForm("username")
	email := c.PostForm("email")
	password := c.PostForm("password")
	subscriptionPlan := c.PostForm("subscription_plan")
	isAdmin := c.PostForm("is_admin") == "on"

	if username == "" || email == "" || password == "" {
		w.renderPage(c, "Crear Usuario", "create_user", gin.H{
			"error": "Todos los campos son obligatorios",
		})
		return
	}

	// Hash password
	hashedPassword, err := auth.HashPassword(password)
	if err != nil {
		w.renderPage(c, "Crear Usuario", "create_user", gin.H{
			"error": "Error al procesar la contraseña",
		})
		return
	}

	// Set limits based on subscription plan
	maxStreams, maxDownloads := w.getPlanLimits(subscriptionPlan)

	// Insert user
	query := `
		INSERT INTO users (username, email, password_hash, subsonic_password, subscription_plan, max_concurrent_streams, max_downloads_per_day, is_admin, is_active)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`

	_, err = w.db.Exec(query, username, email, hashedPassword, password, subscriptionPlan, maxStreams, maxDownloads, isAdmin, true)
	if err != nil {
		w.renderPage(c, "Crear Usuario", "create_user", gin.H{
			"error": "Error al crear el usuario: " + err.Error(),
		})
		return
	}

	c.Redirect(http.StatusFound, "/admin/users")
}

// Página de configuración
func (w *WebController) Settings(c *gin.Context) {
	// Obtener estadísticas del directorio de música
	musicStats := w.getMusicDirectoryStats()

	w.renderPage(c, "Configuración", "settings", gin.H{
		"config":     w.config,
		"musicStats": musicStats,
	})
}

// API endpoint para estadísticas
func (w *WebController) APIStats(c *gin.Context) {
	data := w.getDashboardData()
	c.JSON(http.StatusOK, data)
}

// API endpoint para lista de usuarios
func (w *WebController) APIUsers(c *gin.Context) {
	users := w.getAllUsers()
	c.JSON(http.StatusOK, users)
}

// Explorador de música - Biblioteca
func (w *WebController) MusicBrowser(c *gin.Context) {
	section := c.DefaultQuery("section", "artists")
	pageStr := c.DefaultQuery("page", "1")
	artistFilter := c.Query("artist")

	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		page = 1
	}

	pageSize := 20 // Elementos por página

	var libraryData LibraryData
	libraryData.Section = section
	libraryData.CurrentPage = page
	libraryData.PageSize = pageSize
	libraryData.ArtistFilter = artistFilter

	switch section {
	case "artists":
		libraryData.Artists, libraryData.TotalRecords = w.getArtists(page, pageSize)
	case "albums":
		libraryData.Albums, libraryData.TotalRecords = w.getAlbums(page, pageSize, artistFilter)
	case "songs":
		libraryData.Songs, libraryData.TotalRecords = w.getSongs(page, pageSize, artistFilter)
	default:
		libraryData.Artists, libraryData.TotalRecords = w.getArtists(page, pageSize)
		libraryData.Section = "artists"
	}

	// Calcular total de páginas
	libraryData.TotalPages = (libraryData.TotalRecords + pageSize - 1) / pageSize

	w.renderPage(c, "Biblioteca Musical", "music_browser", gin.H{
		"library": libraryData,
		"path":    w.config.MusicPath,
	})
}

// Actualizar directorio de música
func (w *WebController) UpdateMusicPath(c *gin.Context) {
	newPath := c.PostForm("music_path")

	if newPath == "" {
		musicStats := w.getMusicDirectoryStats()
		w.renderPage(c, "Configuración", "settings", gin.H{
			"config":     w.config,
			"musicStats": musicStats,
			"error":      "El directorio de música no puede estar vacío",
		})
		return
	}

	// Validar que el directorio existe
	if _, err := os.Stat(newPath); os.IsNotExist(err) {
		musicStats := w.getMusicDirectoryStats()
		w.renderPage(c, "Configuración", "settings", gin.H{
			"config":     w.config,
			"musicStats": musicStats,
			"error":      "El directorio especificado no existe: " + newPath,
		})
		return
	}

	// Validar que es un directorio
	fileInfo, err := os.Stat(newPath)
	if err != nil {
		musicStats := w.getMusicDirectoryStats()
		w.renderPage(c, "Configuración", "settings", gin.H{
			"config":     w.config,
			"musicStats": musicStats,
			"error":      "Error al acceder al directorio: " + err.Error(),
		})
		return
	}

	if !fileInfo.IsDir() {
		musicStats := w.getMusicDirectoryStats()
		w.renderPage(c, "Configuración", "settings", gin.H{
			"config":     w.config,
			"musicStats": musicStats,
			"error":      "La ruta especificada no es un directorio: " + newPath,
		})
		return
	}

	// Validar permisos de lectura
	testPath := filepath.Join(newPath, ".test_access")
	testFile, err := os.Create(testPath)
	if err != nil {
		musicStats := w.getMusicDirectoryStats()
		w.renderPage(c, "Configuración", "settings", gin.H{
			"config":     w.config,
			"musicStats": musicStats,
			"error":      "No se tienen permisos de escritura en el directorio: " + newPath,
		})
		return
	}
	testFile.Close()
	os.Remove(testPath) // Limpiar archivo de prueba

	// Actualizar la configuración
	w.config.MusicPath = newPath

	// Persistir el cambio en archivo
	if err := w.saveMusicPathToFile(newPath); err != nil {
		musicStats := w.getMusicDirectoryStats()
		w.renderPage(c, "Configuración", "settings", gin.H{
			"config":     w.config,
			"musicStats": musicStats,
			"error":      "Directorio actualizado pero no se pudo persistir: " + err.Error(),
		})
		return
	}

	// También actualizar la variable de entorno para futuras cargas
	os.Setenv("MUSIC_PATH", newPath)

	// Iniciar escaneo automático de la biblioteca en segundo plano
	go func() {
		scanner := library.NewScanner(w.db)
		if err := scanner.ScanLibrary(newPath); err != nil {
			log.Printf("Error al escanear la biblioteca automáticamente: %v", err)
		} else {
			log.Printf("Escaneo automático de la biblioteca completado con éxito")
		}
	}()

	// Obtener estadísticas actualizadas del nuevo directorio
	musicStats := w.getMusicDirectoryStats()

	w.renderPage(c, "Configuración", "settings", gin.H{
		"config":     w.config,
		"musicStats": musicStats,
		"success":    "Directorio de música actualizado correctamente a: " + newPath + ". Se ha iniciado un escaneo automático de la biblioteca.",
	})
}

// Persistir el directorio de música en un archivo de configuración
func (w *WebController) saveMusicPathToFile(newPath string) error {
	configDir := "config"
	configFile := filepath.Join(configDir, "music_path.txt")

	// Crear directorio si no existe
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return err
	}

	// Escribir el nuevo path al archivo
	return os.WriteFile(configFile, []byte(newPath), 0644)
}

// Cargar el directorio de música desde el archivo de configuración
func (w *WebController) loadMusicPathFromFile() string {
	configFile := filepath.Join("config", "music_path.txt")
	data, err := os.ReadFile(configFile)
	if err != nil {
		return w.config.MusicPath // Retornar el valor por defecto si no existe el archivo
	}
	return string(data)
}

// Helper para renderizar páginas con contexto común
func (w *WebController) renderPage(c *gin.Context, title, template string, data gin.H) {
	username, _ := c.Get("username")
	if data == nil {
		data = gin.H{}
	}
	data["title"] = title
	data["template"] = template
	data["username"] = username
	c.HTML(http.StatusOK, "layout.html", data)
}

// Helpers
func (w *WebController) getDashboardData() DashboardData {
	data := DashboardData{
		MusicPath: w.config.MusicPath,
	}

	// Total users
	w.db.QueryRow("SELECT COUNT(*) FROM users").Scan(&data.TotalUsers)

	// Total artists
	w.db.QueryRow("SELECT COUNT(*) FROM artists").Scan(&data.TotalArtists)

	// Total albums
	w.db.QueryRow("SELECT COUNT(*) FROM albums").Scan(&data.TotalAlbums)

	// Total songs
	w.db.QueryRow("SELECT COUNT(*) FROM songs").Scan(&data.TotalSongs)

	// Recent users
	rows, err := w.db.Query(`
		SELECT username, email, subscription_plan, created_at 
		FROM users 
		ORDER BY created_at DESC 
		LIMIT 5
	`)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var user RecentUser
			rows.Scan(&user.Username, &user.Email, &user.SubscriptionPlan, &user.CreatedAt)
			data.RecentUsers = append(data.RecentUsers, user)
		}
	}

	return data
}

func (w *WebController) getAllUsers() []User {
	var users []User

	rows, err := w.db.Query(`
		SELECT id, username, email, subscription_plan, max_concurrent_streams, 
		       max_downloads_per_day, is_admin, is_active, created_at
		FROM users 
		ORDER BY created_at DESC
	`)
	if err != nil {
		return users
	}
	defer rows.Close()

	for rows.Next() {
		var user User
		rows.Scan(
			&user.ID, &user.Username, &user.Email, &user.SubscriptionPlan,
			&user.MaxConcurrentStreams, &user.MaxDownloadsPerDay,
			&user.IsAdmin, &user.IsActive, &user.CreatedAt,
		)
		users = append(users, user)
	}

	return users
}

func (w *WebController) getPlanLimits(plan string) (int, int) {
	switch plan {
	case "premium":
		return 10, 1000
	case "pro":
		return 5, 100
	default: // free
		return 1, 10
	}
}

func (w *WebController) scanMusicDirectory() ([]string, error) {
	var files []string

	err := filepath.Walk(w.config.MusicPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() {
			ext := filepath.Ext(path)
			if ext == ".mp3" || ext == ".flac" || ext == ".ogg" || ext == ".m4a" {
				relPath, _ := filepath.Rel(w.config.MusicPath, path)
				files = append(files, relPath)
			}
		}
		return nil
	})

	return files, err
}

// Obtener estadísticas del directorio de música
func (w *WebController) getMusicDirectoryStats() map[string]interface{} {
	stats := map[string]interface{}{
		"exists":    false,
		"readable":  false,
		"fileCount": 0,
		"totalSize": float64(0),
		"formats":   make(map[string]int),
		"error":     "",
		"estimated": false,
	}

	// Verificar si el directorio existe
	if _, err := os.Stat(w.config.MusicPath); os.IsNotExist(err) {
		stats["error"] = "El directorio no existe"
		return stats
	}
	stats["exists"] = true

	// Para bibliotecas grandes, usar estimación rápida en lugar de escaneo completo
	return w.getMusicDirectoryStatsOptimized(stats)
}

// getMusicDirectoryStatsOptimized provides fast estimation for large libraries
func (w *WebController) getMusicDirectoryStatsOptimized(stats map[string]interface{}) map[string]interface{} {
	var totalBytes int64
	fileCount := 0
	maxFiles := 1000 // Solo escanear los primeros 1000 archivos para estimación
	scannedFiles := 0

	// Contar archivos con límite para evitar timeouts
	err := filepath.Walk(w.config.MusicPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() {
			ext := strings.ToLower(filepath.Ext(path))
			switch ext {
			case ".mp3", ".flac", ".ogg", ".m4a", ".wav", ".aac":
				fileCount++
				totalBytes += info.Size()
				scannedFiles++

				formats := stats["formats"].(map[string]int)
				formats[ext] = formats[ext] + 1

				// Limitar escaneo para evitar timeouts
				if scannedFiles >= maxFiles {
					stats["estimated"] = true
					return fmt.Errorf("limit_reached") // Signal to stop scanning
				}
			}
		}
		return nil
	})

	// Si se alcanzó el límite, hacer estimación
	if err != nil && err.Error() == "limit_reached" {
		// Estimar total basado en la muestra
		estimationFactor := float64(10) // Estimación conservadora
		stats["fileCount"] = int(float64(fileCount) * estimationFactor)
		stats["totalSize"] = float64(totalBytes) * estimationFactor / (1024 * 1024)
		stats["estimated"] = true
		log.Printf("Large library detected: estimated %d files based on %d samples",
			stats["fileCount"], fileCount)
	} else if err != nil {
		stats["error"] = "Error al escanear directorio: " + err.Error()
		return stats
	} else {
		stats["fileCount"] = fileCount
		stats["totalSize"] = float64(totalBytes) / (1024 * 1024)
		stats["estimated"] = false
	}

	stats["readable"] = true
	return stats
}

// Métodos para la biblioteca de música con paginación (basados en sistema de archivos)

// Estructuras para organizar archivos de música
type FileArtist struct {
	Name      string
	Albums    map[string]*FileAlbum
	SongCount int
}

type FileAlbum struct {
	Name     string
	Artist   string
	Year     int
	Songs    []FileSong
	Duration int
}

type FileSong struct {
	Title    string
	Artist   string
	Album    string
	Track    int
	Year     int
	Duration int
	Size     int64
	Format   string
	Path     string
}

// Escanear y organizar archivos de música
func (w *WebController) scanAndOrganizeMusic() map[string]*FileArtist {
	artists := make(map[string]*FileArtist)

	filepath.Walk(w.config.MusicPath, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}

		ext := strings.ToLower(filepath.Ext(path))
		if ext != ".mp3" && ext != ".flac" && ext != ".ogg" && ext != ".m4a" && ext != ".wav" {
			return nil
		}

		// Extraer información del path y nombre del archivo
		relPath, _ := filepath.Rel(w.config.MusicPath, path)
		parts := strings.Split(relPath, string(filepath.Separator))

		var artistName, albumName, songTitle string

		if len(parts) >= 3 {
			// Estructura: Artist/Album/Song.ext
			artistName = parts[0]
			albumName = parts[1]
			songTitle = strings.TrimSuffix(parts[2], ext)
		} else if len(parts) == 2 {
			// Estructura: Artist/Song.ext
			artistName = parts[0]
			albumName = "Unknown Album"
			songTitle = strings.TrimSuffix(parts[1], ext)
		} else {
			// Estructura: Song.ext
			artistName = "Unknown Artist"
			albumName = "Unknown Album"
			songTitle = strings.TrimSuffix(parts[0], ext)
		}

		// Crear o obtener artista
		if artists[artistName] == nil {
			artists[artistName] = &FileArtist{
				Name:   artistName,
				Albums: make(map[string]*FileAlbum),
			}
		}

		// Crear o obtener álbum
		if artists[artistName].Albums[albumName] == nil {
			artists[artistName].Albums[albumName] = &FileAlbum{
				Name:   albumName,
				Artist: artistName,
				Songs:  []FileSong{},
			}
		}

		// Crear canción
		song := FileSong{
			Title:  songTitle,
			Artist: artistName,
			Album:  albumName,
			Size:   info.Size(),
			Format: ext,
			Path:   relPath,
		}

		// Agregar canción al álbum
		artists[artistName].Albums[albumName].Songs = append(artists[artistName].Albums[albumName].Songs, song)
		artists[artistName].SongCount++

		return nil
	})

	return artists
}

// Obtener artistas con paginación (desde base de datos)
func (w *WebController) getArtists(page, pageSize int) ([]Artist, int) {
	var artists []Artist

	// Contar total de artistas
	var totalRecords int
	err := w.db.QueryRow("SELECT COUNT(*) FROM artists").Scan(&totalRecords)
	if err != nil {
		log.Printf("Error counting artists: %v", err)
		return artists, 0
	}

	// Calcular offset
	offset := (page - 1) * pageSize

	// Obtener artistas con paginación y estadísticas
	query := `
		SELECT 
			a.id, 
			a.name,
			COUNT(DISTINCT al.id) as album_count,
			COUNT(s.id) as song_count
		FROM artists a
		LEFT JOIN albums al ON a.id = al.artist_id
		LEFT JOIN songs s ON a.id = s.artist_id
		GROUP BY a.id, a.name
		ORDER BY a.name
		LIMIT $1 OFFSET $2
	`

	rows, err := w.db.Query(query, pageSize, offset)
	if err != nil {
		log.Printf("Error getting artists: %v", err)
		return artists, totalRecords
	}
	defer rows.Close()

	for rows.Next() {
		var artist Artist
		err := rows.Scan(&artist.ID, &artist.Name, &artist.AlbumCount, &artist.SongCount)
		if err != nil {
			log.Printf("Error scanning artist: %v", err)
			continue
		}
		artists = append(artists, artist)
	}

	return artists, totalRecords
}

// Obtener álbumes con paginación (desde base de datos)
func (w *WebController) getAlbums(page, pageSize int, artistFilter string) ([]Album, int) {
	var albums []Album

	// Construir query base para contar
	countQuery := "SELECT COUNT(*) FROM albums al JOIN artists ar ON al.artist_id = ar.id"

	// Construir query base para datos
	dataQuery := `
		SELECT 
			al.id, 
			al.name, 
			ar.id as artist_id,
			ar.name as artist_name, 
			al.year,
			COUNT(s.id) as song_count,
			COALESCE(SUM(s.duration), 0) as total_duration
		FROM albums al 
		JOIN artists ar ON al.artist_id = ar.id
		LEFT JOIN songs s ON al.id = s.album_id
	`

	var args []interface{}

	// Aplicar filtro de artista si existe
	if artistFilter != "" {
		countQuery += " WHERE ar.name = $1"
		dataQuery += " WHERE ar.name = $1"
		args = append(args, artistFilter)
	}

	// Contar total de álbumes
	var totalRecords int
	err := w.db.QueryRow(countQuery, args...).Scan(&totalRecords)
	if err != nil {
		log.Printf("Error counting albums: %v", err)
		return albums, 0
	}

	// Completar query de datos con paginación
	offset := (page - 1) * pageSize
	dataQuery += ` 
		GROUP BY al.id, al.name, ar.id, ar.name, al.year
		ORDER BY ar.name, al.name`

	// Agregar LIMIT y OFFSET con placeholders correctos
	if artistFilter != "" {
		dataQuery += ` LIMIT $2 OFFSET $3`
		args = append(args, pageSize, offset)
	} else {
		dataQuery += ` LIMIT $1 OFFSET $2`
		args = append(args, pageSize, offset)
	}

	log.Printf("getAlbums query: %s", dataQuery)
	log.Printf("getAlbums args: %v", args)

	rows, err := w.db.Query(dataQuery, args...)
	if err != nil {
		log.Printf("Error getting albums: %v", err)
		return albums, totalRecords
	}
	defer rows.Close()

	for rows.Next() {
		var album Album
		var duration int64 // Use int64 for SUM result from PostgreSQL
		err := rows.Scan(
			&album.ID,
			&album.Name,
			&album.ArtistID,
			&album.ArtistName,
			&album.Year,
			&album.SongCount,
			&duration,
		)
		if err != nil {
			log.Printf("Error scanning album: %v", err)
			continue
		}
		album.Duration = int(duration) // Convert int64 to int
		albums = append(albums, album)
	}

	return albums, totalRecords
}

// Obtener canciones con paginación (desde base de datos)
func (w *WebController) getSongs(page, pageSize int, artistFilter string) ([]Song, int) {
	var songs []Song

	// Construir query base para contar
	countQuery := `
		SELECT COUNT(*) 
		FROM songs s 
		JOIN artists ar ON s.artist_id = ar.id 
		JOIN albums al ON s.album_id = al.id
	`

	// Construir query base para datos
	dataQuery := `
		SELECT 
			s.id, 
			s.title, 
			s.artist_id,
			ar.name as artist_name,
			s.album_id, 
			al.name as album_name,
			s.track_number,
			COALESCE(al.year, 0) as year,
			s.duration,
			s.file_size,
			s.format,
			s.file_path,
			s.bitrate
		FROM songs s 
		JOIN artists ar ON s.artist_id = ar.id
		JOIN albums al ON s.album_id = al.id
	`

	var args []interface{}

	// Aplicar filtro de artista si existe
	if artistFilter != "" {
		countQuery += " WHERE ar.name = $1"
		dataQuery += " WHERE ar.name = $1"
		args = append(args, artistFilter)
	}

	// Contar total de canciones
	var totalRecords int
	err := w.db.QueryRow(countQuery, args...).Scan(&totalRecords)
	if err != nil {
		log.Printf("Error counting songs: %v", err)
		return songs, 0
	}

	// Completar query de datos con paginación
	offset := (page - 1) * pageSize
	dataQuery += ` 
		ORDER BY ar.name, al.name, s.track_number, s.title`

	// Agregar LIMIT y OFFSET con placeholders correctos
	if artistFilter != "" {
		dataQuery += ` LIMIT $2 OFFSET $3`
		args = append(args, pageSize, offset)
	} else {
		dataQuery += ` LIMIT $1 OFFSET $2`
		args = append(args, pageSize, offset)
	}

	rows, err := w.db.Query(dataQuery, args...)
	if err != nil {
		log.Printf("Error getting songs: %v", err)
		return songs, totalRecords
	}
	defer rows.Close()

	for rows.Next() {
		var song Song
		var duration int // duration in seconds
		var format string

		err := rows.Scan(
			&song.ID,
			&song.Title,
			&song.ArtistID,
			&song.ArtistName,
			&song.AlbumID,
			&song.AlbumName,
			&song.Track,
			&song.Year,
			&duration,
			&song.Size,
			&format,
			&song.Path,
			&song.BitRate,
		)
		if err != nil {
			log.Printf("Error scanning song: %v", err)
			continue
		}

		song.Duration = duration
		song.Suffix = strings.TrimPrefix(strings.ToLower(format), ".")
		song.ContentType = getContentType("." + song.Suffix)

		songs = append(songs, song)
	}

	return songs, totalRecords
}

// Helper para obtener content type
func getContentType(ext string) string {
	switch strings.ToLower(ext) {
	case ".mp3":
		return "audio/mpeg"
	case ".flac":
		return "audio/flac"
	case ".ogg":
		return "audio/ogg"
	case ".m4a":
		return "audio/mp4"
	case ".wav":
		return "audio/wav"
	default:
		return "audio/mpeg"
	}
}

// Estructura para resultados de búsqueda
type SearchResults struct {
	Query        string   `json:"query"`
	Artists      []Artist `json:"artists"`
	Albums       []Album  `json:"albums"`
	Songs        []Song   `json:"songs"`
	TotalResults int      `json:"total_results"`
}

// Búsqueda en la biblioteca de música
func (w *WebController) SearchMusic(c *gin.Context) {
	query := strings.TrimSpace(c.Query("q"))
	section := c.DefaultQuery("section", "all")
	pageStr := c.DefaultQuery("page", "1")

	if query == "" {
		// Si no hay query, redirigir a la biblioteca normal
		c.Redirect(http.StatusFound, "/admin/music")
		return
	}

	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		page = 1
	}

	pageSize := 20

	// Realizar búsqueda
	results := w.performMusicSearch(query, section, page, pageSize)

	// Preparar datos para el template
	var libraryData LibraryData
	libraryData.Section = section
	libraryData.CurrentPage = page
	libraryData.PageSize = pageSize

	switch section {
	case "artists":
		libraryData.Artists = results.Artists
		libraryData.TotalRecords = len(results.Artists)
	case "albums":
		libraryData.Albums = results.Albums
		libraryData.TotalRecords = len(results.Albums)
	case "songs":
		libraryData.Songs = results.Songs
		libraryData.TotalRecords = len(results.Songs)
	default: // "all"
		libraryData.Artists = results.Artists
		libraryData.Albums = results.Albums
		libraryData.Songs = results.Songs
		libraryData.TotalRecords = results.TotalResults
		libraryData.Section = "all"
	}

	libraryData.TotalPages = (libraryData.TotalRecords + pageSize - 1) / pageSize

	w.renderPage(c, "Búsqueda: "+query, "music_browser", gin.H{
		"library":       libraryData,
		"path":          w.config.MusicPath,
		"searchQuery":   query,
		"searchResults": results,
	})
}

// Realizar búsqueda en la música (optimizada con base de datos)
func (w *WebController) performMusicSearch(query, section string, page, pageSize int) SearchResults {
	results := SearchResults{
		Query:   query,
		Artists: []Artist{},
		Albums:  []Album{},
		Songs:   []Song{},
	}

	searchTerm := "%" + strings.ToLower(query) + "%"

	// Buscar artistas
	if section == "all" || section == "artists" {
		artistQuery := `
			SELECT 
				a.id, 
				a.name,
				COUNT(DISTINCT al.id) as album_count,
				COUNT(s.id) as song_count
			FROM artists a
			LEFT JOIN albums al ON a.id = al.artist_id
			LEFT JOIN songs s ON a.id = s.artist_id
			WHERE LOWER(a.name) LIKE $1
			GROUP BY a.id, a.name
			ORDER BY a.name
		`

		limit := pageSize
		if section == "all" {
			limit = 10 // Limit for "all" search
		}
		artistQuery += fmt.Sprintf(" LIMIT %d", limit)

		if section == "artists" {
			offset := (page - 1) * pageSize
			artistQuery += fmt.Sprintf(" OFFSET %d", offset)
		}

		rows, err := w.db.Query(artistQuery, searchTerm)
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var artist Artist
				err := rows.Scan(&artist.ID, &artist.Name, &artist.AlbumCount, &artist.SongCount)
				if err == nil {
					results.Artists = append(results.Artists, artist)
				}
			}
		}
	}

	// Buscar álbumes
	if section == "all" || section == "albums" {
		albumQuery := `
			SELECT 
				al.id, 
				al.name, 
				ar.id as artist_id,
				ar.name as artist_name, 
				al.year,
				COUNT(s.id) as song_count,
				COALESCE(SUM(s.duration), 0) as total_duration
			FROM albums al 
			JOIN artists ar ON al.artist_id = ar.id
			LEFT JOIN songs s ON al.id = s.album_id
			WHERE LOWER(al.name) LIKE $1 OR LOWER(ar.name) LIKE $1
			GROUP BY al.id, al.name, ar.id, ar.name, al.year
			ORDER BY ar.name, al.name
		`

		limit := pageSize
		if section == "all" {
			limit = 10
		}
		albumQuery += fmt.Sprintf(" LIMIT %d", limit)

		if section == "albums" {
			offset := (page - 1) * pageSize
			albumQuery += fmt.Sprintf(" OFFSET %d", offset)
		}

		rows, err := w.db.Query(albumQuery, searchTerm)
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var album Album
				err := rows.Scan(
					&album.ID, &album.Name, &album.ArtistID, &album.ArtistName,
					&album.Year, &album.SongCount, &album.Duration,
				)
				if err == nil {
					results.Albums = append(results.Albums, album)
				}
			}
		}
	}

	// Buscar canciones
	if section == "all" || section == "songs" {
		songQuery := `
			SELECT 
				s.id, s.title, s.artist_id, ar.name as artist_name,
				s.album_id, al.name as album_name, s.track_number,
				COALESCE(al.year, 0) as year, s.duration, s.file_size,
				s.format, s.file_path, s.bitrate
			FROM songs s 
			JOIN artists ar ON s.artist_id = ar.id
			JOIN albums al ON s.album_id = al.id
			WHERE LOWER(s.title) LIKE $1 OR LOWER(ar.name) LIKE $1 OR LOWER(al.name) LIKE $1
			ORDER BY ar.name, al.name, s.track_number, s.title
		`

		limit := pageSize
		if section == "all" {
			limit = 10
		}
		songQuery += fmt.Sprintf(" LIMIT %d", limit)

		if section == "songs" {
			offset := (page - 1) * pageSize
			songQuery += fmt.Sprintf(" OFFSET %d", offset)
		}

		rows, err := w.db.Query(songQuery, searchTerm)
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var song Song
				var duration int
				var format string

				err := rows.Scan(
					&song.ID, &song.Title, &song.ArtistID, &song.ArtistName,
					&song.AlbumID, &song.AlbumName, &song.Track, &song.Year,
					&duration, &song.Size, &format, &song.Path, &song.BitRate,
				)
				if err == nil {
					song.Duration = duration
					song.Suffix = strings.TrimPrefix(strings.ToLower(format), ".")
					song.ContentType = getContentType("." + song.Suffix)
					results.Songs = append(results.Songs, song)
				}
			}
		}
	}

	results.TotalResults = len(results.Artists) + len(results.Albums) + len(results.Songs)
	return results
}

// ScanLibrary handles library scanning requests
func (wc *WebController) ScanLibrary(c *gin.Context) {
	// Get scan mode from query parameter (fast, full, incremental)
	scanMode := c.DefaultQuery("mode", "fast")

	// Get music path from config
	musicPath, err := wc.getMusicPath()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get music path: " + err.Error(),
		})
		return
	}

	// Check if music path exists
	if _, err := os.Stat(musicPath); os.IsNotExist(err) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Music path does not exist: " + musicPath,
		})
		return
	}

	// Check if a scan is already in progress
	if wc.scanner.IsScanning || wc.optimizedScanner.IsScanning {
		progress := wc.scanner.GetScanProgress()
		if wc.optimizedScanner.IsScanning {
			progress = wc.optimizedScanner.GetScanProgress()
		}
		c.JSON(http.StatusConflict, gin.H{
			"error":    "A library scan is already in progress",
			"progress": progress,
		})
		return
	}

	// Choose scanner based on mode and library size
	useOptimizedScanner := true

	// Quick file count to determine which scanner to use
	if scanMode == "full" {
		useOptimizedScanner = false // Force regular scanner for full scan
	} else {
		// Count files to decide
		fileCount := wc.quickFileCount(musicPath)
		log.Printf("Detected %d audio files in library", fileCount)

		if fileCount > 10000 || scanMode == "fast" {
			useOptimizedScanner = true
			wc.optimizedScanner.IncrementalMode = (scanMode == "incremental")
		} else {
			useOptimizedScanner = false
		}
	}

	// Start scanning in a goroutine to avoid blocking the request
	go func() {
		var err error
		if useOptimizedScanner {
			log.Printf("Using optimized scanner (mode: %s)", scanMode)
			err = wc.optimizedScanner.ScanLibraryOptimized(musicPath)
		} else {
			log.Printf("Using regular scanner (mode: %s)", scanMode)
			err = wc.scanner.ScanLibrary(musicPath)
		}

		if err != nil {
			log.Printf("Library scan error: %v", err)
		}
	}()

	response := gin.H{
		"message": "Library scan started",
		"path":    musicPath,
		"mode":    scanMode,
		"scanner": "regular",
	}

	if useOptimizedScanner {
		response["scanner"] = "optimized"
	}

	c.JSON(http.StatusOK, response)
}

// GetScanProgress returns the current progress of the library scan
func (wc *WebController) GetScanProgress(c *gin.Context) {
	var progress map[string]interface{}

	if wc.optimizedScanner.IsScanning {
		progress = wc.optimizedScanner.GetScanProgress()
		progress["scanner_type"] = "optimized"
	} else if wc.scanner.IsScanning {
		progress = wc.scanner.GetScanProgress()
		progress["scanner_type"] = "regular"
	} else {
		progress = map[string]interface{}{
			"is_scanning":      false,
			"total_files":      0,
			"processed_files":  0,
			"percent_complete": 0.0,
			"scanner_type":     "none",
		}
	}

	c.JSON(http.StatusOK, progress)
}

// GetLibraryStats returns current library statistics
func (wc *WebController) GetLibraryStats(c *gin.Context) {
	// Use optimized scanner for stats if available
	var stats map[string]int
	var err error

	if wc.optimizedScanner != nil {
		stats, err = wc.optimizedScanner.GetScanStats()
	} else {
		scanner := library.NewScanner(wc.db)
		stats, err = scanner.GetScanStats()
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get library stats: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// getMusicPath reads the music path from the config file
func (wc *WebController) getMusicPath() (string, error) {
	configFile := "config/music_path.txt"
	content, err := os.ReadFile(configFile)
	if err != nil {
		return "", err
	}

	musicPath := strings.TrimSpace(string(content))
	if musicPath == "" {
		return "", fmt.Errorf("music path is empty in config file")
	}

	return musicPath, nil
}

// quickFileCount quickly estimates the number of audio files
func (wc *WebController) quickFileCount(musicPath string) int {
	count := 0
	maxSample := 500 // Only scan first 500 files for quick estimation
	totalFiles := 0

	filepath.Walk(musicPath, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}

		totalFiles++

		ext := strings.ToLower(filepath.Ext(path))
		if ext == ".mp3" || ext == ".flac" || ext == ".ogg" || ext == ".m4a" || ext == ".wav" {
			count++
		}

		// Stop after scanning a small sample for estimation
		if totalFiles >= maxSample {
			return fmt.Errorf("sample_complete") // Stop walking
		}

		return nil
	})

	// If we hit the sample limit, estimate total based on the ratio
	if totalFiles >= maxSample && count > 0 {
		// Simple estimation: if X% of first 500 files are audio, apply same ratio
		ratio := float64(count) / float64(totalFiles)
		estimated := int(ratio * 100000) // Conservative estimate base
		log.Printf("Quick estimation: %d audio files found in sample of %d, estimating %d total",
			count, totalFiles, estimated)
		return estimated
	}

	return count
}

// User management functions

// EditUserForm displays the user edit form
func (w *WebController) EditUserForm(c *gin.Context) {
	userID := c.Param("id")
	log.Printf("DEBUG: EditUserForm called for user ID: %s", userID)

	user, err := w.getUserByID(userID)
	if err != nil {
		log.Printf("DEBUG: Error getting user by ID %s: %v", userID, err)
		c.JSON(http.StatusNotFound, gin.H{"error": "Usuario no encontrado"})
		return
	}

	log.Printf("DEBUG: User found: %s, rendering edit form", user.Username)
	w.renderPage(c, "Editar Usuario", "edit_user", gin.H{
		"user": user,
	})
}

// EditUser processes user edit form submission
func (w *WebController) EditUser(c *gin.Context) {
	userID := c.Param("id")

	// Check if user exists
	existingUser, err := w.getUserByID(userID)
	if err != nil {
		w.renderPage(c, "Editar Usuario", "edit_user", gin.H{
			"error": "Usuario no encontrado",
		})
		return
	}

	// Get current user from session for security validations
	currentUsername, exists := c.Get("username")
	if !exists {
		c.Redirect(http.StatusFound, "/login")
		return
	}

	username := c.PostForm("username")
	email := c.PostForm("email")
	password := c.PostForm("password")
	subscriptionPlan := c.PostForm("subscription_plan")
	maxStreams := c.PostForm("max_concurrent_streams")
	maxDownloads := c.PostForm("max_downloads_per_day")
	isAdmin := c.PostForm("is_admin") == "on"
	isActive := c.PostForm("is_active") == "on"

	if username == "" || email == "" {
		w.renderPage(c, "Editar Usuario", "edit_user", gin.H{
			"error": "El nombre de usuario y email son obligatorios",
			"user":  existingUser,
		})
		return
	}

	// Prevent deactivating or removing admin privileges from last admin
	if existingUser.IsAdmin && (!isAdmin || !isActive) {
		adminCount, err := w.getAdminCount()
		if err == nil && adminCount <= 1 {
			w.renderPage(c, "Editar Usuario", "edit_user", gin.H{
				"error": "No se puede desactivar o remover privilegios del último administrador",
				"user":  existingUser,
			})
			return
		}
	}

	// Prevent current user from deactivating themselves
	if existingUser.Username == currentUsername && !isActive {
		w.renderPage(c, "Editar Usuario", "edit_user", gin.H{
			"error": "No puedes desactivar tu propia cuenta",
			"user":  existingUser,
		})
		return
	}

	// Convert strings to integers
	maxStreamsInt, err := strconv.Atoi(maxStreams)
	if err != nil {
		maxStreamsInt = 1
	}

	maxDownloadsInt, err := strconv.Atoi(maxDownloads)
	if err != nil {
		maxDownloadsInt = 10
	}

	// Prepare update query
	var query string
	var args []interface{}

	if password != "" {
		// Include password update
		hashedPassword, err := auth.HashPassword(password)
		if err != nil {
			w.renderPage(c, "Editar Usuario", "edit_user", gin.H{
				"error": "Error al procesar la contraseña",
				"user":  existingUser,
			})
			return
		}

		query = `
			UPDATE users 
			SET username = $1, email = $2, password_hash = $3, subsonic_password = $4, subscription_plan = $5, 
			    max_concurrent_streams = $6, max_downloads_per_day = $7, is_admin = $8, is_active = $9
			WHERE id = $10
		`
		args = []interface{}{username, email, hashedPassword, password, subscriptionPlan,
			maxStreamsInt, maxDownloadsInt, isAdmin, isActive, userID}
	} else {
		// No password change
		query = `
			UPDATE users 
			SET username = $1, email = $2, subscription_plan = $3, 
			    max_concurrent_streams = $4, max_downloads_per_day = $5, is_admin = $6, is_active = $7
			WHERE id = $8
		`
		args = []interface{}{username, email, subscriptionPlan,
			maxStreamsInt, maxDownloadsInt, isAdmin, isActive, userID}
	}

	_, err = w.db.Exec(query, args...)
	if err != nil {
		w.renderPage(c, "Editar Usuario", "edit_user", gin.H{
			"error": "Error al actualizar el usuario: " + err.Error(),
			"user":  existingUser,
		})
		return
	}

	c.Redirect(http.StatusFound, "/admin/users")
}

// DeleteUser handles user deletion
func (w *WebController) DeleteUser(c *gin.Context) {
	userID := c.Param("id")
	log.Printf("DEBUG: DeleteUser called for user ID: %s", userID)

	// Check if user exists
	userToDelete, err := w.getUserByID(userID)
	if err != nil {
		log.Printf("DEBUG: Error getting user to delete ID %s: %v", userID, err)
		c.JSON(http.StatusNotFound, gin.H{"error": "Usuario no encontrado"})
		return
	}

	log.Printf("DEBUG: User to delete found: %s", userToDelete.Username)

	// Get current user from session
	currentUsername, exists := c.Get("username")
	if !exists {
		c.Redirect(http.StatusFound, "/login")
		return
	}

	// Prevent self-deletion
	if userToDelete.Username == currentUsername {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No puedes eliminar tu propia cuenta"})
		return
	}

	// Prevent deleting last admin
	if userToDelete.IsAdmin {
		adminCount, err := w.getAdminCount()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al verificar administradores"})
			return
		}

		if adminCount <= 1 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "No se puede eliminar el último administrador"})
			return
		}
	}

	// Delete user
	_, err = w.db.Exec("DELETE FROM users WHERE id = $1", userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al eliminar el usuario"})
		return
	}

	c.Redirect(http.StatusFound, "/admin/users")
}

// Helper function to get user by ID
func (w *WebController) getUserByID(userID string) (User, error) {
	var user User

	query := `
		SELECT id, username, email, subscription_plan, max_concurrent_streams, 
		       max_downloads_per_day, is_admin, is_active, created_at
		FROM users 
		WHERE id = $1
	`

	err := w.db.QueryRow(query, userID).Scan(
		&user.ID, &user.Username, &user.Email, &user.SubscriptionPlan,
		&user.MaxConcurrentStreams, &user.MaxDownloadsPerDay,
		&user.IsAdmin, &user.IsActive, &user.CreatedAt,
	)

	return user, err
}

// Helper function to count admin users
func (w *WebController) getAdminCount() (int, error) {
	var count int
	err := w.db.QueryRow("SELECT COUNT(*) FROM users WHERE is_admin = true").Scan(&count)
	return count, err
}

// GetAllUsersForTest returns all users for testing purposes
func (w *WebController) GetAllUsersForTest() []User {
	return w.getAllUsers()
}
