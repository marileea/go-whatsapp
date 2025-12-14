package main

import (
    "context"
    "encoding/json"
    "fmt"
    "log"
    "net/http"
    "os"
    "os/signal"
    "sync"
    "syscall"
    "time"

    "github.com/go-chi/chi/v5"
    "github.com/go-chi/chi/v5/middleware"

    "github.com/godeirect/go-whatsapp-multidevice-api/services/telegram_auth/config"
    httpauth "github.com/godeirect/go-whatsapp-multidevice-api/services/telegram_auth/gen/http/auth/server"
    httpdeeplink "github.com/godeirect/go-whatsapp-multidevice-api/services/telegram_auth/gen/http/deep_link/server"
    httpwebhook "github.com/godeirect/go-whatsapp-multidevice-api/services/telegram_auth/gen/http/webhook/server"
    authservice "github.com/godeirect/go-whatsapp-multidevice-api/services/telegram_auth/internal/auth"
    deeplinkservice "github.com/godeirect/go-whatsapp-multidevice-api/services/telegram_auth/internal/deeplink"
    "github.com/godeirect/go-whatsapp-multidevice-api/services/telegram_auth/internal/logger"
    webhookservice "github.com/godeirect/go-whatsapp-multidevice-api/services/telegram_auth/internal/webhook"
)

func main() {
    // Load configuration
    cfg, err := config.Load()
    if err != nil {
        log.Fatalf("Failed to load configuration: %v", err)
    }

    // Initialize logger
    logger := logger.New(cfg.LogLevel, "text")
    logger.Info(context.Background(), "Starting Telegram Authentication Service", 
        "port", cfg.Port, 
        "env", cfg.Env,
        "log_level", cfg.LogLevel)

    // Initialize services
    authService := authservice.New(cfg.JWTSecret, cfg.TokenTTL)
    deepLinkService := deeplinkservice.New(cfg.DeepLinkBaseURL)
    webhookService := webhookservice.New(cfg.TelegramBotToken, cfg.WebhookURL)

    // Initialize HTTP controllers
    authController := httpauth.NewController(authService)
    deepLinkController := httpdeeplink.NewController(deepLinkService)
    webhookController := httpwebhook.NewController(webhookService)

    // Initialize HTTP server
    server := &http.Server{
        Addr:         fmt.Sprintf(":%d", cfg.Port),
        Handler:      newRouter(authController, deepLinkController, webhookController, logger),
        ReadTimeout:  30 * time.Second,
        WriteTimeout: 30 * time.Second,
        IdleTimeout:  60 * time.Second,
    }

    // Graceful shutdown handling
    quit := make(chan os.Signal, 1)
    signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

    var wg sync.WaitGroup
    wg.Add(1)

    go func() {
        <-quit
        logger.Info(context.Background(), "Shutting down server...")

        ctx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTime)
        defer cancel()

        if err := server.Shutdown(ctx); err != nil {
            logger.Error(context.Background(), "Server forced to shutdown", 
                "error", err)
        }

        wg.Done()
    }()

    // Start server
    logger.Info(context.Background(), "Server is ready", 
        "port", cfg.Port,
        "address", fmt.Sprintf("http://localhost:%d", cfg.Port))

    if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
        logger.Error(context.Background(), "Failed to start server", 
            "error", err)
        os.Exit(1)
    }

    wg.Wait()
    logger.Info(context.Background(), "Server exited")
}

// Router returns a new HTTP router with middleware and routes
func newRouter(
    authController *httpauth.Controller,
    deepLinkController *httpdeeplink.Controller,
    webhookController *httpwebhook.Controller,
    logger *logger.Logger,
) http.Handler {
    r := chi.NewRouter()

    // Middleware
    r.Use(middleware.RequestID)
    r.Use(middleware.RealIP)
    r.Use(middleware.Logger)
    r.Use(middleware.Recoverer)
    r.Use(middleware.Timeout(30 * time.Second))

    // CORS middleware
    r.Use(func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            w.Header().Set("Access-Control-Allow-Origin", "*")
            w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
            w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

            if r.Method == "OPTIONS" {
                w.WriteHeader(http.StatusOK)
                return
            }

            next.ServeHTTP(w, r)
        })
    })

    // Health check endpoint
    r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(map[string]string{
            "status":    "ok",
            "service":   "telegram-auth",
            "timestamp": time.Now().Format(time.RFC3339),
        })
    })

    // Auth routes
    r.Route("/auth", func(r chi.Router) {
        r.Post("/validate", handleAuthValidate(authController, logger))
        r.Post("/refresh", handleAuthRefresh(authController, logger))
        r.Post("/logout", handleAuthLogout(authController, logger))
    })

    // Deep link routes
    r.Route("/deeplink", func(r chi.Router) {
        r.Post("/generate", handleGenerateLink(deepLinkController, logger))
        r.Post("/validate", handleValidateLink(deepLinkController, logger))
        r.Get("/validate/{link}", handleValidateLinkGet(deepLinkController, logger))
    })

    // Webhook routes
    r.Route("/webhook", func(r chi.Router) {
        r.Post("/telegram", handleTelegramUpdate(webhookController, logger))
        r.Post("/verify", handleVerifyWebhook(webhookController, logger))
    })

    // 404 handler
    r.NotFound(func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(http.StatusNotFound)
        json.NewEncoder(w).Encode(map[string]string{
            "error": "Not Found",
            "path":  r.URL.Path,
        })
    })

    // 405 handler
    r.MethodNotAllowed(func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(http.StatusMethodNotAllowed)
        json.NewEncoder(w).Encode(map[string]string{
            "error": "Method Not Allowed",
            "path":  r.URL.Path,
        })
    })

    return r
}

