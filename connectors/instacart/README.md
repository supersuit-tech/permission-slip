# Instacart Connector

Integrates with the [Instacart Developer Platform](https://docs.instacart.com/developer_platform_api/) using plain `net/http` and Bearer API key authentication.

The API creates **Instacart-hosted landing pages**: users open the returned URL, choose a local store, and complete checkout on Instacart. There is no direct inventory or pricing API — matching is driven by product names (and optional UPCs) you send.

## Connector ID

`instacart`

## Credentials

| Key | Required | Description |
|-----|----------|-------------|
| `api_key` | Yes | API key from [Instacart Developer](https://www.instacart.com/developer) (used as `Authorization: Bearer` token). |
| `base_url` | No | `https://connect.instacart.com` (production, default) or `https://connect.dev.instacart.tools` (sandbox). Other hosts are rejected to avoid SSRF. |

Partnership approval is typically required before production keys work.

## Actions

| Action Type | Name | Risk | Description |
|-------------|------|------|-------------|
| `instacart.create_products_link` | Create Products Link | low | `POST /idp/v1/products/products_link` — returns `products_link_url`. |

### Parameter conveniences

- Use **`items`** as an alias for **`line_items`** (normalized at approval time).
- **`line_items`** may be an array of **strings**; each string becomes `{"name": "<string>"}` before calling Instacart.

Full LineItem fields (`line_item_measurements`, `upcs`, `filters`, etc.) follow [Instacart’s request schema](https://docs.instacart.com/developer_platform_api/api/products/create_shopping_list_page).

## Error handling

| HTTP status | Connector error |
|-------------|-----------------|
| 401 | `AuthError` |
| 403 | `AuthError` |
| 429 | `RateLimitError` (honors `Retry-After` when present) |
| Other 4xx/5xx | `ExternalError` |

Response bodies are read with a size cap (2 MiB).
