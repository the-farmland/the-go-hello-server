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
	Name     string  `json:"name"`
	PinLink  string  `json:"pinLink"`
	Lon      float64 `json:"lon"`
	Lat      float64 `json:"lat"`
	Type     string  `json:"type"`
	Info     string  `json:"info,omitempty"`
	Img      string  `json:"img,omitempty"`
	URL      string  `json:"url,omitempty"`
	Position string  `json:"position,omitempty"`
	Team     string  `json:"team,omitempty"`
	Number   string  `json:"number,omitempty"`
	End      string  `json:"end,omitempty"`
}

// New GeoJSON Pin structure
type GeoJsonPin struct {
	Name     string  `json:"name"`
	PinLink  string  `json:"pinLink"`
	Lon      float64 `json:"lon"`
	Lat      float64 `json:"lat"`
	Type     string  `json:"type"`
	Info     string  `json:"info,omitempty"`
	Img      string  `json:"img,omitempty"`
	URL      string  `json:"url,omitempty"`
	Position string  `json:"position,omitempty"`
	Team     string  `json:"team,omitempty"`
	Number   string  `json:"number,omitempty"`
	End      string  `json:"end,omitempty"`
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
	Results             []Pin             `json:"results,omitempty"`
}

type Chart struct {
	ID         int         `json:"id"`
	LocationID string      `json:"location_id"`
	ChartType  string      `json:"chart_type"`
	Title      string      `json:"title"`
	ChartData  interface{} `json:"chart_data"`
}

// Add to Data Structs section
type Reporting struct {
	ID                  int         `json:"id"`
	Name                string      `json:"name"`
	Info                string      `json:"info"`
	Type                string      `json:"type"`
	CreatedBy           string      `json:"created_by"`
	Coordinates         *GeoPoint   `json:"coordinates"`
	CreatedAt           string      `json:"created_at"`
	ParentLocationID    string      `json:"parent_location_id"`
	ParentSublocationID string      `json:"parent_sublocation_id,omitempty"`
}

type Mood struct {
	ID                  int         `json:"id"`
	Name                string      `json:"name"`
	Info                string      `json:"info"`
	Type                string      `json:"type"`
	CreatedBy           string      `json:"created_by"`
	Coordinates         *GeoPoint   `json:"coordinates"`
	CreatedAt           string      `json:"created_at"`
	ParentLocationID    string      `json:"parent_location_id"`
	ParentSublocationID string      `json:"parent_sublocation_id,omitempty"`
}

type Tag struct {
	ID                  int         `json:"id"`
	Item                string      `json:"item"`
	Coordinates         *GeoPoint   `json:"coordinates"`
	CreatedBy           string      `json:"created_by"`
	CreatedAt           string      `json:"created_at"`
	ParentLocationID    string      `json:"parent_location_id"`
	ParentSublocationID string      `json:"parent_sublocation_id,omitempty"`
	Type                string      `json:"type"`
}