// HTTP handlers

func handleAuthValidate(authController *httpauth.Controller, logger *logger.Logger) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        ctx := r.Context()

        var req httpauth.ValidateTokenRequest
        if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
            logger.Warn(ctx, "Failed to decode validate token request", "error", err)
            http.Error(w, "Invalid request body", http.StatusBadRequest)
            return
        }

        result, err := authController.ValidateToken(ctx, &req)
        if err != nil {
            logger.Error(ctx, "Validate token failed", "error", err)
            http.Error(w, "Internal server error", http.StatusInternalServerError)
            return
        }

        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(result)
    }
}

func handleAuthRefresh(authController *httpauth.Controller, logger *logger.Logger) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        ctx := r.Context()

        var req httpauth.RefreshTokenRequest
        if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
            logger.Warn(ctx, "Failed to decode refresh token request", "error", err)
            http.Error(w, "Invalid request body", http.StatusBadRequest)
            return
        }

        result, err := authController.RefreshToken(ctx, &req)
        if err != nil {
            logger.Error(ctx, "Refresh token failed", "error", err)
            http.Error(w, "Internal server error", http.StatusInternalServerError)
            return
        }

        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(result)
    }
}

func handleAuthLogout(authController *httpauth.Controller, logger *logger.Logger) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        ctx := r.Context()

        var req httpauth.LogoutRequest
        if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
            logger.Warn(ctx, "Failed to decode logout request", "error", err)
            http.Error(w, "Invalid request body", http.StatusBadRequest)
            return
        }

        result, err := authController.Logout(ctx, &req)
        if err != nil {
            logger.Error(ctx, "Logout failed", "error", err)
            http.Error(w, "Internal server error", http.StatusInternalServerError)
            return
        }

        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(result)
    }
}

func handleGenerateLink(deepLinkController *httpdeeplink.Controller, logger *logger.Logger) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        ctx := r.Context()

        var req httpdeeplink.GenerateLinkRequest
        if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
            logger.Warn(ctx, "Failed to decode generate link request", "error", err)
            http.Error(w, "Invalid request body", http.StatusBadRequest)
            return
        }

        result, err := deepLinkController.GenerateLink(ctx, &req)
        if err != nil {
            logger.Error(ctx, "Generate link failed", "error", err)
            http.Error(w, "Internal server error", http.StatusInternalServerError)
            return
        }

        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(result)
    }
}

func handleValidateLink(deepLinkController *httpdeeplink.Controller, logger *logger.Logger) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        ctx := r.Context()

        var req httpdeeplink.ValidateLinkRequest
        if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
            logger.Warn(ctx, "Failed to decode validate link request", "error", err)
            http.Error(w, "Invalid request body", http.StatusBadRequest)
            return
        }

        result, err := deepLinkController.ValidateLink(ctx, &req)
        if err != nil {
            logger.Error(ctx, "Validate link failed", "error", err)
            http.Error(w, "Internal server error", http.StatusInternalServerError)
            return
        }

        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(result)
    }
}

func handleValidateLinkGet(deepLinkController *httpdeeplink.Controller, logger *logger.Logger) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        ctx := r.Context()

        link := chi.URLParam(r, "link")
        req := &httpdeeplink.ValidateLinkRequest{
            Link: link,
        }

        result, err := deepLinkController.ValidateLink(ctx, req)
        if err != nil {
            logger.Error(ctx, "Validate link failed", "error", err)
            http.Error(w, "Internal server error", http.StatusInternalServerError)
            return
        }

        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(result)
    }
}

func handleTelegramUpdate(webhookController *httpwebhook.Controller, logger *logger.Logger) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        ctx := r.Context()

        var req httpwebhook.TelegramUpdateRequest
        if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
            logger.Warn(ctx, "Failed to decode telegram update request", "error", err)
            http.Error(w, "Invalid request body", http.StatusBadRequest)
            return
        }

        result, err := webhookController.TelegramUpdate(ctx, &req)
        if err != nil {
            logger.Error(ctx, "Telegram update failed", "error", err)
            http.Error(w, "Internal server error", http.StatusInternalServerError)
            return
        }

        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(result)
    }
}

func handleVerifyWebhook(webhookController *httpwebhook.Controller, logger *logger.Logger) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        ctx := r.Context()

        var req httpwebhook.VerifyWebhookRequest
        if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
            logger.Warn(ctx, "Failed to decode verify webhook request", "error", err)
            http.Error(w, "Invalid request body", http.StatusBadRequest)
            return
        }

        result, err := webhookController.VerifyWebhook(ctx, &req)
        if err != nil {
            logger.Error(ctx, "Verify webhook failed", "error", err)
            http.Error(w, "Internal server error", http.StatusInternalServerError)
            return
        }

        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(result)
    }
}