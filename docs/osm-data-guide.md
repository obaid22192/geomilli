# OpenStreetMap Data Guide: Autocomplete Fields and Classification

This document maps the OpenStreetMap (OSM) data structure, tags, and layers to the `auto-complete` service's internal schemas and classification rules.

---

## 1. Layers & Tag Ingestion

The pipeline downloads raw country-level extracts in `.osm.pbf` format from Geofabrik. It runs `osmium tags-filter` to generate streamable `.geojsonseq` layers containing only travel-relevant features.

We ingest two primary layers:
1. **Administrative Boundaries**: Scoped to `boundary=administrative` and selected settlement places (`place=city`, `place=town`, etc.).
2. **Points of Interest (POIs) & Transport Hubs**: Scoped to tourism, leisure, transport, historic, and dining categories.

---

## 2. Naming & Multilingual Tag Parsing

OSM maps features with rich name variations. To enable robust multi-language autocomplete, we extract all naming fields and index them as alternative search terms.

### Tag Mapping

| OSM Tag | Field in Schema | Description |
|---------|-----------------|-------------|
| `name` | `NameCanonical`, `NameDisplay` | Primary local/default name of the feature |
| `name:<lang>` | `NamesByLanguage` | Language-specific translations (e.g. `name:en`, `name:nl`, `name:fr`) |
| `alt_name` | `NameAltActive` | Alternative names used locally |
| `official_name` | `NameAltActive` | Official or legal name |
| `old_name` | `NameAltActive` | Historic or former name of the place |
| `loc_name` | `NameAltActive` | Colloquial or colloquial local name |
| `short_name` | `NameAltActive` | Shortened name or code (e.g. *AMS* for Schiphol) |
| `int_name` | `NameAltActive` | International name (transliterations) |

*Note: All values in `NamesByLanguage` and alternative name tags are merged, case-insensitively deduplicated, and fed into Meilisearch's `search_terms_flat` and `name_alt_active` fields.*

---

## 3. Entity Classification (19-Class Taxonomy)

The pipeline maps the raw tags of features to one of the following 19 classes. Each class is assigned a base popularity score, which is later enriched with population and pageview popularity metrics.

| Class | Tag Filter criteria | Base Score | Description |
|-------|---------------------|------------|-------------|
| **Country** | `place=country` | `0.75` | Sovereign nations and countries |
| **Airport** | `aeroway=aerodrome` or `aeroway=airport` | `0.80` | Airports, airfields, heliports |
| **City** | `admin_level=6-8`, or `place=city`/`town`/`village` | `0.65` | Cities, towns, municipalities, villages |
| **TrainStation** | `railway=station` or `railway=halt` | `0.60` | Train stations and railway halts |
| **TouristAttraction** | `tourism=attraction`/`museum`/`viewpoint`/`zoo`/`theme_park`/`gallery`/`aquarium` | `0.58` | Major tourist destinations, museums, galleries |
| **Stadium** | `leisure=stadium` | `0.56` | Sports stadiums and arenas |
| **Region** | `admin_level=4` | `0.55` | States, provinces, administrative regions |
| **Hotel** | `tourism=hotel`/`hostel`/`motel`/`guest_house`/`apartment`/`camp_site`/`chalet` | `0.50` | Lodgings and tourist accommodations |
| **HistoricSite** | `historic=castle`/`monument`/`memorial`/`archaeological_site`/`ruins` | `0.50` | Historic monuments, castles, ruins |
| **Island** | `place=island` or `place=islet` | `0.50` | Named islands and islets |
| **Park** | `leisure=park`/`nature_reserve`/`marina` or `boundary=national_park` | `0.48` | Parks, reserves, national parks |
| **BusStation** | `amenity=bus_station` | `0.48` | Bus terminals and stations |
| **FerryTerminal** | `amenity=ferry_terminal` | `0.48` | Passenger ferry terminals |
| **Beach** | `natural=beach` or `leisure=beach_resort` | `0.45` | Public beaches and beach resorts |
| **Neighborhood** | `admin_level=9-10`, or `place=suburb`/`neighbourhood`/`quarter` | `0.45` | City quarters, suburbs, neighborhoods |
| **Restaurant** | `amenity=restaurant`/`cafe`/`bar`/`pub` | `0.40` | Dining establishments, cafes, bars, pubs |
| **Entertainment** | `amenity=theatre`/`cinema`/`nightclub` or `leisure=water_park` | `0.40` | Theaters, cinemas, entertainment venues, nightclubs |
| **NaturalFeature** | `natural=peak`/`volcano`/`cave_entrance`/`glacier` | `0.38` | Mountain peaks, volcanoes, caves |
| **Shopping** | `shop=mall`/`department_store` | `0.38` | Shopping malls and department stores |

---

## 4. Custom Extracted Attributes

In addition to core coordinates and names, we extract entity-specific fields to enrich search matching:

### Airport Attributes
- `iata`: 3-letter IATA code (mapped to `iata_code`, e.g. *AMS*).
- `icao`: 4-letter ICAO code (mapped to `icao_code`, e.g. *EHAM*).

### Hotel Attributes
- `stars`: Star rating count (mapped to `star_rating`, e.g. *4*).
- `tourism`: Value of the tourism tag (mapped to `accommodation_type`, e.g. *hotel*, *hostel*).

### Restaurant Attributes
- `cuisine`: Dynamic cuisine categories (mapped to `cuisine`, e.g. *italian;pizza*). Split by semicolon and appended to search terms so users can search by food type.
