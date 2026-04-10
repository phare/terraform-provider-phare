# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.0.8] - 2026-04-10

### Added

- `phare_uptime_monitor_tcp` and `phare_uptime_monitor_tcp` now support the `region_threshold` parameter

## [0.0.7] - 2026-03-27

### Added

- `phare_uptime_status_page` resource now supports logo and favicon file uploads via new attributes:
  - `logo_light` - Path to light theme logo image file (jpeg/png/svg)
  - `logo_dark` - Path to dark theme logo image file (jpeg/png/svg)
  - `favicon_light` - Path to light theme favicon file (png/svg)
  - `favicon_dark` - Path to dark theme favicon file (png/svg)

### Changed

- `components` attribute is now required for the `phare_uptime_status_page` resource

## [0.0.6] - 2026-03-14

### Changed

- Fix resource validation when using project-scoped API keys
- Add acceptance tests for project-scoped API keys

## [0.0.5] - 2026-03-06

### Changed

- Documentation updates

## [0.0.4] - 2026-03-06

### Added

- `uptime_status_page_resource` now support the `color_scheme` and `theme` properties

### Changed

- Added string trimming to avoid state mismatch
- Update `uptime_monitor_tcp_resource` schema to nested block
- Update `uptime_monitor_http_resource` schema to nested block
- Update `project_resource` schema to nested block
- Update `uptime_status_page_resource` schema to nested block
- Improve and update documentation

## [0.0.3] - 2026-02-22

### Changed

- Updated documentation
- Updated development tooling

## [0.0.2] - 2026-02-22

### Changed

- Code linting

## [0.0.1] - 2026-02-22

### Added

- Initial release
