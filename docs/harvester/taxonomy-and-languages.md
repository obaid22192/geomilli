# Taxonomy & Languages

Geomilli classifies raw OpenStreetMap (OSM) data into **19 normalized taxonomy classes** and indexes multilingual name translations defined by strict, config-driven schemas.

---

## 1. Config-Driven Architectural Principles

1. **Strictly Config-Driven**: All class matching rules, priority order, attribute extraction rules, base popularity scores, and layer mappings are declared in `config/taxonomy.yaml`. There are **no hardcoded `switch` statements or conditional classification logic** in the Go code.
2. **Order Determines Priority**: In `config/taxonomy.yaml`, rules are evaluated from **top to bottom**. The **first match wins**. POI layer classes come before administrative boundary classes so that features with both POI tags and administrative boundaries (e.g., Vatican City) are classified as POIs.
3. **Class vs. Subclass Hierarchy**:
   - **`class`**: Broad top-level category (e.g. `TouristAttraction`, `HistoricSite`, `Restaurant`, `NaturalFeature`). Used for primary filtering, facets, and indexing.
   - **`subclass`**: The specific OSM tag value that triggered the match (e.g. `museum`, `castle`, `monument`, `bar`, `peak`, `viewpoint`). Stored on the document and searchable.
4. **Language Whitelisting**: Supported languages are whitelisted in `config/languages.yaml`. Any language not on the whitelist is stripped during name normalization to avoid index payload bloat.
5. **Runtime Exposure via API**: `geomilli-api` exposes the current configuration dynamically at `GET /v1/taxonomy` and `GET /v1/languages`. `geomilli-studio` reads this endpoint so new taxonomy additions appear automatically in the UI.

---

## 2. The 19 Canonical Classes Reference

The 19 canonical classes defined in `config/taxonomy.yaml`, their matching tags, extracted attributes, and live document counts:

| Class | Layer | OSM Match Criteria | Example Subclasses | Extracted Attributes | Base Score |
| :--- | :---: | :--- | :--- | :--- | :---: |
| **Airport** | `poi` | `aeroway=aerodrome, airport` | `aerodrome` | `iata_code` (IATA), `icao_code` (ICAO) | `0.80` |
| **TrainStation** | `poi` | `railway=station, halt` | `station`, `halt` | — | `0.60` |
| **TouristAttraction**| `poi` | `tourism=attraction, museum, viewpoint, zoo, theme_park, gallery, aquarium`<br/>`amenity=place_of_worship` (requires wikidata/wikipedia) | `museum`, `viewpoint`, `attraction`, `zoo`, `gallery`, `place_of_worship` | — | `0.58` |
| **Stadium** | `poi` | `leisure=stadium` | `stadium` | — | `0.56` |
| **Hotel** | `poi` | `tourism=hotel, hostel, motel, guest_house, apartment, camp_site, chalet` | `hotel`, `hostel`, `apartment`, `guest_house`, `camp_site`, `chalet` | `star_rating` (`stars=*`), `accommodation_type` (`tourism=*`) | `0.50` |
| **HistoricSite** | `poi` | `historic=castle, monument, memorial, archaeological_site, ruins, fort, battlefield` | `castle`, `monument`, `memorial`, `ruins`, `archaeological_site`, `fort` | — | `0.50` |
| **Park** | `poi` | `leisure=park, nature_reserve, marina`<br/>`boundary=national_park` | `park`, `nature_reserve`, `national_park`, `marina` | — | `0.48` |
| **BusStation** | `poi` | `amenity=bus_station` | `bus_station` | — | `0.48` |
| **FerryTerminal** | `poi` | `amenity=ferry_terminal` | `ferry_terminal` | — | `0.48` |
| **Beach** | `poi` | `natural=beach`<br/>`leisure=beach_resort` | `beach`, `beach_resort` | — | `0.45` |
| **Restaurant** | `poi` | `amenity=restaurant, cafe, bar, pub` | `restaurant`, `cafe`, `bar`, `pub` | `cuisine` (`cuisine=*`) | `0.40` |
| **Entertainment** | `poi` | `amenity=theatre, cinema, nightclub`<br/>`leisure=water_park` | `theatre`, `cinema`, `nightclub`, `water_park` | — | `0.40` |
| **NaturalFeature** | `poi` | `natural=peak, volcano, cave_entrance, glacier` *(node geometry)* | `peak`, `cave_entrance`, `volcano`, `glacier` | — | `0.38` |
| **Shopping** | `poi` | `shop=mall, department_store` | `mall`, `department_store` | — | `0.38` |
| **Country** | `admin` | `place=country` | `country` | — | `0.75` |
| **City** | `admin` | `admin_level=6, 7, 8`<br/>`place=city, town, village` | `city`, `town`, `village` | `population`, `admin_level` | `0.65` |
| **Region** | `admin` | `admin_level=4` | `state`, `province`, `region` | `admin_level` | `0.55` |
| **Island** | `admin` | `place=island, islet` | `island`, `islet` | — | `0.50` |
| **Neighborhood** | `admin` | `admin_level=9, 10`<br/>`place=suburb, neighbourhood, neighborhood, quarter` | `suburb`, `neighbourhood`, `quarter` | `admin_level` | `0.45` |

