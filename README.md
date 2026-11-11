# Morgonfikan

A small, read-only service for displaying the morning fika schedule. The Swedish webpage and JSON API use the same configured schedule.

## API

All dates use the `Europe/Stockholm` time zone.

- `GET /api/v1/today` returns today's day object.
- `GET /api/v1/week` returns `{ "days": [...] }` for the next five weekdays, excluding today.
- `GET /healthz` returns `204 No Content` when the process is running.

A day object has an ISO `date`, a `served` boolean, an `items` array, and an optional closure `note`. A closed or weekend day returns `served: false` and an empty `items` array. API dates without a configured schedule return `503 Service Unavailable`.

The read-only API allows cross-origin `GET`, `HEAD`, and preflight `OPTIONS` requests without credentials.

## Schedule configuration

Edit [`config/schedule.yaml`](config/schedule.yaml) to add a schedule version. Each version has:

- `valid_from`: inclusive date when the version becomes effective.
- `anchor_date`: a Monday that defines the first week of that version's rotation.
- `weekdays`: an ordered, non-empty list of complete menu labels for each weekday from Monday through Friday.

The version with the latest `valid_from` not after the requested date is used. The anchor week uses the first entry for each weekday. Each subsequent week advances each weekday's list independently and wraps at its length. A new version should have its own Monday anchor date; closures do not pause or shift rotations.

Edit [`config/closures.yaml`](config/closures.yaml) to add explicit no-serving dates. Each closure has a unique ISO `date` and may have a short `note`. Closures affect only the listed date and override the regular schedule.

The current schedule starts on 2026-11-09. Menus are stored as complete Swedish labels and returned without translation.

## Validation and image

Run the test suite with `go test ./...` and validate both YAML files with `go run ./cmd/validate-config`. The GitHub Actions workflow runs tests, static checks, configuration validation, and dependency/container vulnerability scans. On successful pushes to `main`, it publishes a multi-platform image (`amd64` and `arm64`) to GHCR using an immutable `<commit-count>-<8-character-sha>` tag.

After the first publish, set the GHCR package visibility to public to allow anonymous image pulls; GitHub does not inherit package visibility from the public source repository.
