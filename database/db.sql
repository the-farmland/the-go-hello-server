create table public.tags (
  id serial not null,
  item text not null,
  coordinates jsonb not null,
  created_by text not null,
  created_at timestamp with time zone null default now(),
  parent_location_id text not null,
  parent_sublocation_id text null,
  type text not null,
  constraint tags_pkey primary key (id),
  constraint fk_tag_location foreign KEY (parent_location_id) references locations (id) on delete CASCADE
) TABLESPACE pg_default;

create index IF not exists idx_tags_location on public.tags using btree (parent_location_id) TABLESPACE pg_default;

create index IF not exists idx_tags_created_by on public.tags using btree (created_by) TABLESPACE pg_default;

create index IF not exists idx_tags_created_at on public.tags using btree (created_at) TABLESPACE pg_default;

create index IF not exists idx_tags_type on public.tags using btree (type) TABLESPACE pg_default;

create table public.sublocations (
  id text not null,
  name text not null,
  info text null,
  coordinates jsonb null,
  svgpin text null,
  parent_location_id text not null,
  zoom text null,
  constraint sublocations_pkey primary key (id),
  constraint fk_parent_location foreign KEY (parent_location_id) references locations (id) on delete CASCADE
) TABLESPACE pg_default;

create index IF not exists idx_sublocations_parent on public.sublocations using btree (parent_location_id) TABLESPACE pg_default;

create index IF not exists idx_sublocations_search on public.sublocations using gin (
  to_tsvector(
    'english'::regconfig,
    (
      (COALESCE(name, ''::text) || ' '::text) || COALESCE(info, ''::text)
    )
  )
) TABLESPACE pg_default;


create table public.reportings (
  id serial not null,
  name text not null,
  info text null,
  type text not null,
  created_by text not null,
  coordinates jsonb not null,
  created_at timestamp with time zone null default now(),
  parent_location_id text not null,
  parent_sublocation_id text null,
  constraint reportings_pkey primary key (id),
  constraint fk_reporting_location foreign KEY (parent_location_id) references locations (id) on delete CASCADE
) TABLESPACE pg_default;

create index IF not exists idx_reportings_location on public.reportings using btree (parent_location_id) TABLESPACE pg_default;

create index IF not exists idx_reportings_created_by on public.reportings using btree (created_by) TABLESPACE pg_default;

create index IF not exists idx_reportings_created_at on public.reportings using btree (created_at) TABLESPACE pg_default;



create table public.moods (
  id serial not null,
  name text not null,
  info text null,
  type text not null,
  created_by text not null,
  coordinates jsonb not null,
  created_at timestamp with time zone null default now(),
  parent_location_id text not null,
  parent_sublocation_id text null,
  constraint moods_pkey primary key (id),
  constraint fk_mood_location foreign KEY (parent_location_id) references locations (id) on delete CASCADE
) TABLESPACE pg_default;

create index IF not exists idx_moods_location on public.moods using btree (parent_location_id) TABLESPACE pg_default;

create index IF not exists idx_moods_created_by on public.moods using btree (created_by) TABLESPACE pg_default;


create table public.locations (
  id text not null,
  name text not null,
  country text not null,
  state text null,
  description text null,
  svg_link text null,
  rating numeric(3, 1) null default 0.0,
  map_main_image text null,
  map_cover_image text null,
  main_background_image text null,
  map_full_address text null,
  map_png_link text null,
  boards jsonb null,
  coordinates jsonb null,
  landmarks jsonb null,
  parent_location_id text null,
  business jsonb null,
  hospitality jsonb null,
  events jsonb null,
  psa jsonb null,
  geojson text null,
  hotzones jsonb null,
  zoom text null,
  results jsonb null,
  constraint locations_pkey primary key (id)
) TABLESPACE pg_default;

