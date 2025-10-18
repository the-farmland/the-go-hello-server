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



-- Updated get_location_by_id function with results column
CREATE OR REPLACE FUNCTION public.get_location_by_id(location_id text)
RETURNS TABLE(
    id text,
    name text,
    country text,
    state text,
    description text,
    svg_link text,
    rating numeric,
    map_main_image text,
    map_cover_image text,
    main_background_image text,
    map_full_address text,
    map_png_link text,
    boards jsonb,
    coordinates jsonb,
    landmarks jsonb,
    parent_location_id text,
    business jsonb,
    hospitality jsonb,
    events jsonb,
    psa jsonb,
    sublocations jsonb,
    geojson text,
    hotzones jsonb,
    zoom text,
    results jsonb
)
LANGUAGE plpgsql
AS $$
DECLARE
    is_sublocation boolean := false;
    parent_id text;
    child_id text;
BEGIN
    -- Check if the location_id contains underscore (sublocation format)
    IF position('_' in location_id) > 0 THEN
        is_sublocation := true;
        parent_id := split_part(location_id, '_', 1);
        child_id := split_part(location_id, '_', 2);
    END IF;

    IF is_sublocation THEN
        -- Return parent location data with sublocation info
        RETURN QUERY
        SELECT 
            l.id,
            COALESCE(s.name, l.name) as name,
            l.country,
            l.state,
            COALESCE(s.info, l.description) as description,
            l.svg_link,
            l.rating,
            l.map_main_image,
            l.map_cover_image,
            l.main_background_image,
            l.map_full_address,
            l.map_png_link,
            l.boards,
            COALESCE(s.coordinates, l.coordinates) as coordinates,
            l.landmarks,
            l.parent_location_id,
            l.business,
            l.hospitality,
            l.events,
            l.psa,
            jsonb_build_object(
                'current_sublocation', jsonb_build_object(
                    'id', s.id,
                    'name', s.name,
                    'info', s.info,
                    'coordinates', s.coordinates,
                    'svgpin', s.svgpin,
                    'zoom', s.zoom
                ),
                'all_sublocations', (
                    SELECT jsonb_agg(
                        jsonb_build_object(
                            'id', sub.id,
                            'name', sub.name,
                            'info', sub.info,
                            'coordinates', sub.coordinates,
                            'svgpin', sub.svgpin,
                            'zoom', sub.zoom
                        )
                    )
                    FROM sublocations sub 
                    WHERE sub.parent_location_id = l.id
                )
            ) as sublocations,
            l.geojson,
            l.hotzones,
            COALESCE(s.zoom, l.zoom) as zoom,
            l.results
        FROM locations l
        LEFT JOIN sublocations s ON s.id = child_id AND s.parent_location_id = parent_id
        WHERE l.id = parent_id;
    ELSE
        -- Return regular location with all sublocations
        RETURN QUERY
        SELECT 
            l.id,
            l.name,
            l.country,
            l.state,
            l.description,
            l.svg_link,
            l.rating,
            l.map_main_image,
            l.map_cover_image,
            l.main_background_image,
            l.map_full_address,
            l.map_png_link,
            l.boards,
            l.coordinates,
            l.landmarks,
            l.parent_location_id,
            l.business,
            l.hospitality,
            l.events,
            l.psa,
            CASE 
                WHEN EXISTS(SELECT 1 FROM sublocations WHERE sublocations.parent_location_id = l.id) THEN
                    jsonb_build_object(
                        'all_sublocations', (
                            SELECT jsonb_agg(
                                jsonb_build_object(
                                    'id', sub.id,
                                    'name', sub.name,
                                    'info', sub.info,
                                    'coordinates', sub.coordinates,
                                    'svgpin', sub.svgpin,
                                    'zoom', sub.zoom
                                )
                            )
                            FROM sublocations sub 
                            WHERE sub.parent_location_id = l.id
                        )
                    )
                ELSE NULL
            END as sublocations,
            l.geojson,
            l.hotzones,
            l.zoom,
            l.results
        FROM locations l
        WHERE l.id = location_id;
    END IF;
END;
$$;


