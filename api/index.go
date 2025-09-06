// File: api/index.go
package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"sync"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ====================================================================================
// Global Database Connection Pool
//
// In a serverless environment, global variables are preserved between invocations
// of the same function instance. We use a sync.Once to ensure the database
// connection pool is initialized only once per instance, improving performance.
// ====================================================================================

var (
	dbpool *pgxpool.Pool
	once   sync.Once
)

// initDB establishes the database connection pool. It fetches the connection string
// from an environment variable for security and flexibility.
func initDB() {
	once.Do(func() {
		conninfo := os.Getenv("DATABASE_URL")
		if conninfo == "" {
			// Fallback to the hardcoded string from the C++ code if the env var is not set.
			// IMPORTANT: It is strongly recommended to use environment variables in production.
			conninfo = "postgresql://postgres.vxqsqaysrpxliofqxjyu:the-plus-maps-password@aws-0-us-east-2.pooler.supabase.com:5432/postgres?sslmode=require"
		}

		var err error
		dbpool, err = pgxpool.New(context.Background(), conninfo)
		if err != nil {
			log.Fatalf("Unable to create connection pool: %v\n", err)
		}
	})
}

// ====================================================================================
// Data Structs (Mirrors C++ structs and TypeScript interfaces)
//
// These structs use `json` tags to control how they are serialized into JSON,
// ensuring the API output matches the frontend's expectations.
// ====================================================================================

// GeoPoint corresponds to the GeoJSON-style point.
type GeoPoint struct {
	Type        string    `json:"type"`        // e.g., "Point"
	Coordinates []float64 `json:"coordinates"` // [longitude, latitude]
}

// Landmark corresponds to a single landmark object.
type Landmark struct {
	LandmarkName      string   `json:"landmarkName"`
	LandmarkCordinates GeoPoint `json:"landmarkCordinates"`
	LandmarkPinSVG  string   `json:"landmarkPinSVG"`
}

// Location is the Go equivalent of the C++ Location struct.
// Note that fields that are JSON objects in the final output (boards, coordinates, landmarks)
// are represented by their corresponding Go struct types.
type Location struct {
	ID                  string      `json:"id"`
	Name                string      `json:"name"`
	Country             string      `json:"country"`
	State               string      `json:"state"`
	Description         string      `json:"description"`
	SvgLink             string      `json:"svg_link"`
	Rating              float64     `json:"rating"`
	MapMainImage        string      `json:"map_main_image"`
	MapCoverImage       string      `json:"map_cover_image"`
	MainBackgroundImage string      `json:"main_background_image"`
	MapFullAddress      string      `json:"map_full_address"`
	MapPngLink          string      `json:"map_png_link"`
	Boards              interface{} `json:"boards"`      // Use interface{} to hold the parsed JSON
	Coordinates         *GeoPoint   `json:"coordinates"` // Pointer to allow for null
	Landmarks           []Landmark  `json:"landmarks"`   // Slice to allow for null/empty array
}

// Chart is the Go equivalent of the C++ Chart struct.
type Chart struct {
	ID         int         `json:"id"`
	LocationID string      `json:"location_id"`
	ChartType  string      `json:"chart_type"`
	Title      string      `json:"title"`
	ChartData  interface{} `json:"chart_data"` // Use interface{} to hold the parsed JSON
}

// ====================================================================================
// Application Service (Mirrors C++ AppService)
//
// This struct holds the business logic for interacting with the database.
// Methods are attached to this struct, similar to C++ class methods.
//====================================================================================

type AppService struct {
	db *pgxpool.Pool
}

// NewAppService creates a new instance of our application service.
func NewAppService(db *pgxpool.Pool) *AppService {
	return &AppService{db: db}
}