type SublocationData struct {
	ID                string    `json:"id"`
	Name              string    `json:"name"`
	Info              string    `json:"info"`
	Coordinates       *GeoPoint `json:"coordinates"`
	SvgPin            string    `json:"svgpin"`
	ParentLocationID  string    `json:"parent_location_id"`
	Zoom              string    `json:"zoom"`
	Type              string    `json:"type"`
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

// Unified coordinate parsing function
// Handles multiple formats:
// 1. {"lat": "47.5798", "lon": "19.2474", "type": "Point"} - main coordinates
// 2. {"coordinates": {"coordinates": [lon, lat]}} - nested pin format
// 3. {"coordinates": [lon, lat]} - simple array format
// 4. {"lat": 47.5798, "lon": 19.2474} - numeric format
func parseAnyCoordinates(data interface{}) (lat float64, lon float64, err error) {
	switch v := data.(type) {
	case string:
		// Parse JSON string
		var coordMap map[string]interface{}
		if err := json.Unmarshal([]byte(v), &coordMap); err != nil {
			return 0, 0, fmt.Errorf("failed to parse coordinate string: %w", err)
		}
		return parseCoordinateMap(coordMap)
	
	case map[string]interface{}:
		return parseCoordinateMap(v)
	
	default:
		return 0, 0, fmt.Errorf("unsupported coordinate type: %T", data)
	}
}

func parseCoordinateMap(coordMap map[string]interface{}) (lat float64, lon float64, err error) {
	// Try direct lat/lon fields first (main coordinates format)
	if latVal, hasLat := coordMap["lat"]; hasLat {
		if lonVal, hasLon := coordMap["lon"]; hasLon {
			lat = parseFloatValue(latVal)
			lon = parseFloatValue(lonVal)
			if lat != 0 || lon != 0 {
				return lat, lon, nil
			}
		}
	}

	// Try nested coordinates format: {"coordinates": {"coordinates": [lon, lat]}}
	if coordsField, ok := coordMap["coordinates"].(map[string]interface{}); ok {
		if coordsArray, ok := coordsField["coordinates"].([]interface{}); ok && len(coordsArray) >= 2 {
			lon = parseFloatValue(coordsArray[0])
			lat = parseFloatValue(coordsArray[1])
			if lat != 0 || lon != 0 {
				return lat, lon, nil
			}
		}
	}

	// Try simple array format: {"coordinates": [lon, lat]}
	if coordsArray, ok := coordMap["coordinates"].([]interface{}); ok && len(coordsArray) >= 2 {
		lon = parseFloatValue(coordsArray[0])
		lat = parseFloatValue(coordsArray[1])
		if lat != 0 || lon != 0 {
			return lat, lon, nil
		}
	}

	return 0, 0, fmt.Errorf("no valid coordinates found in map")
}

func parseFloatValue(val interface{}) float64 {
	switch v := val.(type) {
	case float64:
		return v
	case float32:
		return float64(v)
	case int:
		return float64(v)
	case int64:
		return float64(v)
	case string:
		if parsed, err := strconv.ParseFloat(v, 64); err == nil {
			return parsed
		}
	}
	return 0
}

// Helper function to parse main location coordinates
func parseMainCoordinates(coordinatesJSON string) (*GeoPoint, error) {
	if coordinatesJSON == "" {
		return nil, nil
	}

	lat, lon, err := parseAnyCoordinates(coordinatesJSON)
	if err != nil {
		log.Printf("Error parsing main coordinates: %v", err)
		return nil, err
	}

	if lat == 0 && lon == 0 {
		log.Printf("Warning: Main coordinates are zero: %s", coordinatesJSON)
		return nil, fmt.Errorf("invalid coordinates")
	}

	return &GeoPoint{
		Type: "Point",
		Lat:  lat,
		Lon:  lon,
	}, nil
}

// Unified pin parsing function
func parsePinArray(jsonStr string, pinType string) []Pin {
	if jsonStr == "" {
		return []Pin{}
	}

	var pins []Pin
	var rawPins []map[string]interface{}
	
	if err := json.Unmarshal([]byte(jsonStr), &rawPins); err != nil {
		log.Printf("Error parsing %s pins JSON: %v", pinType, err)
		return []Pin{}
	}

	for idx, raw := range rawPins {
		pin := Pin{}
		
		// Parse basic fields
		if name, ok := raw["name"].(string); ok {
			pin.Name = name
		}
		if pinLink, ok := raw["pinLink"].(string); ok {
			pin.PinLink = pinLink
		}
		if pType, ok := raw["type"].(string); ok {
			pin.Type = pType
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
		// Parse optional fields for results
		if position, ok := raw["position"].(string); ok {
			pin.Position = position
		}
		if team, ok := raw["team"].(string); ok {
			pin.Team = team
		}
		if number, ok := raw["number"].(string); ok {
			pin.Number = number
		}
		if end, ok := raw["end"].(string); ok {
			pin.End = end
		}

		// Parse coordinates using unified function
		// Try multiple possible coordinate fields
		var lat, lon float64
		var coordErr error

		// First try "coordinates" field
		if coords, ok := raw["coordinates"]; ok {
			lat, lon, coordErr = parseAnyCoordinates(coords)
		}

		// If that fails, try direct lat/lon fields
		if coordErr != nil || (lat == 0 && lon == 0) {
			if latVal, hasLat := raw["lat"]; hasLat {
				if lonVal, hasLon := raw["lon"]; hasLon {
					lat = parseFloatValue(latVal)
					lon = parseFloatValue(lonVal)
					coordErr = nil
				}
			}
		}

		if coordErr == nil && (lat != 0 || lon != 0) {
			pin.Lat = lat
			pin.Lon = lon
			log.Printf("Parsed %s pin #%d '%s': lat=%f, lon=%f", pinType, idx, pin.Name, lat, lon)
		} else {
			log.Printf("Warning: Could not parse coordinates for %s pin #%d '%s'", pinType, idx, pin.Name)
		}

		pins = append(pins, pin)
	}

	return pins
}

func (s *AppService) rowToLocation(row pgx.Row) (Location, error) {
	var loc Location

	var state, svgLink, mapMainImage, mapCoverImage, mainBgImage, mapFullAddress, mapPngLink, parentLocationID, geojson, zoom sql.NullString
	var rating sql.NullFloat64
	var boardsJSON, coordinatesJSON, landmarksJSON, businessJSON, hospitalityJSON, eventsJSON, psaJSON, sublocationsJSON, hotzonesJSON, resultsJSON sql.NullString

	err := row.Scan(
		&loc.ID, &loc.Name, &loc.Country, &state, &loc.Description,
		&svgLink, &rating, &mapMainImage, &mapCoverImage,
		&mainBgImage, &mapFullAddress, &mapPngLink,
		&boardsJSON, &coordinatesJSON, &landmarksJSON,
		&parentLocationID, &businessJSON, &hospitalityJSON, &eventsJSON, &psaJSON,
		&sublocationsJSON, &geojson, &hotzonesJSON, &zoom, &resultsJSON,
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

	// Parse main coordinates
	if coordinatesJSON.Valid && coordinatesJSON.String != "" {
		if parsedCoords, err := parseMainCoordinates(coordinatesJSON.String); err == nil {
			loc.Coordinates = parsedCoords
			log.Printf("Parsed main coordinates for %s: lat=%f, lon=%f", loc.ID, parsedCoords.Lat, parsedCoords.Lon)
		} else {
			log.Printf("Failed to parse main coordinates for %s: %v", loc.ID, err)
		}
	}

	// Parse all pin arrays using unified function
	if landmarksJSON.Valid && landmarksJSON.String != "" {
		loc.Landmarks = parsePinArray(landmarksJSON.String, "landmarks")
		log.Printf("Parsed %d landmarks for %s", len(loc.Landmarks), loc.ID)
	}

	if businessJSON.Valid && businessJSON.String != "" {
		loc.Business = parsePinArray(businessJSON.String, "business")
		log.Printf("Parsed %d business pins for %s", len(loc.Business), loc.ID)
	}

	if hospitalityJSON.Valid && hospitalityJSON.String != "" {
		loc.Hospitality = parsePinArray(hospitalityJSON.String, "hospitality")
		log.Printf("Parsed %d hospitality pins for %s", len(loc.Hospitality), loc.ID)
	}

	if eventsJSON.Valid && eventsJSON.String != "" {
		loc.Events = parsePinArray(eventsJSON.String, "events")
		log.Printf("Parsed %d event pins for %s", len(loc.Events), loc.ID)
	}

	if psaJSON.Valid && psaJSON.String != "" {
		loc.PSA = parsePinArray(psaJSON.String, "psa")
		log.Printf("Parsed %d PSA pins for %s", len(loc.PSA), loc.ID)
	}

	if hotzonesJSON.Valid && hotzonesJSON.String != "" {
		loc.Hotzones = parsePinArray(hotzonesJSON.String, "hotzones")
		log.Printf("Parsed %d hotzone pins for %s", len(loc.Hotzones), loc.ID)
	}

	// Parse results array
	if resultsJSON.Valid && resultsJSON.String != "" {
		loc.Results = parsePinArray(resultsJSON.String, "results")
		log.Printf("Parsed %d result pins for %s", len(loc.Results), loc.ID)
	}

	// Parse sublocations
	if sublocationsJSON.Valid && sublocationsJSON.String != "" {
		var sublocs SublocationsData
		if err := json.Unmarshal([]byte(sublocationsJSON.String), &sublocs); err == nil {
			// Coordinates for sublocations are already parsed correctly by the database function
			if sublocs.CurrentSublocation != nil && sublocs.CurrentSublocation.Coordinates != nil {
				log.Printf("Current sublocation %s coordinates: lat=%f, lon=%f", 
					sublocs.CurrentSublocation.ID,
					sublocs.CurrentSublocation.Coordinates.Lat, 
					sublocs.CurrentSublocation.Coordinates.Lon)
			}
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

// REPORTING AND MOODS FUNTIONS BEYOND HERE 

func (s *AppService) CreateReporting(ctx context.Context, name, info, reportType, createdBy string, coordinates *GeoPoint, parentLocationID, parentSublocationID string) (Reporting, error) {
	coordsJSON, err := json.Marshal(coordinates)
	if err != nil {
		return Reporting{}, fmt.Errorf("failed to marshal coordinates: %w", err)
	}

	var reporting Reporting
	var coordsStr sql.NullString
	var sublocationID sql.NullString
	var createdAt sql.NullString

	err = s.db.QueryRow(ctx,
		"SELECT * FROM create_reporting($1, $2, $3, $4, $5, $6, $7);",
		name, info, reportType, createdBy, string(coordsJSON), parentLocationID, 
		sql.NullString{String: parentSublocationID, Valid: parentSublocationID != ""},
	).Scan(&reporting.ID, &reporting.Name, &reporting.Info, &reporting.Type, 
		&reporting.CreatedBy, &coordsStr, &createdAt, &reporting.ParentLocationID, &sublocationID)

	if err != nil {
		return Reporting{}, fmt.Errorf("failed to create reporting: %w", err)
	}

	if coordsStr.Valid && coordsStr.String != "" {
		if parsedCoords, err := parseMainCoordinates(coordsStr.String); err == nil {
			reporting.Coordinates = parsedCoords
		}
	}

	if createdAt.Valid {
		reporting.CreatedAt = createdAt.String
	}

	if sublocationID.Valid {
		reporting.ParentSublocationID = sublocationID.String
	}

	return reporting, nil
}

func (s *AppService) GetReportingsByLocation(ctx context.Context, locationID string) ([]Reporting, error) {
	rows, err := s.db.Query(ctx, "SELECT * FROM get_reportings_by_location($1);", locationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get reportings: %w", err)
	}
	defer rows.Close()

	var reportings []Reporting
	for rows.Next() {
		var r Reporting
		var coordsStr sql.NullString
		var sublocationID sql.NullString
		var createdAt sql.NullString

		err := rows.Scan(&r.ID, &r.Name, &r.Info, &r.Type, &r.CreatedBy, 
			&coordsStr, &createdAt, &r.ParentLocationID, &sublocationID)
		if err != nil {
			log.Printf("Warning: failed to scan reporting row: %v", err)
			continue
		}

		if coordsStr.Valid && coordsStr.String != "" {
			if parsedCoords, err := parseMainCoordinates(coordsStr.String); err == nil {
				r.Coordinates = parsedCoords
			}
		}

		if createdAt.Valid {
			r.CreatedAt = createdAt.String
		}

		if sublocationID.Valid {
			r.ParentSublocationID = sublocationID.String
		}

		reportings = append(reportings, r)
	}

	return reportings, nil
}

func (s *AppService) DeleteReporting(ctx context.Context, id int, userID string) (bool, error) {
	var deleted bool
	err := s.db.QueryRow(ctx, "SELECT delete_reporting($1, $2);", id, userID).Scan(&deleted)
	if err != nil {
		return false, fmt.Errorf("failed to delete reporting: %w", err)
	}
	return deleted, nil
}

func (s *AppService) EditReporting(ctx context.Context, id int, userID, name, info, reportType string) (Reporting, error) {
	var r Reporting
	var coordsStr sql.NullString
	var sublocationID sql.NullString
	var createdAt sql.NullString

	err := s.db.QueryRow(ctx,
		"SELECT * FROM edit_reporting($1, $2, $3, $4, $5);",
		id, userID, name, info, reportType,
	).Scan(&r.ID, &r.Name, &r.Info, &r.Type, &r.CreatedBy, 
		&coordsStr, &createdAt, &r.ParentLocationID, &sublocationID)

	if err != nil {
		return Reporting{}, fmt.Errorf("failed to edit reporting: %w", err)
	}

	if coordsStr.Valid && coordsStr.String != "" {
		if parsedCoords, err := parseMainCoordinates(coordsStr.String); err == nil {
			r.Coordinates = parsedCoords
		}
	}

	if createdAt.Valid {
		r.CreatedAt = createdAt.String
	}

	if sublocationID.Valid {
		r.ParentSublocationID = sublocationID.String
	}

	return r, nil
}

func (s *AppService) CreateMood(ctx context.Context, name, info, moodType, createdBy string, coordinates *GeoPoint, parentLocationID, parentSublocationID string) (Mood, error) {
	coordsJSON, err := json.Marshal(coordinates)
	if err != nil {
		return Mood{}, fmt.Errorf("failed to marshal coordinates: %w", err)
	}

	var mood Mood
	var coordsStr sql.NullString
	var sublocationID sql.NullString
	var createdAt sql.NullString

	err = s.db.QueryRow(ctx,
		"SELECT * FROM create_mood($1, $2, $3, $4, $5, $6, $7);",
		name, info, moodType, createdBy, string(coordsJSON), parentLocationID,
		sql.NullString{String: parentSublocationID, Valid: parentSublocationID != ""},
	).Scan(&mood.ID, &mood.Name, &mood.Info, &mood.Type, &mood.CreatedBy, 
		&coordsStr, &createdAt, &mood.ParentLocationID, &sublocationID)

	if err != nil {
		return Mood{}, fmt.Errorf("failed to create mood: %w", err)
	}

	if coordsStr.Valid && coordsStr.String != "" {
		if parsedCoords, err := parseMainCoordinates(coordsStr.String); err == nil {
			mood.Coordinates = parsedCoords
		}
	}

	if createdAt.Valid {
		mood.CreatedAt = createdAt.String
	}

	if sublocationID.Valid {
		mood.ParentSublocationID = sublocationID.String
	}

	return mood, nil
}

func (s *AppService) GetMoodsByLocation(ctx context.Context, locationID string) ([]Mood, error) {
	rows, err := s.db.Query(ctx, "SELECT * FROM get_moods_by_location($1);", locationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get moods: %w", err)
	}
	defer rows.Close()

	var moods []Mood
	for rows.Next() {
		var m Mood
		var coordsStr sql.NullString
		var sublocationID sql.NullString
		var createdAt sql.NullString

		err := rows.Scan(&m.ID, &m.Name, &m.Info, &m.Type, &m.CreatedBy,
			&coordsStr, &createdAt, &m.ParentLocationID, &sublocationID)
		if err != nil {
			log.Printf("Warning: failed to scan mood row: %v", err)
			continue
		}

		if coordsStr.Valid && coordsStr.String != "" {
			if parsedCoords, err := parseMainCoordinates(coordsStr.String); err == nil {
				m.Coordinates = parsedCoords
			}
		}

		if createdAt.Valid {
			m.CreatedAt = createdAt.String
		}

		if sublocationID.Valid {
			m.ParentSublocationID = sublocationID.String
		}

		moods = append(moods, m)
	}

	return moods, nil
}

func (s *AppService) DeleteMood(ctx context.Context, id int, userID string) (bool, error) {
	var deleted bool
	err := s.db.QueryRow(ctx, "SELECT delete_mood($1, $2);", id, userID).Scan(&deleted)
	if err != nil {
		return false, fmt.Errorf("failed to delete mood: %w", err)
	}
	return deleted, nil
}

// TAGS FUNCTIONS

func (s *AppService) CreateTag(ctx context.Context, item, createdBy string, coordinates *GeoPoint, parentLocationID, parentSublocationID, tagType string) (Tag, error) {
	coordsJSON, err := json.Marshal(coordinates)
	if err != nil {
		return Tag{}, fmt.Errorf("failed to marshal coordinates: %w", err)
	}

	var tag Tag
	var coordsStr sql.NullString
	var sublocationID sql.NullString
	var createdAt sql.NullString

	err = s.db.QueryRow(ctx,
		"SELECT * FROM create_tag($1, $2, $3, $4, $5, $6);",
		item, string(coordsJSON), createdBy, parentLocationID,
		sql.NullString{String: parentSublocationID, Valid: parentSublocationID != ""}, tagType,
	).Scan(&tag.ID, &tag.Item, &coordsStr, &tag.CreatedBy, 
		&createdAt, &tag.ParentLocationID, &sublocationID, &tag.Type)

	if err != nil {
		return Tag{}, fmt.Errorf("failed to create tag: %w", err)
	}

	if coordsStr.Valid && coordsStr.String != "" {
		if parsedCoords, err := parseMainCoordinates(coordsStr.String); err == nil {
			tag.Coordinates = parsedCoords
		}
	}

	if createdAt.Valid {
		tag.CreatedAt = createdAt.String
	}

	if sublocationID.Valid {
		tag.ParentSublocationID = sublocationID.String
	}

	return tag, nil
}

func (s *AppService) GetTagsByLocation(ctx context.Context, locationID string) ([]Tag, error) {
	rows, err := s.db.Query(ctx, "SELECT * FROM get_tags_by_location($1);", locationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get tags: %w", err)
	}
	defer rows.Close()

	var tags []Tag
	for rows.Next() {
		var t Tag
		var coordsStr sql.NullString
		var sublocationID sql.NullString
		var createdAt sql.NullString

		err := rows.Scan(&t.ID, &t.Item, &coordsStr, &t.CreatedBy,
			&createdAt, &t.ParentLocationID, &sublocationID, &t.Type)
		if err != nil {
			log.Printf("Warning: failed to scan tag row: %v", err)
			continue
		}

		if coordsStr.Valid && coordsStr.String != "" {
			if parsedCoords, err := parseMainCoordinates(coordsStr.String); err == nil {
				t.Coordinates = parsedCoords
			}
		}

		if createdAt.Valid {
			t.CreatedAt = createdAt.String
		}

		if sublocationID.Valid {
			t.ParentSublocationID = sublocationID.String
		}

		tags = append(tags, t)
	}

	return tags, nil
}

func (s *AppService) DeleteTag(ctx context.Context, id int, userID string) (bool, error) {
	var deleted bool
	err := s.db.QueryRow(ctx, "SELECT delete_tag($1, $2);", id, userID).Scan(&deleted)
	if err != nil {
		return false, fmt.Errorf("failed to delete tag: %w", err)
	}
	return deleted, nil
}

func (s *AppService) EditTag(ctx context.Context, id int, userID, item, tagType string) (Tag, error) {
	var t Tag
	var coordsStr sql.NullString
	var sublocationID sql.NullString
	var createdAt sql.NullString

	err := s.db.QueryRow(ctx,
		"SELECT * FROM edit_tag($1, $2, $3, $4);",
		id, userID, item, tagType,
	).Scan(&t.ID, &t.Item, &coordsStr, &t.CreatedBy,
		&createdAt, &t.ParentLocationID, &sublocationID, &t.Type)

	if err != nil {
		return Tag{}, fmt.Errorf("failed to edit tag: %w", err)
	}

	if coordsStr.Valid && coordsStr.String != "" {
		if parsedCoords, err := parseMainCoordinates(coordsStr.String); err == nil {
			t.Coordinates = parsedCoords
		}
	}

	if createdAt.Valid {
		t.CreatedAt = createdAt.String
	}

	if sublocationID.Valid {
		t.ParentSublocationID = sublocationID.String
	}

	return t, nil
}




func (s *AppService) CreatePinForColumn(ctx context.Context, locationID, column string, pinData map[string]interface{}) (map[string]interface{}, error) {
	pinDataJSON, err := json.Marshal(pinData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal pin data: %w", err)
	}

	var result string
	err = s.db.QueryRow(ctx,
		"SELECT create_pin_for_column($1, $2, $3);",
		locationID, column, string(pinDataJSON),
	).Scan(&result)

	if err != nil {
		return nil, fmt.Errorf("failed to create pin: %w", err)
	}

	var resultData map[string]interface{}
	if err := json.Unmarshal([]byte(result), &resultData); err != nil {
		return nil, fmt.Errorf("failed to parse result: %w", err)
	}

	return resultData, nil
}

func (s *AppService) UpdatePinOfColumn(ctx context.Context, locationID, column string, pinIndex int, pinData map[string]interface{}) (map[string]interface{}, error) {
	pinDataJSON, err := json.Marshal(pinData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal pin data: %w", err)
	}

	var result string
	err = s.db.QueryRow(ctx,
		"SELECT update_pin_of_column($1, $2, $3, $4);",
		locationID, column, pinIndex, string(pinDataJSON),
	).Scan(&result)

	if err != nil {
		return nil, fmt.Errorf("failed to update pin: %w", err)
	}

	var resultData map[string]interface{}
	if err := json.Unmarshal([]byte(result), &resultData); err != nil {
		return nil, fmt.Errorf("failed to parse result: %w", err)
	}

	return resultData, nil
}

func (s *AppService) DeletePinOfColumn(ctx context.Context, locationID, column string, pinIndex int) (bool, error) {
	var deleted bool
	err := s.db.QueryRow(ctx,
		"SELECT delete_pin_of_column($1, $2, $3);",
		locationID, column, pinIndex,
	).Scan(&deleted)

	if err != nil {
		return false, fmt.Errorf("failed to delete pin: %w", err)
	}

	return deleted, nil
}

func (s *AppService) GetPinsForColumn(ctx context.Context, locationID, column string) ([]map[string]interface{}, error) {
	var result string
	err := s.db.QueryRow(ctx,
		"SELECT get_pins_for_column($1, $2);",
		locationID, column,
	).Scan(&result)

	if err != nil {
		return nil, fmt.Errorf("failed to get pins: %w", err)
	}

	var pins []map[string]interface{}
	if err := json.Unmarshal([]byte(result), &pins); err != nil {
		return nil, fmt.Errorf("failed to parse pins: %w", err)
	}

	return pins, nil
}


// ----------------------- SUBLOCAITONS /////////////////////////////////////// 
func (s *AppService) CreateSublocation(ctx context.Context, id, name, info, svgPin, parentLocationID, zoom, sublType string, coordinates *GeoPoint) (SublocationData, error) {
	coordsJSON, err := json.Marshal(coordinates)
	if err != nil {
		return SublocationData{}, fmt.Errorf("failed to marshal coordinates: %w", err)
	}

	var sublocation SublocationData
	var coordsStr sql.NullString
	var infoStr, svgPinStr, zoomStr sql.NullString

	err = s.db.QueryRow(ctx,
		"SELECT * FROM create_sublocation($1, $2, $3, $4, $5, $6, $7, $8);",
		id, name, 
		sql.NullString{String: info, Valid: info != ""}, 
		string(coordsJSON), 
		sql.NullString{String: svgPin, Valid: svgPin != ""}, 
		parentLocationID,
		sql.NullString{String: zoom, Valid: zoom != ""},
		sublType,
	).Scan(&sublocation.ID, &sublocation.Name, &infoStr, &coordsStr, 
		&svgPinStr, &sublocation.ParentLocationID, &zoomStr, &sublocation.Type)

	if err != nil {
		return SublocationData{}, fmt.Errorf("failed to create sublocation: %w", err)
	}

	if coordsStr.Valid && coordsStr.String != "" {
		if parsedCoords, err := parseMainCoordinates(coordsStr.String); err == nil {
			sublocation.Coordinates = parsedCoords
		}
	}

	if infoStr.Valid {
		sublocation.Info = infoStr.String
	}
	if svgPinStr.Valid {
		sublocation.SvgPin = svgPinStr.String
	}
	if zoomStr.Valid {
		sublocation.Zoom = zoomStr.String
	}

	return sublocation, nil
}

func (s *AppService) GetSublocationsByLocation(ctx context.Context, parentLocationID string) ([]SublocationData, error) {
	rows, err := s.db.Query(ctx, "SELECT * FROM get_sublocations_by_location($1);", parentLocationID)
	if err != nil {
		return []SublocationData{}, fmt.Errorf("failed to get sublocations: %w", err)
	}
	defer rows.Close()

	var sublocations []SublocationData = []SublocationData{} // Initialize as empty slice instead of nil
	for rows.Next() {
		var subloc SublocationData
		var coordsStr sql.NullString
		var infoStr, svgPinStr, zoomStr sql.NullString

		err := rows.Scan(&subloc.ID, &subloc.Name, &infoStr, &coordsStr, 
			&svgPinStr, &subloc.ParentLocationID, &zoomStr, &subloc.Type)
		if err != nil {
			log.Printf("Warning: failed to scan sublocation row: %v", err)
			continue
		}

		if coordsStr.Valid && coordsStr.String != "" {
			if parsedCoords, err := parseMainCoordinates(coordsStr.String); err == nil {
				subloc.Coordinates = parsedCoords
			} else {
				log.Printf("Warning: failed to parse coordinates for sublocation %s: %v", subloc.ID, err)
			}
		}

		if infoStr.Valid {
			subloc.Info = infoStr.String
		}
		if svgPinStr.Valid {
			subloc.SvgPin = svgPinStr.String
		}
		if zoomStr.Valid {
			subloc.Zoom = zoomStr.String
		}

		sublocations = append(sublocations, subloc)
	}

	if err := rows.Err(); err != nil {
		log.Printf("Error iterating sublocation rows: %v", err)
		return []SublocationData{}, fmt.Errorf("error iterating rows: %w", err)
	}

	return sublocations, nil
}

func (s *AppService) UpdateSublocation(ctx context.Context, id, name, info, svgPin, zoom, sublType string) (SublocationData, error) {
	var subloc SublocationData
	var coordsStr sql.NullString
	var infoStr, svgPinStr, zoomStr sql.NullString

	err := s.db.QueryRow(ctx,
		"SELECT * FROM update_sublocation($1, $2, $3, $4, $5, $6);",
		id, name, 
		sql.NullString{String: info, Valid: info != ""}, 
		sql.NullString{String: svgPin, Valid: svgPin != ""}, 
		sql.NullString{String: zoom, Valid: zoom != ""}, 
		sublType,
	).Scan(&subloc.ID, &subloc.Name, &infoStr, &coordsStr, 
		&svgPinStr, &subloc.ParentLocationID, &zoomStr, &subloc.Type)

	if err != nil {
		return SublocationData{}, fmt.Errorf("failed to update sublocation: %w", err)
	}

	if coordsStr.Valid && coordsStr.String != "" {
		if parsedCoords, err := parseMainCoordinates(coordsStr.String); err == nil {
			subloc.Coordinates = parsedCoords
		}
	}

	if infoStr.Valid {
		subloc.Info = infoStr.String
	}
	if svgPinStr.Valid {
		subloc.SvgPin = svgPinStr.String
	}
	if zoomStr.Valid {
		subloc.Zoom = zoomStr.String
	}

	return subloc, nil
}

func (s *AppService) DeleteSublocation(ctx context.Context, id string) (bool, error) {
	var deleted bool
	err := s.db.QueryRow(ctx, "SELECT delete_sublocation($1);", id).Scan(&deleted)
	if err != nil {
		return false, fmt.Errorf("failed to delete sublocation: %w", err)
	}
	return deleted, nil
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

// Add these cases to the handleRpcRequest switch statement
	case "createReporting":
		var params struct {
			Name                string   `json:"name"`
			Info                string   `json:"info"`
			Type                string   `json:"type"`
			Coordinates         GeoPoint `json:"coordinates"`
			ParentLocationID    string   `json:"parentLocationId"`
			ParentSublocationID string   `json:"parentSublocationId"`
		}
		if err := json.Unmarshal(req.Params, &params); err != nil || params.Name == "" {
			processingError = fmt.Errorf("missing or invalid parameters")
			break
		}
		resultData, processingError = service.CreateReporting(ctx, params.Name, params.Info, params.Type, uid, &params.Coordinates, params.ParentLocationID, params.ParentSublocationID)

	case "getReportingsByLocation":
		var params struct {
			LocationID string `json:"locationId"`
		}
		if err := json.Unmarshal(req.Params, &params); err != nil || params.LocationID == "" {
			processingError = fmt.Errorf("missing or invalid 'locationId' parameter")
			break
		}
		resultData, processingError = service.GetReportingsByLocation(ctx, params.LocationID)

	case "deleteReporting":
		var params struct {
			ID int `json:"id"`
		}
		if err := json.Unmarshal(req.Params, &params); err != nil {
			processingError = fmt.Errorf("missing or invalid 'id' parameter")
			break
		}
		resultData, processingError = service.DeleteReporting(ctx, params.ID, uid)

	case "editReporting":
		var params struct {
			ID   int    `json:"id"`
			Name string `json:"name"`
			Info string `json:"info"`
			Type string `json:"type"`
		}
		if err := json.Unmarshal(req.Params, &params); err != nil {
			processingError = fmt.Errorf("missing or invalid parameters")
			break
		}
		resultData, processingError = service.EditReporting(ctx, params.ID, uid, params.Name, params.Info, params.Type)

	case "createMood":
		var params struct {
			Name                string   `json:"name"`
			Info                string   `json:"info"`
			Type                string   `json:"type"`
			Coordinates         GeoPoint `json:"coordinates"`
			ParentLocationID    string   `json:"parentLocationId"`
			ParentSublocationID string   `json:"parentSublocationId"`
		}
		if err := json.Unmarshal(req.Params, &params); err != nil || params.Name == "" {
			processingError = fmt.Errorf("missing or invalid parameters")
			break
		}
		resultData, processingError = service.CreateMood(ctx, params.Name, params.Info, params.Type, uid, &params.Coordinates, params.ParentLocationID, params.ParentSublocationID)

	case "getMoodsByLocation":
		var params struct {
			LocationID string `json:"locationId"`
		}
		if err := json.Unmarshal(req.Params, &params); err != nil || params.LocationID == "" {
			processingError = fmt.Errorf("missing or invalid 'locationId' parameter")
			break
		}
		resultData, processingError = service.GetMoodsByLocation(ctx, params.LocationID)

	case "deleteMood":
		var params struct {
			ID int `json:"id"`
		}
		if err := json.Unmarshal(req.Params, &params); err != nil {
			processingError = fmt.Errorf("missing or invalid 'id' parameter")
			break
		}
		resultData, processingError = service.DeleteMood(ctx, params.ID, uid)

	case "createTag":
		var params struct {
			Item                string   `json:"item"`
			Coordinates         GeoPoint `json:"coordinates"`
			ParentLocationID    string   `json:"parentLocationId"`
			ParentSublocationID string   `json:"parentSublocationId"`
			Type                string   `json:"type"`
		}
		if err := json.Unmarshal(req.Params, &params); err != nil || params.Item == "" {
			processingError = fmt.Errorf("missing or invalid parameters")
			break
		}
		resultData, processingError = service.CreateTag(ctx, params.Item, uid, &params.Coordinates, params.ParentLocationID, params.ParentSublocationID, params.Type)

	case "getTagsByLocation":
		var params struct {
			LocationID string `json:"locationId"`
		}
		if err := json.Unmarshal(req.Params, &params); err != nil || params.LocationID == "" {
			processingError = fmt.Errorf("missing or invalid 'locationId' parameter")
			break
		}
		resultData, processingError = service.GetTagsByLocation(ctx, params.LocationID)

	case "deleteTag":
		var params struct {
			ID int `json:"id"`
		}
		if err := json.Unmarshal(req.Params, &params); err != nil {
			processingError = fmt.Errorf("missing or invalid 'id' parameter")
			break
		}
		resultData, processingError = service.DeleteTag(ctx, params.ID, uid)

	case "editTag":
		var params struct {
			ID   int    `json:"id"`
			Item string `json:"item"`
			Type string `json:"type"`
		}
		if err := json.Unmarshal(req.Params, &params); err != nil {
			processingError = fmt.Errorf("missing or invalid parameters")
			break
		}
		resultData, processingError = service.EditTag(ctx, params.ID, uid, params.Item, params.Type)


	case "createPinForColumn":
		var params struct {
			LocationID string                 `json:"locationId"`
			Column     string                 `json:"column"`
			PinData    map[string]interface{} `json:"pinData"`
		}
		if err := json.Unmarshal(req.Params, &params); err != nil || params.LocationID == "" || params.Column == "" {
			processingError = fmt.Errorf("missing or invalid parameters")
			break
		}
		resultData, processingError = service.CreatePinForColumn(ctx, params.LocationID, params.Column, params.PinData)

	case "updatePinOfColumn":
		var params struct {
			LocationID string                 `json:"locationId"`
			Column     string                 `json:"column"`
			PinIndex   int                    `json:"pinIndex"`
			PinData    map[string]interface{} `json:"pinData"`
		}
		if err := json.Unmarshal(req.Params, &params); err != nil || params.LocationID == "" || params.Column == "" {
			processingError = fmt.Errorf("missing or invalid parameters")
			break
		}
		resultData, processingError = service.UpdatePinOfColumn(ctx, params.LocationID, params.Column, params.PinIndex, params.PinData)

	case "deletePinOfColumn":
		var params struct {
			LocationID string `json:"locationId"`
			Column     string `json:"column"`
			PinIndex   int    `json:"pinIndex"`
		}
		if err := json.Unmarshal(req.Params, &params); err != nil || params.LocationID == "" || params.Column == "" {
			processingError = fmt.Errorf("missing or invalid parameters")
			break
		}
		resultData, processingError = service.DeletePinOfColumn(ctx, params.LocationID, params.Column, params.PinIndex)

	case "getPinsForColumn":
		var params struct {
			LocationID string `json:"locationId"`
			Column     string `json:"column"`
		}
		if err := json.Unmarshal(req.Params, &params); err != nil || params.LocationID == "" || params.Column == "" {
			processingError = fmt.Errorf("missing or invalid parameters")
			break
		}
		resultData, processingError = service.GetPinsForColumn(ctx, params.LocationID, params.Column)





// -------------------------------------SUBLOCATIONS ------------------------------------

	case "createSublocation":
		var params struct {
			ID               string   `json:"id"`
			Name             string   `json:"name"`
			Info             string   `json:"info"`
			SvgPin           string   `json:"svgpin"`
			ParentLocationID string   `json:"parentLocationId"`
			Zoom             string   `json:"zoom"`
			Type             string   `json:"type"`
			Coordinates      GeoPoint `json:"coordinates"`
		}
		if err := json.Unmarshal(req.Params, &params); err != nil || params.ID == "" || params.Name == "" || params.ParentLocationID == "" {
			processingError = fmt.Errorf("missing or invalid parameters")
			break
		}
		resultData, processingError = service.CreateSublocation(ctx, params.ID, params.Name, params.Info, params.SvgPin, params.ParentLocationID, params.Zoom, params.Type, &params.Coordinates)

// In your get_sublocations_by_location case in handleRpcRequest
case "getSublocationsByLocation":
    var params struct {
        ParentLocationID string `json:"parentLocationId"`
    }
    if err := json.Unmarshal(req.Params, &params); err != nil || params.ParentLocationID == "" {
        processingError = fmt.Errorf("missing or invalid 'parentLocationId' parameter")
        break
    }
    resultData, processingError = service.GetSublocationsByLocation(ctx, params.ParentLocationID)
    
    // Ensure we always return an array, not nil
    if sublocations, ok := resultData.([]SublocationData); ok {
        if sublocations == nil {
            resultData = []SublocationData{}
        }
    }

	case "updateSublocation":
		var params struct {
			ID     string `json:"id"`
			Name   string `json:"name"`
			Info   string `json:"info"`
			SvgPin string `json:"svgpin"`
			Zoom   string `json:"zoom"`
			Type   string `json:"type"`
		}
		if err := json.Unmarshal(req.Params, &params); err != nil || params.ID == "" {
			processingError = fmt.Errorf("missing or invalid parameters")
			break
		}
		resultData, processingError = service.UpdateSublocation(ctx, params.ID, params.Name, params.Info, params.SvgPin, params.Zoom, params.Type)

	case "deleteSublocation":
		var params struct {
			ID string `json:"id"`
		}
		if err := json.Unmarshal(req.Params, &params); err != nil || params.ID == "" {
			processingError = fmt.Errorf("missing or invalid 'id' parameter")
			break
		}
		resultData, processingError = service.DeleteSublocation(ctx, params.ID)
		// -------------------------- SUBLOCTIONS -------------------------------------------

		
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