-- Updated get_top_locations function with results column
CREATE OR REPLACE FUNCTION public.get_top_locations(limit_count integer)
RETURNS TABLE(
    id text,
    name text,
    country text,
    state text,
    description text,
    svg_link text,
    rating numeric,
    map_main_image text,
    map_cover_image text,
    main_background_image text,
    map_full_address text,
    map_png_link text,
    boards jsonb,
    coordinates jsonb,
    landmarks jsonb,
    parent_location_id text,
    business jsonb,
    hospitality jsonb,
    events jsonb,
    psa jsonb,
    sublocations jsonb,
    geojson text,
    hotzones jsonb,
    zoom text,
    results jsonb
)
LANGUAGE plpgsql
AS $$
BEGIN
    RETURN QUERY
    SELECT 
        l.id,
        l.name,
        l.country,
        l.state,
        l.description,
        l.svg_link,
        l.rating,
        l.map_main_image,
        l.map_cover_image,
        l.main_background_image,
        l.map_full_address,
        l.map_png_link,
        l.boards,
        l.coordinates,
        l.landmarks,
        l.parent_location_id,
        l.business,
        l.hospitality,
        l.events,
        l.psa,
        CASE 
            WHEN EXISTS(SELECT 1 FROM sublocations WHERE sublocations.parent_location_id = l.id) THEN
                jsonb_build_object(
                    'all_sublocations', (
                        SELECT jsonb_agg(
                            jsonb_build_object(
                                'id', sub.id,
                                'name', sub.name,
                                'info', sub.info,
                                'coordinates', sub.coordinates,
                                'svgpin', sub.svgpin,
                                'zoom', sub.zoom
                            )
                        )
                        FROM sublocations sub 
                        WHERE sub.parent_location_id = l.id
                    )
                )
            ELSE NULL
        END as sublocations,
        l.geojson,
        l.hotzones,
        l.zoom,
        l.results
    FROM locations l
    WHERE l.rating IS NOT NULL
    ORDER BY l.rating DESC NULLS LAST
    LIMIT limit_count;
END;
$$;


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

