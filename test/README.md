# Frontend Smoke Environment

This folder is a minimal SPA fixture used to verify:

- static asset serving
- SPA fallback (`/docs`, `/about`, etc. -> `index.html`)
- runtime compression generation (`br` / `zstd` / `gzip`)

## Build Fixture

```powershell
pnpm -C test build
```

Output directory:

`test/build/dist`

## Run Spack With Fixture

```powershell
$env:SPACK_ASSETS_ROOT = (Resolve-Path .\test\build\dist).Path
$env:SPACK_ASSETS_PATH = "/"
go run .
```

Then open:

- `http://127.0.0.1/`
- `http://127.0.0.1/docs`
- `http://127.0.0.1/about`
- `http://127.0.0.1/catalog`

## Compression Check

First request may return identity while async generation is running.
Repeat the request once or twice:

```powershell
curl.exe -I -H "Accept-Encoding: br,zstd,gzip" http://127.0.0.1/assets/payload.json
```

Expected on warm hit:

- `Content-Encoding: br`, `zstd`, or `gzip`
- `Vary: Accept-Encoding`
