# Meilisearch: Local vs Cloud

Geomilli is designed to run seamlessly against a **self-hosted local Meilisearch container** or a fully managed **Meilisearch Cloud** cluster with zero application code changes.

---

## Architecture Comparison

| Feature | Local Docker Meilisearch | Meilisearch Cloud |
| :--- | :--- | :--- |
| **Hosting Model** | Container on host machine (`getmeili/meilisearch:v1.13`) | Managed multi-tenant or dedicated cluster |
| **Default Host** | `http://127.0.0.1:7700` or `http://meilisearch:7700` | `https://ms-xxxx.region.meilisearch.io` |
| **API Key** | Configurable (`masterKey123` in `.env.example`) | Cloud Dashboard API Key |
| **Storage** | Local Docker volume (`meili_data`) | Cloud SSD / High-IOPS managed storage |
| **High Availability** | Single node | Auto-failover & regional replicas |
| **SSL / TLS** | Self-managed reverse proxy (e.g. Caddy/Nginx) | Automatic HTTPS & TLS 1.3 |

---

## 1. Local Meilisearch (Default)

In standard local mode, Docker Compose boots Meilisearch alongside `geomilli-api`, `geomilli-harvester`, and `geomilli-studio`:

```bash
docker compose up -d
```

### Environment Settings (`.env`):
```dotenv
MEILI_HOST=http://meilisearch:7700
MEILI_API_KEY=masterKey123
```

Within the Docker network, services communicate over `http://meilisearch:7700`. From your host workstation, Meilisearch is reachable at `http://localhost:7700`.

---

## 2. Meilisearch Cloud Deployment

When deploying to production with Meilisearch Cloud:

1. Create a project at [cloud.meilisearch.com](https://cloud.meilisearch.com).
2. Note your **URL** (e.g., `https://ms-981273abc.sfo.meilisearch.io`) and **Default Admin API Key**.
3. Update your `.env` file:
   ```dotenv
   MEILI_HOST=https://ms-981273abc.sfo.meilisearch.io
   MEILI_API_KEY=meili_cloud_admin_key_xxxxxxxxxxxxx
   ```
4. Start the stack using the cloud compose override:
   ```bash
   docker compose -f docker-compose.yml -f docker-compose.cloud.yml up -d
   ```

### What `docker-compose.cloud.yml` Does:
- **Disables the local `meilisearch` container**: Frees up local CPU, RAM, and port `7700`.
- **Reconfigures `geomilli-api` and `geomilli-harvester`**: Connects directly to the external Meilisearch Cloud endpoint over secure HTTPS.
- **Adjusts health dependencies**: Allows the API and harvester to boot immediately without waiting for a local container health check.

---

## 3. Quota & Indexing Considerations for Cloud

When indexing large regions into Meilisearch Cloud:
- Ensure your cloud plan has sufficient document quota (Belgium produces ~180k places; Europe produces ~30M places; Planet produces ~120M places).
- The shadow index atomic swap (`swapIndexes`) operates identically on Cloud and Local, ensuring zero-downtime upgrades.