-- 6. Update search_locations to use id instead of pinLink
CREATE OR REPLACE FUNCTION public.search_locations(search_query text)
RETURNS TABLE(
    id text,
    name text,
    country text,
    state text,
    description text,
    svg_link text,
    rating numeric,
    map_main_image text,
    map_cover_image text,
    main_background_image text,
    map_full_address text,
    map_png_link text,
    boards jsonb,
    coordinates jsonb,
    landmarks jsonb,
    parent_location_id text,
    business jsonb,
    hospitality jsonb,
    events jsonb,
    psa jsonb,
    sublocations jsonb,
    geojson text,
    hotzones jsonb,
    zoom text,
    results jsonb
)
LANGUAGE plpgsql
AS $$
BEGIN
  RETURN QUERY
  WITH pin_match AS (
    SELECT
      l.id AS location_id,
      'landmarks'::text AS col,
      p.idx,
      p.pin ->> 'name' AS pin_name,
      p.pin ->> 'id' AS pin_id
    FROM locations l
    CROSS JOIN LATERAL jsonb_array_elements(COALESCE(l.landmarks, '[]'::jsonb)) WITH ORDINALITY AS p(pin, idx)
    WHERE (p.pin ->> 'name') ILIKE '%' || search_query || '%'
       OR (p.pin ->> 'info') ILIKE '%' || search_query || '%'
  UNION ALL
    SELECT l.id, 'business', p.idx, p.pin ->> 'name', p.pin ->> 'id'
    FROM locations l
    CROSS JOIN LATERAL jsonb_array_elements(COALESCE(l.business, '[]'::jsonb)) WITH ORDINALITY AS p(pin, idx)
    WHERE (p.pin ->> 'name') ILIKE '%' || search_query || '%'
       OR (p.pin ->> 'info') ILIKE '%' || search_query || '%'
  UNION ALL
    SELECT l.id, 'hospitality', p.idx, p.pin ->> 'name', p.pin ->> 'id'
    FROM locations l
    CROSS JOIN LATERAL jsonb_array_elements(COALESCE(l.hospitality, '[]'::jsonb)) WITH ORDINALITY AS p(pin, idx)
    WHERE (p.pin ->> 'name') ILIKE '%' || search_query || '%'
       OR (p.pin ->> 'info') ILIKE '%' || search_query || '%'
  UNION ALL
    SELECT l.id, 'events', p.idx, p.pin ->> 'name', p.pin ->> 'id'
    FROM locations l
    CROSS JOIN LATERAL jsonb_array_elements(COALESCE(l.events, '[]'::jsonb)) WITH ORDINALITY AS p(pin, idx)
    WHERE (p.pin ->> 'name') ILIKE '%' || search_query || '%'
       OR (p.pin ->> 'info') ILIKE '%' || search_query || '%'
  UNION ALL
    SELECT l.id, 'psa', p.idx, p.pin ->> 'name', p.pin ->> 'id'
    FROM locations l
    CROSS JOIN LATERAL jsonb_array_elements(COALESCE(l.psa, '[]'::jsonb)) WITH ORDINALITY AS p(pin, idx)
    WHERE (p.pin ->> 'name') ILIKE '%' || search_query || '%'
       OR (p.pin ->> 'info') ILIKE '%' || search_query || '%'
  UNION ALL
    SELECT l.id, 'hotzones', p.idx, p.pin ->> 'name', p.pin ->> 'id'
    FROM locations l
    CROSS JOIN LATERAL jsonb_array_elements(COALESCE(l.hotzones, '[]'::jsonb)) WITH ORDINALITY AS p(pin, idx)
    WHERE (p.pin ->> 'name') ILIKE '%' || search_query || '%'
       OR (p.pin ->> 'info') ILIKE '%' || search_query || '%'
  UNION ALL
    SELECT l.id, 'results', p.idx, p.pin ->> 'name', p.pin ->> 'id'
    FROM locations l
    CROSS JOIN LATERAL jsonb_array_elements(COALESCE(l.results, '[]'::jsonb)) WITH ORDINALITY AS p(pin, idx)
    WHERE (p.pin ->> 'name') ILIKE '%' || search_query || '%'
       OR (p.pin ->> 'info') ILIKE '%' || search_query || '%'
  ),
  sub_match AS (
    SELECT s.parent_location_id AS location_id, s.id AS sub_id
    FROM sublocations s
    WHERE s.name ILIKE '%' || search_query || '%'
       OR s.info ILIKE '%' || search_query || '%'
  )
  SELECT 
      l.id,
      l.name,
      l.country,
      l.state,
      l.description,
      l.svg_link,
      l.rating,
      l.map_main_image,
      l.map_cover_image,
      l.main_background_image,
      l.map_full_address,
      l.map_png_link,
      l.boards,
      l.coordinates,
      l.landmarks,
      l.parent_location_id,
      l.business,
      l.hospitality,
      l.events,
      l.psa,
      CASE 
          WHEN EXISTS(SELECT 1 FROM sublocations WHERE sublocations.parent_location_id = l.id) THEN
              jsonb_build_object(
                  'all_sublocations', (
                      SELECT jsonb_agg(
                          jsonb_build_object(
                              'id', sub.id,
                              'name', sub.name,
                              'info', sub.info,
                              'coordinates', sub.coordinates,
                              'svgpin', sub.svgpin,
                              'zoom', sub.zoom
                          )
                      )
                      FROM sublocations sub 
                      WHERE sub.parent_location_id = l.id
                  )
              )
          ELSE NULL
      END as sublocations,
      l.geojson,
      l.hotzones,
      l.zoom,
      jsonb_build_array() ||
      COALESCE(
        (
          SELECT jsonb_agg(
            jsonb_build_object(
              'type', 'pin_match',
              'column', pm.col,
              'pinId', pm.pin_id,
              'name', pm.pin_name
            )
          )
          FROM pin_match pm
          WHERE pm.location_id = l.id
        ), '[]'::jsonb
      ) ||
      COALESCE(
        (
          SELECT jsonb_agg(
            jsonb_build_object(
              'type', 'sublocation_match',
              'sublocation_id', sm.sub_id
            )
          )
          FROM sub_match sm
          WHERE sm.location_id = l.id
        ), '[]'::jsonb
      ) as results
  FROM locations l
  WHERE 
       l.name ILIKE '%' || search_query || '%'
    OR l.country ILIKE '%' || search_query || '%'
    OR COALESCE(l.state, '') ILIKE '%' || search_query || '%'
    OR COALESCE(l.map_full_address, '') ILIKE '%' || search_query || '%'
    OR EXISTS (SELECT 1 FROM pin_match pm WHERE pm.location_id = l.id)
    OR EXISTS (SELECT 1 FROM sub_match sm WHERE sm.location_id = l.id)
  ORDER BY l.rating DESC NULLS LAST, l.name ASC;
END;
$$;



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