---

## 3. Important: Class vs. Subclass Disambiguation

A common point of confusion is querying for concepts that are **subclasses** as if they were top-level **classes**.

When querying the API or filtering in the Studio, **`class` filters the broad category**, while **`subclass` filters the specific type**:

| What you want to find | Correct `class` Parameter | Correct `subclass` Parameter | Example Query |
| :--- | :--- | :--- | :--- |
| **Museums** | `class=TouristAttraction` | `subclass=museum` | `GET /v1/autocomplete?q=Louvre&class=TouristAttraction&subclass=museum` |
| **Castles** | `class=HistoricSite` | `subclass=castle` | `GET /v1/search?class=HistoricSite&subclass=castle` |
| **Monuments** | `class=HistoricSite` | `subclass=monument` | `GET /v1/nearby?lat=50.85&lng=4.35&class=HistoricSite&subclass=monument` |
| **Bars & Pubs** | `class=Restaurant` | `subclass=bar` (or `pub`) | `GET /v1/nearby?lat=50.85&lng=4.35&class=Restaurant&subclass=bar` |
| **Mountain Peaks** | `class=NaturalFeature` | `subclass=peak` | `GET /v1/search?q=Blanc&class=NaturalFeature&subclass=peak` |
| **Malls & Department Stores** | `class=Shopping` | `subclass=mall` | `GET /v1/autocomplete?q=Anspach&class=Shopping` |
| **Hostels & Apartments** | `class=Hotel` | `subclass=hostel` (or `apartment`) | `GET /v1/search?class=Hotel&subclass=hostel` |

> [!WARNING]
> Sending `class=Museum` or `class=Castle` or `class=Bar` will return **0 results** because there is no top-level class with those names. Use `class=TouristAttraction&subclass=museum` or `class=HistoricSite&subclass=castle`.

---

## 4. Full-Text Search vs. Filter Attributes

In Meilisearch, fields are classified into **Searchable** (matched during text queries `q=...`) and **Filterable** (exact matching via `class=...` or filter expressions):

* **`SearchableAttributes`**:
  `["name_display", "name_alt_active", "iata_code", "icao_code", "class", "subclass", "admin_hierarchy", "country_name"]`
  * When a user searches for `"Stadium"` or `"Beach"` in the search bar, the text query matches `class` and `subclass` directly.
  * When a user searches for `"BRU"` or `"CRL"`, the text query matches `iata_code`.
* **`FilterableAttributes`**:
  `["class", "subclass", "country_code", "admin_level", "_geo", "star_rating", "cuisine", "iata_code", "icao_code", "accommodation_type"]`
  * Dedicated parameter filters (`&class=TrainStation`, `&country=BE`, `&cuisine=italian`) use exact inverted index filters with zero query overhead.

---

## 4. How to Add or Modify Taxonomy Classes

Because Geomilli uses a single source of truth for taxonomy, adding new categories or tags is automated through configuration.

### Scenario A: Adding a Subclass to an Existing Class
*Example: You want to add `winery` and `brewery` to the `Restaurant` class.*

1. Open `config/taxonomy.yaml` (or `geomilli-harvester/config/taxonomy.yaml`).
2. Locate the `Restaurant` class block:
   ```yaml
   - name: Restaurant
     base_score: 0.40
     layer: poi
     match:
       - key: amenity
         values: [restaurant, cafe, bar, pub, winery, brewery] # <-- Add winery, brewery
   ```