// rowToLocation converts a database row into a Go Location struct.
// It handles parsing the JSON strings from the 'boards', 'coordinates', and 'landmarks' columns.
func (s *AppService) rowToLocation(row pgx.Row) (Location, error) {
	var loc Location
	// These string variables will temporarily hold the JSON data from the database
	var boardsJSON, coordinatesJSON, landmarksJSON string

	err := row.Scan(
		&loc.ID, &loc.Name, &loc.Country, &loc.State, &loc.Description,
		&loc.SvgLink, &loc.Rating, &loc.MapMainImage, &loc.MapCoverImage,
		&loc.MainBackgroundImage, &loc.MapFullAddress, &loc.MapPngLink,
		&boardsJSON, &coordinatesJSON, &landmarksJSON,
	)
	if err != nil {
		return Location{}, err
	}

	// Safely parse the JSON strings into their respective struct fields
	if boardsJSON != "" {
		_ = json.Unmarshal([]byte(boardsJSON), &loc.Boards)
	}
	if coordinatesJSON != "" {
		_ = json.Unmarshal([]byte(coordinatesJSON), &loc.Coordinates)
	}
	if landmarksJSON != "" {
		_ = json.Unmarshal([]byte(landmarksJSON), &loc.Landmarks)
	}

	return loc, nil
}

// GetTopLocations fetches the top-rated locations.
func (s *AppService) GetTopLocations(ctx context.Context, limit int) ([]Location, error) {
	rows, err := s.db.Query(ctx, "SELECT * FROM get_top_locations($1);", limit)
	if err != nil {
		return nil, fmt.Errorf("database query failed: %w", err)
	}
	defer rows.Close()

	var locations []Location
	for rows.Next() {
		loc, err := s.rowToLocation(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan location row: %w", err)
		}
		locations = append(locations, loc)
	}
	return locations, nil
}

// GetLocationByID fetches a single location by its ID.
func (s *AppService) GetLocationByID(ctx context.Context, id string) (Location, error) {
	row := s.db.QueryRow(ctx, "SELECT * FROM get_location_by_id($1);", id)
	loc, err := s.rowToLocation(row)
	if err != nil {
		return Location{}, fmt.Errorf("location not found or scan failed: %w", err)
	}
	return loc, nil
}

// SearchLocations performs a text-based search for locations.
func (s *AppService) SearchLocations(ctx context.Context, query string) ([]Location, error) {
	rows, err := s.db.Query(ctx, "SELECT * FROM search_locations($1);", query)
	if err != nil {
		return nil, fmt.Errorf("database search query failed: %w", err)
	}
	defer rows.Close()

	var locations []Location
	for rows.Next() {
		loc, err := s.rowToLocation(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan location row: %w", err)
		}
		locations = append(locations, loc)
	}
	return locations, nil
}

// GetChartsForLocation fetches all charts associated with a location.
func (s *AppService) GetChartsForLocation(ctx context.Context, locationID string) ([]Chart, error) {
	rows, err := s.db.Query(ctx, "SELECT id, location_id, chart_type, title, chart_data FROM charts WHERE location_id = $1 ORDER BY id;", locationID)
	if err != nil {
		return nil, fmt.Errorf("chart query failed: %w", err)
	}
	defer rows.Close()

	var charts []Chart
	for rows.Next() {
		var chart Chart
		var chartDataJSON string
		if err := rows.Scan(&chart.ID, &chart.LocationID, &chart.ChartType, &chart.Title, &chartDataJSON); err != nil {
			return nil, fmt.Errorf("failed to scan chart row: %w", err)
		}
		if chartDataJSON != "" {
			_ = json.Unmarshal([]byte(chartDataJSON), &chart.ChartData)
		}
		charts = append(charts, chart)
	}
	return charts, nil
}

// ====================================================================================
// User Logging and Rate Limiting Helpers (Mirrors C++ helpers)
// ====================================================================================

func logUserRequest(ctx context.Context, uid string) {
	_, err := dbpool.Exec(ctx, "SELECT log_user_request($1);", uid)
	if err != nil {
		log.Printf("Error logging user request for UID %s: %v", uid, err)
	}
}

func logUserResponse(ctx context.Context, uid string) {
	_, err := dbpool.Exec(ctx, "SELECT log_user_response($1);", uid)
	if err != nil {
		log.Printf("Error logging user response for UID %s: %v", uid, err)
	}
}

func isUserBlocked(ctx context.Context, uid string) (bool, error) {
	var blocked bool
	err := dbpool.QueryRow(ctx, "SELECT is_user_blocked($1);", uid).Scan(&blocked)
	if err != nil {
		// If the function doesn't return a row, assume not blocked but log the error.
		log.Printf("Error checking if user is blocked for UID %s: %v", uid, err)
		return false, nil
	}
	return blocked, nil
}

