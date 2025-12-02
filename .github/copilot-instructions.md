# GitHub Copilot Instructions - iCCTV HTTP Service

## 🏗️ Architecture Overview

This is a Go-based HTTP service for managing OrangePi devices, buildings, and NVR systems in an iCCTV surveillance network. The project follows a clean 3-layer architecture:

- **Controllers** (`controllers/`): Handle HTTP requests/responses with interface-based design
- **Services** (`services/`): Business logic with dependency injection pattern
- **Models** (`models/`): Data structures with GORM ORM and JSON serialization

Key architectural patterns:

- Interface-first design (all services/controllers define interfaces first)
- Dependency injection via `container/` package
- Centralized routing in `routes/routes.go` with middleware integration
- GORM for database operations with MySQL/SQLite dual support

## 🔧 Essential Development Commands

```bash
# Build and run
go run main.go                    # Starts service on :8080 (or HTTP_ADDR env var)
go build -o main.exe .           # Build executable

# Database operations
# Uses DB_DRIVER=mysql or sqlite (default: sqlite)
# MySQL: DB_HOST, DB_PORT, DB_USER, DB_PASS, DB_NAME env vars
# SQLite: Creates local .db file automatically

# Testing
powershell -ExecutionPolicy Bypass -File test_*.ps1  # Run API test scripts
```

## 📝 Code Conventions & Patterns

### Service Layer Pattern

```go
// Always define interfaces first
type XxxServiceInterface interface {
    Method(ctx context.Context, params) (result, error)
}

// Constructor with dependency injection
func NewXxxService(db *gorm.DB, dep *OtherService) *XxxService {
    return &XxxService{db: db, dep: dep}
}
```

### Controller Layer Pattern

```go
// Request payload structs with pointer types for optional fields
type xxxPayload struct {
    RequiredField string  `json:"required_field"`
    OptionalField *string `json:"optional_field"`  // Use pointer for nullable
}

// Standard response pattern
func (c *XxxController) Method(w http.ResponseWriter, r *http.Request) {
    // 1. Parse request
    // 2. Call service
    // 3. Use respondData()/respondError() from base.go
}
```

### Model Layer Pattern

```go
type Xxx struct {
    ModelFields                    // Embed common fields (ID, CreatedAt, etc.)
    FieldName string `gorm:"type:varchar(255);not null" json:"field_name"`

    // JSON fields use GORM serializer
    JSONField []CustomType `gorm:"type:json;serializer:json" json:"json_field"`

    // Foreign key relationships
    Related *RelatedModel `gorm:"foreignKey:ForeignKey;references:ID" json:"related,omitempty"`
}
```

## 🔗 Critical Integration Points

### Authentication Flow

- JWT-based auth via `middlewares/auth_middleware.go`
- All admin endpoints require `requireAdmin()` middleware wrapper
- Token validation through `AuthService.ValidateToken()`

### Database Relationships

- **OrangePi ↔ Building**: Via `ISmartID` field (not direct ID)
- **Building ↔ NVR**: Via `BuildingID` foreign key
- **MediaMTX Paths**: Stored as JSON in `OrangePi.MediaMTXPaths` field

### External Service Integration

- **OrangePi Remote Management**: HTTP calls to remote devices via `PublicNetService`
- **Port Updates**: Remote FRPC configuration through `/api/orangepi/remote/ports`
- **Health Checks**: Device status monitoring via `/api/orangepi/remote/health`

## 🚨 Project-Specific Gotchas

### JSON Field Handling

When updating JSON fields (like `MediaMTXPaths`), always use the `updateMediaMTXPaths bool` parameter pattern in service methods to control selective updates.

### ISmartID vs ID

Buildings use `ISmartID` (string) as the primary identifier for OrangePi relationships, not the numeric `ID`. This is a domain-specific requirement for iCCTV integration.

### Remote Device Communication

All remote OrangePi operations require:

1. Device lookup for port configuration
2. Public network config retrieval via `PublicNetService.Get()`
3. HTTP client with 15-30s timeouts
4. Proper error handling for network failures

### Response Format

All API responses use the standardized format from `controllers/base.go`:

```go
// Success: {"success": true, "data": {...}}
// Error: {"success": false, "error": "message"}
```

## 📁 Key Files for Understanding

- `container/container.go`: Dependency injection setup and service wiring
- `routes/routes.go`: Complete API endpoint mapping with middleware
- `models/model.go`: Base model structure and pagination types
- `databases/db.go`: Dual database support (MySQL/SQLite) initialization
- `controllers/base.go`: Standardized response helpers (`respondData`, `respondError`)
- `README.md`: Complete API documentation with examples

## 🔄 Common Workflows

### Adding New Entity

1. Define model in `models/` with `ModelFields` embedding
2. Create service interface and implementation in `services/`
3. Add controller with interface in `controllers/`
4. Register routes in `routes/routes.go`
5. Update container dependencies in `container/container.go`

### Adding New API Endpoint

1. Add method to service interface
2. Implement business logic in service
3. Add controller method using standard pattern
4. Register route with appropriate middleware
5. Update API documentation in `README.md`

### Database Schema Changes

- Modify `sql/init.sql` for new installations
- Use `sql/fix_database.sql` for migration scripts
- Test with both MySQL and SQLite drivers

我的命令行需要使用 powershell，请使用 powershell 命令.
简体中文回答我的问题.