create index IF not exists idx_locations_search on public.locations using gin (
  to_tsvector(
    'english'::regconfig,
    (
      (
        (
          (
            (
              (COALESCE(name, ''::text) || ' '::text) || COALESCE(country, ''::text)
            ) || ' '::text
          ) || COALESCE(state, ''::text)
        ) || ' '::text
      ) || COALESCE(map_full_address, ''::text)
    )
  )
) TABLESPACE pg_default;





-- Replace the existing search_locations with this enhanced version

CREATE OR REPLACE FUNCTION create_pin_for_column(location_id text, column_name text, pin_data jsonb)
RETURNS jsonb AS $$
DECLARE
  pin_with_id jsonb;
  new_pins jsonb;
  pin_id text;
BEGIN
  -- Generate a unique ID for the pin if not provided
  pin_id := COALESCE(pin_data->>'id', gen_random_uuid()::text);
  
  -- Add id to pin_data
  pin_with_id := pin_data || jsonb_build_object('id', pin_id);
  
  -- Add pinImg field if not present
  IF NOT (pin_with_id ? 'pinImg') THEN
    pin_with_id := pin_with_id || jsonb_build_object('pinImg', '');
  END IF;

  -- Dynamically update the column
  EXECUTE format('
    UPDATE locations 
    SET %I = COALESCE(%I, ''[]''::jsonb) || $1
    WHERE id = $2
    RETURNING %I
  ', column_name, column_name, column_name)
  USING jsonb_build_array(pin_with_id), location_id
  INTO new_pins;

  RETURN jsonb_build_object(
    'success', true,
    'pin_id', pin_id,
    'message', 'Pin created successfully'
  );
END;
$$ LANGUAGE plpgsql;

-- 5. Update update_pin_of_column function
CREATE OR REPLACE FUNCTION update_pin_of_column(location_id text, column_name text, pin_index int, pin_data jsonb)
RETURNS jsonb AS $$
DECLARE
  updated_pins jsonb;
  existing_pin jsonb;
  pin_id text;
BEGIN
  -- Get the existing pin to preserve its id
  EXECUTE format('
    SELECT ($1 ->> %L) FROM locations WHERE id = $2
  ', pin_index)
  INTO existing_pin
  USING column_name, location_id;

  -- Use existing pin_id or generate new one
  pin_id := COALESCE(existing_pin->>'id', md5(pin_data->>'name' || '-' || pin_data->>'pinLink' || '-' || pin_index::text));

  -- Merge pin_data with id
  pin_data := pin_data || jsonb_build_object('id', pin_id);

  -- Add pinImg field if not present
  IF NOT (pin_data ? 'pinImg') THEN
    pin_data := pin_data || jsonb_build_object('pinImg', '');
  END IF;

  EXECUTE format('
    UPDATE locations
    SET %I = jsonb_set(%I, ARRAY[%L], $1)
    WHERE id = $2
  ', column_name, column_name, pin_index)
  USING pin_data, location_id;

  RETURN jsonb_build_object(
    'success', true,
    'pin_id', pin_id,
    'message', 'Pin updated successfully'
  );
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION get_pins_for_column(location_id text, column_name text)
RETURNS jsonb AS $$
DECLARE
  valid_columns text[] := ARRAY['landmarks', 'business', 'hospitality', 'events', 'psa', 'hotzones', 'results', 'drivers', 'walker', 'transit', 'utilities', 'manmade', 'natural', 'municipal'];
  result jsonb;
BEGIN
  -- Validate column name
  IF NOT column_name = ANY(valid_columns) THEN
    RAISE EXCEPTION 'Invalid column name. Must be one of: %', array_to_string(valid_columns, ', ');
  END IF;

  -- Dynamically get the pins from the specified column
  EXECUTE format('SELECT COALESCE(%I, ''[]''::jsonb) FROM locations WHERE id = %L', column_name, location_id)
  INTO result;

  RETURN COALESCE(result, '[]'::jsonb);
END;
$$ LANGUAGE plpgsql;





-- Add new columns to locations table
ALTER TABLE public.locations ADD COLUMN IF NOT EXISTS drivers jsonb NULL;
ALTER TABLE public.locations ADD COLUMN IF NOT EXISTS walker jsonb NULL;
ALTER TABLE public.locations ADD COLUMN IF NOT EXISTS transit jsonb NULL;
ALTER TABLE public.locations ADD COLUMN IF NOT EXISTS utilities jsonb NULL;
ALTER TABLE public.locations ADD COLUMN IF NOT EXISTS manmade jsonb NULL;
ALTER TABLE public.locations ADD COLUMN IF NOT EXISTS natural jsonb NULL;
ALTER TABLE public.locations ADD COLUMN IF NOT EXISTS municipal jsonb NULL;




-- Create moods table
CREATE TABLE IF NOT EXISTS public.moods (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    info TEXT,
    type TEXT NOT NULL,
    created_by TEXT NOT NULL,
    coordinates JSONB NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    parent_location_id TEXT NOT NULL,
    parent_sublocation_id TEXT,
    CONSTRAINT fk_mood_location FOREIGN KEY (parent_location_id) REFERENCES locations (id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_moods_location ON public.moods USING btree (parent_location_id);
CREATE INDEX IF NOT EXISTS idx_moods_created_by ON public.moods USING btree (created_by);

-- Create reportings table
CREATE TABLE IF NOT EXISTS public.reportings (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    info TEXT,
    type TEXT NOT NULL,
    created_by TEXT NOT NULL,
    coordinates JSONB NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    parent_location_id TEXT NOT NULL,
    parent_sublocation_id TEXT,
    CONSTRAINT fk_reporting_location FOREIGN KEY (parent_location_id) REFERENCES locations (id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_reportings_location ON public.reportings USING btree (parent_location_id);
CREATE INDEX IF NOT EXISTS idx_reportings_created_by ON public.reportings USING btree (created_by);
CREATE INDEX IF NOT EXISTS idx_reportings_created_at ON public.reportings USING btree (created_at);

-- Function to purge old reportings (older than 24 hours)
CREATE OR REPLACE FUNCTION purge_old_reportings()
RETURNS INTEGER
LANGUAGE plpgsql
AS $$
DECLARE
    deleted_count INTEGER;
BEGIN
    DELETE FROM reportings
    WHERE created_at < NOW() - INTERVAL '24 hours';
    
    GET DIAGNOSTICS deleted_count = ROW_COUNT;
    RETURN deleted_count;
END;
$$;

-- Function to create a new reporting
CREATE OR REPLACE FUNCTION create_reporting(
    p_name TEXT,
    p_info TEXT,
    p_type TEXT,
    p_created_by TEXT,
    p_coordinates JSONB,
    p_parent_location_id TEXT,
    p_parent_sublocation_id TEXT DEFAULT NULL
)
RETURNS TABLE(
    id INTEGER,
    name TEXT,
    info TEXT,
    type TEXT,
    created_by TEXT,
    coordinates JSONB,
    created_at TIMESTAMP WITH TIME ZONE,
    parent_location_id TEXT,
    parent_sublocation_id TEXT
)
LANGUAGE plpgsql
AS $$
BEGIN
    RETURN QUERY
    INSERT INTO reportings (name, info, type, created_by, coordinates, parent_location_id, parent_sublocation_id)
    VALUES (p_name, p_info, p_type, p_created_by, p_coordinates, p_parent_location_id, p_parent_sublocation_id)
    RETURNING reportings.id, reportings.name, reportings.info, reportings.type, reportings.created_by, 
              reportings.coordinates, reportings.created_at, reportings.parent_location_id, reportings.parent_sublocation_id;
END;
$$;

-- Function to get reportings by location
CREATE OR REPLACE FUNCTION get_reportings_by_location(p_location_id TEXT)
RETURNS TABLE(
    id INTEGER,
    name TEXT,
    info TEXT,
    type TEXT,
    created_by TEXT,
    coordinates JSONB,
    created_at TIMESTAMP WITH TIME ZONE,
    parent_location_id TEXT,
    parent_sublocation_id TEXT
)
LANGUAGE plpgsql
AS $$
BEGIN
    RETURN QUERY
    SELECT r.id, r.name, r.info, r.type, r.created_by, r.coordinates, 
           r.created_at, r.parent_location_id, r.parent_sublocation_id
    FROM reportings r
    WHERE r.parent_location_id = p_location_id
      AND r.created_at > NOW() - INTERVAL '24 hours'
    ORDER BY r.created_at DESC;
END;
$$;

-- Function to delete a reporting
CREATE OR REPLACE FUNCTION delete_reporting(p_id INTEGER, p_user_id TEXT)
RETURNS BOOLEAN
LANGUAGE plpgsql
AS $$
DECLARE
    deleted BOOLEAN;
BEGIN
    DELETE FROM reportings
    WHERE id = p_id AND created_by = p_user_id;
    
    deleted := FOUND;
    RETURN deleted;
END;
$$;

-- Function to edit a reporting
CREATE OR REPLACE FUNCTION edit_reporting(
    p_id INTEGER,
    p_user_id TEXT,
    p_name TEXT,
    p_info TEXT,
    p_type TEXT
)
RETURNS TABLE(
    id INTEGER,
    name TEXT,
    info TEXT,
    type TEXT,
    created_by TEXT,
    coordinates JSONB,
    created_at TIMESTAMP WITH TIME ZONE,
    parent_location_id TEXT,
    parent_sublocation_id TEXT
)
LANGUAGE plpgsql
AS $$
BEGIN
    RETURN QUERY
    UPDATE reportings
    SET name = p_name,
        info = p_info,
        type = p_type
    WHERE reportings.id = p_id AND created_by = p_user_id
    RETURNING reportings.id, reportings.name, reportings.info, reportings.type, reportings.created_by,
              reportings.coordinates, reportings.created_at, reportings.parent_location_id, reportings.parent_sublocation_id;
END;
$$;

-- Function to create a new mood
CREATE OR REPLACE FUNCTION create_mood(
    p_name TEXT,
    p_info TEXT,
    p_type TEXT,
    p_created_by TEXT,
    p_coordinates JSONB,
    p_parent_location_id TEXT,
    p_parent_sublocation_id TEXT DEFAULT NULL
)
RETURNS TABLE(
    id INTEGER,
    name TEXT,
    info TEXT,
    type TEXT,
    created_by TEXT,
    coordinates JSONB,
    created_at TIMESTAMP WITH TIME ZONE,
    parent_location_id TEXT,
    parent_sublocation_id TEXT
)
LANGUAGE plpgsql
AS $$
BEGIN
    RETURN QUERY
    INSERT INTO moods (name, info, type, created_by, coordinates, parent_location_id, parent_sublocation_id)
    VALUES (p_name, p_info, p_type, p_created_by, p_coordinates, p_parent_location_id, p_parent_sublocation_id)
    RETURNING moods.id, moods.name, moods.info, moods.type, moods.created_by,
              moods.coordinates, moods.created_at, moods.parent_location_id, moods.parent_sublocation_id;
END;
$$;

-- Function to get moods by location
CREATE OR REPLACE FUNCTION get_moods_by_location(p_location_id TEXT)
RETURNS TABLE(
    id INTEGER,
    name TEXT,
    info TEXT,
    type TEXT,
    created_by TEXT,
    coordinates JSONB,
    created_at TIMESTAMP WITH TIME ZONE,
    parent_location_id TEXT,
    parent_sublocation_id TEXT
)
LANGUAGE plpgsql
AS $$
BEGIN
    RETURN QUERY
    SELECT m.id, m.name, m.info, m.type, m.created_by, m.coordinates,
           m.created_at, m.parent_location_id, m.parent_sublocation_id
    FROM moods m
    WHERE m.parent_location_id = p_location_id
    ORDER BY m.created_at DESC;
END;
$$;

-- Function to delete a mood
CREATE OR REPLACE FUNCTION delete_mood(p_id INTEGER, p_user_id TEXT)
RETURNS BOOLEAN
LANGUAGE plpgsql
AS $$
DECLARE
    deleted BOOLEAN;
BEGIN
    DELETE FROM moods
    WHERE id = p_id AND created_by = p_user_id;
    
    deleted := FOUND;
    RETURN deleted;
END;
$$;



-- Create tags table
CREATE TABLE IF NOT EXISTS public.tags (
    id SERIAL PRIMARY KEY,
    item TEXT NOT NULL,
    coordinates JSONB NOT NULL,
    created_by TEXT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    parent_location_id TEXT NOT NULL,
    parent_sublocation_id TEXT,
    type TEXT NOT NULL,
    CONSTRAINT fk_tag_location FOREIGN KEY (parent_location_id) REFERENCES locations (id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_tags_location ON public.tags USING btree (parent_location_id);
CREATE INDEX IF NOT EXISTS idx_tags_created_by ON public.tags USING btree (created_by);
CREATE INDEX IF NOT EXISTS idx_tags_created_at ON public.tags USING btree (created_at);
CREATE INDEX IF NOT EXISTS idx_tags_type ON public.tags USING btree (type);

-- Function to create a new tag
CREATE OR REPLACE FUNCTION create_tag(
    p_item TEXT,
    p_coordinates JSONB,
    p_created_by TEXT,
    p_parent_location_id TEXT,
    p_parent_sublocation_id TEXT DEFAULT NULL,
    p_type TEXT DEFAULT 'others'
)
RETURNS TABLE(
    id INTEGER,
    item TEXT,
    coordinates JSONB,
    created_by TEXT,
    created_at TIMESTAMP WITH TIME ZONE,
    parent_location_id TEXT,
    parent_sublocation_id TEXT,
    type TEXT
)
LANGUAGE plpgsql
AS $$
BEGIN
    RETURN QUERY
    INSERT INTO tags (item, coordinates, created_by, parent_location_id, parent_sublocation_id, type)
    VALUES (p_item, p_coordinates, p_created_by, p_parent_location_id, p_parent_sublocation_id, p_type)
    RETURNING tags.id, tags.item, tags.coordinates, tags.created_by, 
              tags.created_at, tags.parent_location_id, tags.parent_sublocation_id, tags.type;
END;
$$;

-- Function to get tags by location
CREATE OR REPLACE FUNCTION get_tags_by_location(p_location_id TEXT)
RETURNS TABLE(
    id INTEGER,
    item TEXT,
    coordinates JSONB,
    created_by TEXT,
    created_at TIMESTAMP WITH TIME ZONE,
    parent_location_id TEXT,
    parent_sublocation_id TEXT,
    type TEXT
)
LANGUAGE plpgsql
AS $$
BEGIN
    RETURN QUERY
    SELECT t.id, t.item, t.coordinates, t.created_by, t.created_at, 
           t.parent_location_id, t.parent_sublocation_id, t.type
    FROM tags t
    WHERE t.parent_location_id = p_location_id
    ORDER BY t.created_at DESC;
END;
$$;

-- Function to delete a tag
CREATE OR REPLACE FUNCTION delete_tag(p_id INTEGER, p_user_id TEXT)
RETURNS BOOLEAN
LANGUAGE plpgsql
AS $$
DECLARE
    deleted BOOLEAN;
BEGIN
    DELETE FROM tags
    WHERE id = p_id AND created_by = p_user_id;
    
    deleted := FOUND;
    RETURN deleted;
END;
$$;

-- Function to edit a tag
CREATE OR REPLACE FUNCTION edit_tag(
    p_id INTEGER,
    p_user_id TEXT,
    p_item TEXT,
    p_type TEXT
)
RETURNS TABLE(
    id INTEGER,
    item TEXT,
    coordinates JSONB,
    created_by TEXT,
    created_at TIMESTAMP WITH TIME ZONE,
    parent_location_id TEXT,
    parent_sublocation_id TEXT,
    type TEXT
)
LANGUAGE plpgsql
AS $$
BEGIN
    RETURN QUERY
    UPDATE tags
    SET item = p_item,
        type = p_type
    WHERE tags.id = p_id AND created_by = p_user_id
    RETURNING tags.id, tags.item, tags.coordinates, tags.created_by,
              tags.created_at, tags.parent_location_id, tags.parent_sublocation_id, tags.type;
END;
$$;

-- Add type column to sublocations table
ALTER TABLE public.sublocations ADD COLUMN IF NOT EXISTS type text DEFAULT 'island';

-- Create index for the new type column
CREATE INDEX IF NOT EXISTS idx_sublocations_type ON public.sublocations USING btree (type);

-- Function to create a new sublocation
CREATE OR REPLACE FUNCTION create_sublocation(
    p_id TEXT,
    p_name TEXT,
    p_info TEXT,
    p_coordinates JSONB,
    p_svgpin TEXT,
    p_parent_location_id TEXT,
    p_zoom TEXT DEFAULT NULL,
    p_type TEXT DEFAULT 'island'
)
RETURNS TABLE(
    id TEXT,
    name TEXT,
    info TEXT,
    coordinates JSONB,
    svgpin TEXT,
    parent_location_id TEXT,
    zoom TEXT,
    type TEXT
)
LANGUAGE plpgsql
AS $$
BEGIN
    RETURN QUERY
    INSERT INTO sublocations (id, name, info, coordinates, svgpin, parent_location_id, zoom, type)
    VALUES (p_id, p_name, p_info, p_coordinates, p_svgpin, p_parent_location_id, p_zoom, p_type)
    RETURNING sublocations.id, sublocations.name, sublocations.info, sublocations.coordinates, 
              sublocations.svgpin, sublocations.parent_location_id, sublocations.zoom, sublocations.type;
END;
$$;

-- Function to get sublocations by parent location
CREATE OR REPLACE FUNCTION get_sublocations_by_location(p_parent_location_id TEXT)
RETURNS TABLE(
    id TEXT,
    name TEXT,
    info TEXT,
    coordinates JSONB,
    svgpin TEXT,
    parent_location_id TEXT,
    zoom TEXT,
    type TEXT
)
LANGUAGE plpgsql
AS $$
BEGIN
    RETURN QUERY
    SELECT s.id, s.name, s.info, s.coordinates, s.svgpin, s.parent_location_id, s.zoom, s.type
    FROM sublocations s
    WHERE s.parent_location_id = p_parent_location_id
    ORDER BY s.name;
END;
$$;

-- Function to update a sublocation
CREATE OR REPLACE FUNCTION update_sublocation(
    p_id TEXT,
    p_name TEXT,
    p_info TEXT,
    p_svgpin TEXT,
    p_zoom TEXT DEFAULT NULL,
    p_type TEXT DEFAULT 'island'
)
RETURNS TABLE(
    id TEXT,
    name TEXT,
    info TEXT,
    coordinates JSONB,
    svgpin TEXT,
    parent_location_id TEXT,
    zoom TEXT,
    type TEXT
)
LANGUAGE plpgsql
AS $$
BEGIN
    RETURN QUERY
    UPDATE sublocations
    SET name = p_name,
        info = p_info,
        svgpin = p_svgpin,
        zoom = p_zoom,
        type = p_type
    WHERE sublocations.id = p_id
    RETURNING sublocations.id, sublocations.name, sublocations.info, sublocations.coordinates,
              sublocations.svgpin, sublocations.parent_location_id, sublocations.zoom, sublocations.type;
END;
$$;

-- Function to delete a sublocation
CREATE OR REPLACE FUNCTION delete_sublocation(p_id TEXT)
RETURNS BOOLEAN
LANGUAGE plpgsql
AS $$
DECLARE
    deleted BOOLEAN;
BEGIN
    DELETE FROM sublocations
    WHERE id = p_id;
    
    deleted := FOUND;
    RETURN deleted;
END;
$$;