// ====================================================================================
// JSON-RPC Handling and Dispatcher
// ====================================================================================

// RpcRequest defines the structure of an incoming JSON-RPC request.
// Using json.RawMessage for Params allows us to delay parsing until we know the method.
type RpcRequest struct {
	Method string          `json:"method"`
	Params json.RawMessage `json:"params"`
}

// GenericParams is used to extract the 'userid' before full param parsing.
type GenericParams struct {
	UserID string `json:"userid"`
}

// writeJSONResponse is a helper to standardize JSON responses.
func writeJSONResponse(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(data)
}

// handleRpcRequest is the core dispatcher, equivalent to the C++ RpcDispatcher.
func handleRpcRequest(w http.ResponseWriter, r *http.Request) {
	// Decode the generic RPC request to find the method and params.
	var req RpcRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONResponse(w, http.StatusBadRequest, map[string]interface{}{"success": false, "error": "Invalid JSON request"})
		return
	}

	// Extract userid for logging and rate limiting.
	var genericParams GenericParams
	_ = json.Unmarshal(req.Params, &genericParams)
	uid := genericParams.UserID

	ctx := r.Context()
	service := NewAppService(dbpool)

	// User blocking logic
	if uid != "" {
		blocked, err := isUserBlocked(ctx, uid)
		if err != nil {
			// Log the error but proceed; fail open.
			log.Printf("Could not check user block status: %v", err)
		}
		if blocked {
			writeJSONResponse(w, http.StatusTooManyRequests, map[string]interface{}{"success": false, "error": "Rate limit exceeded"})
			return
		}
		logUserRequest(ctx, uid)
		// Defer logging the response until the function returns.
		defer logUserResponse(ctx, uid)
	}

	var resultData interface{}
	var processingError error

	// Dispatch to the correct method handler.
	switch req.Method {
	case "getTopLocations":
		var params struct {
			Limit int `json:"limit"`
		}
		_ = json.Unmarshal(req.Params, &params)
		if params.Limit == 0 {
			params.Limit = 10 // Default limit
		}
		resultData, processingError = service.GetTopLocations(ctx, params.Limit)

	case "getLocationById":
		var params struct {
			ID string `json:"id"`
		}
		if err := json.Unmarshal(req.Params, &params); err != nil || params.ID == "" {
			processingError = fmt.Errorf("missing or invalid 'id' parameter")
			break
		}
		resultData, processingError = service.GetLocationByID(ctx, params.ID)

	case "searchLocations":
		var params struct {
			Query string `json:"query"`
		}
		if err := json.Unmarshal(req.Params, &params); err != nil || params.Query == "" {
			processingError = fmt.Errorf("missing or invalid 'query' parameter")
			break
		}
		resultData, processingError = service.SearchLocations(ctx, params.Query)

	case "getChartsForLocation":
		var params struct {
			LocationID string `json:"locationId"`
		}
		if err := json.Unmarshal(req.Params, &params); err != nil || params.LocationID == "" {
			processingError = fmt.Errorf("missing or invalid 'locationId' parameter")
			break
		}
		resultData, processingError = service.GetChartsForLocation(ctx, params.LocationID)

	default:
		processingError = fmt.Errorf("method not found: %s", req.Method)
	}

	// Send the final response.
	if processingError != nil {
		writeJSONResponse(w, http.StatusBadRequest, map[string]interface{}{"success": false, "error": processingError.Error()})
		return
	}

	writeJSONResponse(w, http.StatusOK, map[string]interface{}{"success": true, "data": resultData})
}

// ====================================================================================
// Main Vercel Handler
//
// This is the single entry point for all requests to the serverless function.
// It sets CORS headers and routes requests to the appropriate handler based on the path.
// ====================================================================================

func Handler(w http.ResponseWriter, r *http.Request) {
	// Initialize the database connection pool on the first request.
	initDB()

	// Set CORS headers for all responses. This is equivalent to the Drogon post-handling advice.
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")

	// Handle CORS preflight requests.
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	// Simple path-based routing.
	switch r.URL.Path {
	case "/health":
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	case "/rpc":
		if r.Method != "POST" {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		handleRpcRequest(w, r)
	default:
		http.NotFound(w, r)
	}
}