3. Re-run harvester pipeline for your target extract:
   ```bash
   make refresh
   # or: go run ./geomilli-harvester/cmd/pipeline --stage filter,transform,index
   ```
   *Note: `geomilli-harvester`'s `gen-filters` utility automatically generates the updated Osmium tag filter expressions directly from the YAML config. No Go code modification is needed!*

---

### Scenario B: Adding a Completely New Top-Level Class
*Example: You want to add a new `Hospital` class to index medical facilities.*

#### Step 1: Declare the Class in `config/taxonomy.yaml`
Add your new class definition in the desired priority position (POI classes before admin classes):

```yaml
  - name: Hospital
    base_score: 0.55
    layer: poi
    match:
      - key: amenity
        values: [hospital, clinic]
    attributes:
      - tag: emergency
        field: emergency_type
        transform: lower
```

#### Step 2: Add Any Custom Schema Fields (If Applicable)
If your new class extracts custom attributes (like `emergency_type` above) that need to be searchable or filterable in Meilisearch:
1. Add the field to `schema.Place` in `geomilli-harvester/internal/schema/place.go` and `geomilli-api/internal/schema/place.go`:
   ```go
   EmergencyType string `json:"emergency_type,omitempty"`
   ```
2. If filterable, register it in `FilterableAttributes` in `geomilli-harvester/internal/meili/indexer.go`:
   ```go
   "class", "subclass", "country_code", "emergency_type", ...
   ```

*(If you are not adding new custom attributes and only matching OSM tags into `class` and `subclass`, skip Step 2!)*

#### Step 3: Run the Ingestion Pipeline
Execute the pipeline stage:
```bash
make refresh
# or: go run ./geomilli-harvester/cmd/pipeline --countries BE --stage all
```

#### Step 4: Verification
1. **Verify Taxonomy API**:
   ```bash
   curl -s http://localhost:8080/v1/taxonomy | jq '.classes[] | select(.name=="Hospital")'
   ```
2. **Verify Search API**:
   ```bash
   curl -s "http://localhost:8080/v1/search?class=Hospital" | jq '{total_hits: .total_hits, sample: .results[:2]}'
   ```
3. **Verify Studio UI**:
   Open Geomilli Studio (`http://localhost:3000` or `http://localhost:3333`). The new `Hospital` pill will appear dynamically in the Taxonomy Class Filter bar.

---

## 5. Attribute Extraction Engine

The `attributes` block in `taxonomy.yaml` tells the harvester to pull specific OSM tags into structured schema fields on the indexed document:

```yaml
attributes:
  - tag: stars            # Source OSM tag (e.g. stars=4)
    field: star_rating    # Destination field in Place document
    transform: int        # Transformations: int | float | upper | lower
  - tag: cuisine
    field: cuisine
```

Supported transformations:
- `int`: Converts string to integer (e.g. `"4"` -> `4`).
- `float`: Converts string to floating-point number.
- `upper`: Uppercases string (e.g. `"bru"` -> `"BRU"` for airport IATA codes).
- `lower`: Lowercases string.
- *Omitted*: Retains string value as-is.

---

## 6. Multilingual Name Normalization (`config/languages.yaml`)

OpenStreetMap contains multilingual names via tags such as `name:en`, `name:fr`, `name:nl`, `name:de`, `int_name`, and `alt_name`.

To prevent unbounded index bloat and search degradation, only whitelisted languages configured in `config/languages.yaml` are stored:

```yaml
version: 1
languages:
  - code: en
    name: English
    enabled: true
  - code: nl
    name: Dutch
    enabled: true
  - code: fr
    name: French
    enabled: true
  - code: de
    name: German
    enabled: true
  - code: es
    name: Spanish
    enabled: true
```

### Ingested Document Output
When a place is indexed, its language translations are organized into `names_by_language`:
```json
{
  "id": "osm-feature-fallback-d39329480840060c",
  "name_display": "Bruxelles - Brussel",
  "names_by_language": {
    "en": "Brussels",
    "fr": "Bruxelles",
    "nl": "Brussel",
    "de": "Brüssel",
    "es": "Bruselas"
  },
  "class": "City",
  "subclass": "city",
  "country_code": "BE",
  "_geo": {
    "lat": 50.8467,
    "lng": 4.3525
  },
  "popularity_score": 0.86
}
```

When a user provides the `lang` parameter (e.g., `GET /v1/autocomplete?q=Brux&lang=en`), the API dynamically localized `name_display` to `"Brussels"`.
