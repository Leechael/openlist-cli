# OpenList URL-Based Routes (Non-JSON API)

These routes return binary data (file downloads) or HTTP redirects, not JSON.
They are not included in `openapi.json` because they use path-based routing
with signatures rather than standard REST patterns.

## File Download

### Direct Download (302 Redirect)
```
GET /d/{path}?sign={sign}
HEAD /d/{path}?sign={sign}
```

Redirects (302) to the storage provider's direct URL.

| Parameter | Location | Required | Description |
|-----------|----------|----------|-------------|
| `path`    | URL path | Yes | File path, URL-encoded. e.g. `/d/阿里云盘/file.ts` |
| `sign`    | Query    | Conditional | Required when path is password-protected or `SignAll` setting is enabled |
| `type`    | Query    | No | Link type hint passed to storage driver |

**How to get `sign`**: The `sign` field is returned for each file in `/api/fs/list` and `/api/fs/get` responses.

**Example**:
```bash
# 1. List directory to get sign values
curl -s -X POST http://HOST:5244/api/fs/list \
  -H 'Content-Type: application/json' \
  -d '{"path":"/阿里云盘/movies","page":1,"per_page":0}' \
  | jq '.data.content[] | {name, sign}'

# 2. Download using sign
curl -L -o movie.ts \
  "http://HOST:5244/d/阿里云盘/movies/movie.ts?sign=BASE64SIGN:0"
```

### Proxy Download (Stream Through Server)
```
GET /p/{path}?sign={sign}
HEAD /p/{path}?sign={sign}
```

Streams file through the OpenList server instead of redirecting.
Useful when the storage provider URL is not directly accessible.

| Parameter | Location | Required | Description |
|-----------|----------|----------|-------------|
| `path`    | URL path | Yes | File path, URL-encoded |
| `sign`    | Query    | Conditional | Same rules as `/d/` |
| `type`    | Query    | No | Link type hint |
| `d`       | Query    | No | If present, skip proxy URL generation |
| `raw`     | Query    | No | For `.md` files: `raw=true` skips markdown-to-HTML conversion |

**Markdown handling**: `.md` files served via `/p/` are automatically converted
to sanitized HTML unless `?raw=true` is appended.

---

## Archive File Download

### Archive Driver Download (302 Redirect)
```
GET /ad/{archive_path}?sign={sign}&inner={inner_path}&pass={archive_password}
HEAD /ad/{archive_path}?sign={sign}&inner={inner_path}&pass={archive_password}
```

Extracts a file from an archive using the storage driver's native capability.

| Parameter | Location | Required | Description |
|-----------|----------|----------|-------------|
| `archive_path` | URL path | Yes | Path to the archive file |
| `sign`    | Query    | Conditional | **Archive signature** (different key from regular sign!) |
| `inner`   | Query    | Yes | Path within the archive |
| `pass`    | Query    | No | Archive file password (if encrypted) |
| `type`    | Query    | No | Link type hint |

### Archive Proxy Download (Stream Through Server)
```
GET /ap/{archive_path}?sign={sign}&inner={inner_path}&pass={archive_password}
HEAD /ap/{archive_path}?sign={sign}&inner={inner_path}&pass={archive_password}
```

Same parameters as `/ad/` but streams through server.

### Archive Internal Extract (Server-Side Extraction)
```
GET /ae/{archive_path}?sign={sign}&inner={inner_path}&pass={archive_password}
HEAD /ae/{archive_path}?sign={sign}&inner={inner_path}&pass={archive_password}
```

Server downloads the archive and extracts the file internally. Used when the
storage driver does not support native archive extraction.

**How to get archive sign and routes**: Call `POST /api/fs/archive/meta`.
The response includes `sign` (archive signature) and `raw_url` which indicates
whether to use `/ae` (internal extract) or `/ad` (driver download).

---

## Sharing Download

### Shared File Download
```
GET /sd/{sharing_id}
GET /sd/{sharing_id}/{path}
HEAD /sd/{sharing_id}
HEAD /sd/{sharing_id}/{path}
```

Download files from a sharing link. No signature required.

| Parameter | Location | Required | Description |
|-----------|----------|----------|-------------|
| `sharing_id` | URL path | Yes | 12-char sharing ID |
| `path`    | URL path | No | Subpath within the share (for directory shares) |

### Shared Archive Extract
```
GET /sad/{sharing_id}
GET /sad/{sharing_id}/{path}
HEAD /sad/{sharing_id}
HEAD /sad/{sharing_id}/{path}
```

Extract files from archives within a sharing link.

---

## Signature System

### Regular File Signatures
- Algorithm: HMAC-SHA256
- Key: System `Token` setting value
- Data: The file's full path
- Format: `{base64url_hmac}:{expiry_unix_timestamp}` (expiry `0` = never expires)
- Expiry controlled by `link_expiration` setting (in hours, 0 = no expiry)

### Archive File Signatures
- Algorithm: HMAC-SHA256
- Key: System `Token` + `"-archive"` suffix (different from regular signatures!)
- Data: The archive file's full path
- Same format as regular signatures

### When Signatures Are Required
A signature is required when ANY of these conditions are met:
1. `SignAll` system setting is enabled
2. The storage has `enable_sign` set to `true`
3. The path has a `Meta` entry with a `password` set (and the path matches)

---

## Download URL Construction Recipe

### For regular files:
```
{base_url}/d/{url_encode(path)}?sign={sign_from_api}
```

### For proxy download:
```
{base_url}/p/{url_encode(path)}?sign={sign_from_api}
```

### Alternative: Use raw_url from /api/fs/get
```bash
# Get the storage provider's direct URL
RAW_URL=$(curl -s -X POST http://HOST:5244/api/fs/get \
  -H "Authorization: $TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"path":"/阿里云盘/file.ts"}' | jq -r '.data.raw_url')

# Download directly from provider (faster, bypasses OpenList)
aria2c "$RAW_URL"
```

Note: `raw_url` has its own expiry from the storage provider (usually shorter).
The `/d/` route handles re-fetching expired URLs automatically.

---

## WebDAV Access

OpenList also exposes a WebDAV server:
```
/dav/*path
```
Supports standard WebDAV methods (PROPFIND, GET, PUT, DELETE, MKCOL, MOVE, COPY).
Authentication via HTTP Basic Auth with OpenList credentials.

## S3 Gateway

S3-compatible gateway available at:
```
/s3/*path
```
Supports basic S3 operations for compatible clients.
