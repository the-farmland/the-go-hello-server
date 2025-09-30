// File: api/index.go
package handler

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"sync"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ====================================================================================
// Global Database Connection Pool
// ====================================================================================

var (
	dbpool *pgxpool.Pool
	once   sync.Once
)

func initDB() {
	once.Do(func() {
		conninfo := os.Getenv("DATABASE_URL")
		if conninfo == "" {
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
// Data Structs
// ====================================================================================

type GeoPoint struct {
	Type string  `json:"type"`
	Lon  float64 `json:"lon"`
	Lat  float64 `json:"lat"`
}

type Pin struct {
	Name    string  `json:"name"`
	PinLink string  `json:"pinLink"`
	Lon     float64 `json:"lon"`
	Lat     float64 `json:"lat"`
	Type    string  `json:"type"`
	Info    string  `json:"info,omitempty"`
	Img     string  `json:"img,omitempty"`
	URL     string  `json:"url,omitempty"`
}

// New GeoJSON Pin structure
type GeoJsonPin struct {
	Name    string  `json:"name"`
	PinLink string  `json:"pinLink"`
	Lon     float64 `json:"lon"`
	Lat     float64 `json:"lat"`
	Type    string  `json:"type"`
	Info    string  `json:"info,omitempty"`
	Img     string  `json:"img,omitempty"`
	URL     string  `json:"url,omitempty"`
}

// GeoJSON Style configuration
type GeoJsonStyle struct {
	StrokeWidth float64 `json:"strokeWidth,omitempty"`
	StrokeColor string  `json:"strokeColor,omitempty"`
	FillOpacity float64 `json:"fillOpacity,omitempty"`
	Opacity     float64 `json:"opacity,omitempty"`
	Weight      int     `json:"weight,omitempty"`
	DashArray   string  `json:"dashArray,omitempty"`
	LineCap     string  `json:"lineCap,omitempty"`
	LineJoin    string  `json:"lineJoin,omitempty"`
}

// Enhanced GeoJSON Data structure
type GeoJsonData struct {
	Type       string        `json:"type"`
	Features   []interface{} `json:"features,omitempty"`
	Geometry   interface{}   `json:"geometry,omitempty"`
	Properties interface{}   `json:"properties,omitempty"`
	// Enhanced properties
	Subtype    string        `json:"subtype,omitempty"`
	Title      string        `json:"title,omitempty"`
	Info       string        `json:"info,omitempty"`
	Fill       string        `json:"fill,omitempty"` // Color hex code or image URL
	GeoJsonPin *GeoJsonPin   `json:"geojsonpin,omitempty"`
	Img        string        `json:"img,omitempty"` // For popup modal
	Style      *GeoJsonStyle `json:"style,omitempty"`
}

type Sublocation struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Info        string    `json:"info"`
	Coordinates *GeoPoint `json:"coordinates"`
	SvgPin      string    `json:"svgpin"`
	Zoom        string    `json:"zoom"`
}

type SublocationsData struct {
	CurrentSublocation *Sublocation  `json:"current_sublocation,omitempty"`
	AllSublocations    []Sublocation `json:"all_sublocations,omitempty"`
}

type Location struct {
	ID                  string            `json:"id"`
	Name                string            `json:"name"`
	Country             string            `json:"country"`
	State               string            `json:"state"`
	Description         string            `json:"description"`
	SvgLink             string            `json:"svg_link"`
	Rating              float64           `json:"rating"`
	MapMainImage        string            `json:"map_main_image"`
	MapCoverImage       string            `json:"map_cover_image"`
	MainBackgroundImage string            `json:"main_background_image"`
	MapFullAddress      string            `json:"map_full_address"`
	MapPngLink          string            `json:"map_png_link"`
	Boards              interface{}       `json:"boards"`
	Coordinates         *GeoPoint         `json:"coordinates"`
	Landmarks           []Pin             `json:"landmarks"`
	ParentLocationID    string            `json:"parent_location_id"`
	Business            []Pin             `json:"business"`
	Hospitality         []Pin             `json:"hospitality"`
	Events              []Pin             `json:"events"`
	PSA                 []Pin             `json:"psa"`
	Sublocations        *SublocationsData `json:"sublocations,omitempty"`
	Geojson             *GeoJsonData      `json:"geojson,omitempty"` // Changed to structured data
	Hotzones            []Pin             `json:"hotzones,omitempty"`
	Zoom                string            `json:"zoom"`
}

type Chart struct {
	ID         int         `json:"id"`
	LocationID string      `json:"location_id"`
	ChartType  string      `json:"chart_type"`
	Title      string      `json:"title"`
	ChartData  interface{} `json:"chart_data"`
}

// ====================================================================================
// Application Service
// ====================================================================================

type AppService struct {
	db *pgxpool.Pool
}

func NewAppService(db *pgxpool.Pool) *AppService {
	return &AppService{db: db}
}

// Helper function to parse coordinates from database JSON
// Database stores: {"lat": "47.5798", "lon": "19.2474", "type": "Point"}
// We need to convert to proper float64 values
func parseCoordinates(coordinatesJSON string) (*GeoPoint, error) {
	if coordinatesJSON == "" {
		return nil, nil
	}

	var rawCoords map[string]interface{}
	if err := json.Unmarshal([]byte(coordinatesJSON), &rawCoords); err != nil {
		log.Printf("Error parsing coordinates JSON: %v", err)
		return nil, err
	}

	// Extract lat and lon - they might be strings or floats
	var lat, lon float64
	var coordType string = "Point"

	// Handle lat
	if latVal, ok := rawCoords["lat"]; ok {
		switch v := latVal.(type) {
		case string:
			if parsed, err := strconv.ParseFloat(v, 64); err == nil {
				lat = parsed
			}
		case float64:
			lat = v
		}
	}

	// Handle lon
	if lonVal, ok := rawCoords["lon"]; ok {
		switch v := lonVal.(type) {
		case string:
			if parsed, err := strconv.ParseFloat(v, 64); err == nil {
				lon = parsed
			}
		case float64:
			lon = v
		}
	}

	// Handle type
	if typeVal, ok := rawCoords["type"].(string); ok {
		coordType = typeVal
	}

	// Validate coordinates
	if lat == 0 && lon == 0 {
		log.Printf("Warning: Coordinates are zero or invalid: %s", coordinatesJSON)
		return nil, fmt.Errorf("invalid coordinates")
	}

	return &GeoPoint{
		Type: coordType,
		Lat:  lat,
		Lon:  lon,
	}, nil
}

// Helper function to parse pin coordinates from nested JSON
// Database stores pins like: {"coordinates": {"coordinates": [lon, lat]}}
func parsePinCoordinates(coords map[string]interface{}) (float64, float64, error) {
	// First check if coordinates field exists
	if coordsField, ok := coords["coordinates"].(map[string]interface{}); ok {
		// Then check for nested coordinates array
		if coordsArray, ok := coordsField["coordinates"].([]interface{}); ok && len(coordsArray) == 2 {
			lon, lonOk := coordsArray[0].(float64)
			lat, latOk := coordsArray[1].(float64)
			if lonOk && latOk {
				return lon, lat, nil
			}
		}
	}

	// Try direct coordinates array format: {"coordinates": [lon, lat]}
	if coordsArray, ok := coords["coordinates"].([]interface{}); ok && len(coordsArray) == 2 {
		lon, lonOk := coordsArray[0].(float64)
		lat, latOk := coordsArray[1].(float64)
		if lonOk && latOk {
			return lon, lat, nil
		}
	}

	return 0, 0, fmt.Errorf("invalid coordinate format")
}

func (s *AppService) rowToLocation(row pgx.Row) (Location, error) {
	var loc Location

	var state, svgLink, mapMainImage, mapCoverImage, mainBgImage, mapFullAddress, mapPngLink, parentLocationID, geojson, zoom sql.NullString
	var rating sql.NullFloat64
	var boardsJSON, coordinatesJSON, landmarksJSON, businessJSON, hospitalityJSON, eventsJSON, psaJSON, sublocationsJSON, hotzonesJSON sql.NullString

	err := row.Scan(
		&loc.ID, &loc.Name, &loc.Country, &state, &loc.Description,
		&svgLink, &rating, &mapMainImage, &mapCoverImage,
		&mainBgImage, &mapFullAddress, &mapPngLink,
		&boardsJSON, &coordinatesJSON, &landmarksJSON,
		&parentLocationID, &businessJSON, &hospitalityJSON, &eventsJSON, &psaJSON,
		&sublocationsJSON, &geojson, &hotzonesJSON, &zoom,
	)
	if err != nil {
		return Location{}, err
	}

	if state.Valid {
		loc.State = state.String
	}
	if svgLink.Valid {
		loc.SvgLink = svgLink.String
	}
	if rating.Valid {
		loc.Rating = rating.Float64
	}
	if mapMainImage.Valid {
		loc.MapMainImage = mapMainImage.String
	}
	if mapCoverImage.Valid {
		loc.MapCoverImage = mapCoverImage.String
	}
	if mainBgImage.Valid {
		loc.MainBackgroundImage = mainBgImage.String
	}
	if mapFullAddress.Valid {
		loc.MapFullAddress = mapFullAddress.String
	}
	if mapPngLink.Valid {
		loc.MapPngLink = mapPngLink.String
	}
	if parentLocationID.Valid {
		loc.ParentLocationID = parentLocationID.String
	}
	if zoom.Valid {
		loc.Zoom = zoom.String
	}

	// Parse GeoJSON with enhanced structure
	if geojson.Valid && geojson.String != "" {
		var geoJsonData GeoJsonData
		if err := json.Unmarshal([]byte(geojson.String), &geoJsonData); err == nil {
			loc.Geojson = &geoJsonData
		}
	}

	if boardsJSON.Valid && boardsJSON.String != "" {
		_ = json.Unmarshal([]byte(boardsJSON.String), &loc.Boards)
	}

	// Parse main coordinates using helper function
	if coordinatesJSON.Valid && coordinatesJSON.String != "" {
		if parsedCoords, err := parseCoordinates(coordinatesJSON.String); err == nil {
			loc.Coordinates = parsedCoords
			log.Printf("Parsed main coordinates for %s: lat=%f, lon=%f", loc.ID, parsedCoords.Lat, parsedCoords.Lon)
		} else {
			log.Printf("Failed to parse main coordinates for %s: %v", loc.ID, err)
		}
	}

	// Helper function to parse pin arrays
	parsePinArray := func(jsonStr string) []Pin {
		var pins []Pin
		var rawPins []map[string]interface{}
		if err := json.Unmarshal([]byte(jsonStr), &rawPins); err == nil {
			for _, raw := range rawPins {
				pin := Pin{}
				if name, ok := raw["name"].(string); ok {
					pin.Name = name
				}
				if pinLink, ok := raw["pinLink"].(string); ok {
					pin.PinLink = pinLink
				}
				if pinType, ok := raw["type"].(string); ok {
					pin.Type = pinType
				}
				if info, ok := raw["info"].(string); ok {
					pin.Info = info
				}
				if img, ok := raw["img"].(string); ok {
					pin.Img = img
				}
				if url, ok := raw["url"].(string); ok {
					pin.URL = url
				}
				if coords, ok := raw["coordinates"].(map[string]interface{}); ok {
					if lon, lat, err := parsePinCoordinates(coords); err == nil {
						pin.Lon = lon
						pin.Lat = lat
					}
				}
				pins = append(pins, pin)
			}
		}
		return pins
	}

	// Parse all pin arrays
	if landmarksJSON.Valid && landmarksJSON.String != "" {
		loc.Landmarks = parsePinArray(landmarksJSON.String)
	}

	if businessJSON.Valid && businessJSON.String != "" {
		loc.Business = parsePinArray(businessJSON.String)
	}

	if hospitalityJSON.Valid && hospitalityJSON.String != "" {
		loc.Hospitality = parsePinArray(hospitalityJSON.String)
	}

	if eventsJSON.Valid && eventsJSON.String != "" {
		loc.Events = parsePinArray(eventsJSON.String)
	}

	if psaJSON.Valid && psaJSON.String != "" {
		loc.PSA = parsePinArray(psaJSON.String)
	}

	if hotzonesJSON.Valid && hotzonesJSON.String != "" {
		loc.Hotzones = parsePinArray(hotzonesJSON.String)
	}

	// Parse sublocations
	if sublocationsJSON.Valid && sublocationsJSON.String != "" {
		var sublocs SublocationsData
		if err := json.Unmarshal([]byte(sublocationsJSON.String), &sublocs); err == nil {
			// Parse coordinates for current sublocation
			if sublocs.CurrentSublocation != nil && sublocs.CurrentSublocation.Coordinates != nil {
				// Coordinates are already parsed as GeoPoint, just validate
				log.Printf("Current sublocation coordinates: lat=%f, lon=%f", 
					sublocs.CurrentSublocation.Coordinates.Lat, 
					sublocs.CurrentSublocation.Coordinates.Lon)
			}
			// Parse coordinates for all sublocations
			if sublocs.AllSublocations != nil {
				for i := range sublocs.AllSublocations {
					if sublocs.AllSublocations[i].Coordinates != nil {
						log.Printf("Sublocation %s coordinates: lat=%f, lon=%f", 
							sublocs.AllSublocations[i].ID,
							sublocs.AllSublocations[i].Coordinates.Lat, 
							sublocs.AllSublocations[i].Coordinates.Lon)
					}
				}
			}
			loc.Sublocations = &sublocs
		}
	}

	return loc, nil
}

func (s *AppService) GetTopLocations(ctx context.Context, limit int) ([]Location, error) {
	// Add validation
	if limit <= 0 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}

	rows, err := s.db.Query(ctx, "SELECT * FROM get_top_locations($1);", limit)
	if err != nil {
		return nil, fmt.Errorf("database query failed: %w", err)
	}
	defer rows.Close()

	var locations []Location
	for rows.Next() {
		loc, err := s.rowToLocation(rows)
		if err != nil {
			log.Printf("Warning: failed to scan location row: %v", err)
			continue // Skip invalid rows instead of failing completely
		}
		locations = append(locations, loc)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return locations, nil
}

func (s *AppService) GetLocationByID(ctx context.Context, id string) (Location, error) {
	row := s.db.QueryRow(ctx, "SELECT * FROM get_location_by_id($1);", id)
	loc, err := s.rowToLocation(row)
	if err != nil {
		return Location{}, fmt.Errorf("location not found or scan failed: %w", err)
	}
	return loc, nil
}

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

func (s *AppService) GetChartsForLocation(ctx context.Context, locationID string) ([]Chart, error) {
	rows, err := s.db.Query(ctx, "SELECT id, location_id, chart_type, title, chart_data FROM charts WHERE location_id = $1 ORDER BY id;", locationID)
	if err != nil {
		return nil, fmt.Errorf("chart query failed: %w", err)
	}
	defer rows.Close()

	var charts []Chart
	for rows.Next() {
		var chart Chart
		var chartDataJSON sql.NullString
		if err := rows.Scan(&chart.ID, &chart.LocationID, &chart.ChartType, &chart.Title, &chartDataJSON); err != nil {
			return nil, fmt.Errorf("failed to scan chart row: %w", err)
		}
		if chartDataJSON.Valid && chartDataJSON.String != "" {
			_ = json.Unmarshal([]byte(chartDataJSON.String), &chart.ChartData)
		}
		charts = append(charts, chart)
	}
	return charts, nil
}

// ====================================================================================
// User Logging and Rate Limiting Helpers
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
		log.Printf("Error checking if user is blocked for UID %s: %v", uid, err)
		return false, nil
	}
	return blocked, nil
}

// ====================================================================================
// JSON-RPC Handling and Dispatcher
// ====================================================================================

type RpcRequest struct {
	Method string          `json:"method"`
	Params json.RawMessage `json:"params"`
}

type GenericParams struct {
	UserID string `json:"userid"`
}

func writeJSONResponse(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(data)
}

func handleRpcRequest(w http.ResponseWriter, r *http.Request) {
	var req RpcRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONResponse(w, http.StatusBadRequest, map[string]interface{}{"success": false, "error": "Invalid JSON request"})
		return
	}

	var genericParams GenericParams
	_ = json.Unmarshal(req.Params, &genericParams)
	uid := genericParams.UserID

	ctx := r.Context()
	service := NewAppService(dbpool)

	if uid != "" {
		blocked, err := isUserBlocked(ctx, uid)
		if err != nil {
			log.Printf("Could not check user block status: %v", err)
		}
		if blocked {
			writeJSONResponse(w, http.StatusTooManyRequests, map[string]interface{}{"success": false, "error": "Rate limit exceeded"})
			return
		}
		logUserRequest(ctx, uid)
		defer logUserResponse(ctx, uid)
	}

	var resultData interface{}
	var processingError error

	switch req.Method {
	case "getTopLocations":
		var params struct {
			Limit int `json:"limit"`
		}
		_ = json.Unmarshal(req.Params, &params)
		if params.Limit == 0 {
			params.Limit = 10
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

	if processingError != nil {
		writeJSONResponse(w, http.StatusBadRequest, map[string]interface{}{"success": false, "error": processingError.Error()})
		return
	}

	writeJSONResponse(w, http.StatusOK, map[string]interface{}{"success": true, "data": resultData})
}

// ====================================================================================
// Main Vercel Handler
// ====================================================================================

func Handler(w http.ResponseWriter, r *http.Request) {
	initDB()

	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusNoContent)
		return
	}

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
