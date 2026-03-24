# ScreenSpace Dev Environment
# Usage: tilt up

# Infrastructure: Postgres + MinIO via docker-compose
docker_compose('./docker-compose.dev.yml')

# Label the infra resources
dc_resource('postgres', labels=['infra'])
dc_resource('minio', labels=['infra'])
dc_resource('minio-init', labels=['infra'])

# Seed dev data: users, wallpapers, favorites
local_resource(
    'seed',
    cmd='cd server && go run ./cmd/seed',
    env={
        'DATABASE_URL': 'postgres://screenspace:devpassword@localhost:5432/screenspace?sslmode=disable',
        'S3_ENDPOINT': 'http://localhost:9000',
        'S3_BUCKET': 'screenspace',
        'S3_ACCESS_KEY': 'minioadmin',
        'S3_SECRET_KEY': 'minioadmin',
        'JWT_SECRET': 'dev-secret-do-not-use-in-production',
        'ADMIN_EMAIL': 'admin@screenspace.dev',
    },
    resource_deps=['postgres', 'minio-init'],
    auto_init=True,
    labels=['infra'],
)

# Go API server - built and run locally with live reload
local_resource(
    'server',
    serve_cmd='cd server && go run .',
    deps=['server'],
    ignore=['server/*_test.go', 'server/coverage.out'],
    serve_env={
        'DATABASE_URL': 'postgres://screenspace:devpassword@localhost:5432/screenspace?sslmode=disable',
        'S3_ENDPOINT': 'http://localhost:9000',
        'S3_BUCKET': 'screenspace',
        'S3_ACCESS_KEY': 'minioadmin',
        'S3_SECRET_KEY': 'minioadmin',
        'JWT_SECRET': 'dev-secret-do-not-use-in-production',
        'ADMIN_EMAIL': 'admin@screenspace.dev',
        'PORT': '8080',
    },
    resource_deps=['postgres', 'minio-init', 'seed'],
    labels=['backend'],
    readiness_probe=probe(http_get=http_get_action(port=8080, path='/api/v1/health')),
)

# Go tests
local_resource(
    'server-tests',
    cmd='cd server && go test ./... -v -count=1',
    deps=['server'],
    ignore=['server/coverage.out'],
    env={
        'DATABASE_URL': 'postgres://screenspace:devpassword@localhost:5432/screenspace?sslmode=disable',
        'S3_ENDPOINT': 'http://localhost:9000',
        'S3_ACCESS_KEY': 'minioadmin',
        'S3_SECRET_KEY': 'minioadmin',
        'JWT_SECRET': 'test-secret',
    },
    resource_deps=['postgres', 'minio-init'],
    auto_init=False,
    trigger_mode=TRIGGER_MODE_MANUAL,
    labels=['test'],
)

# macOS app build check
local_resource(
    'app-build',
    cmd='cd app && swift build 2>&1',
    deps=['app/Sources'],
    auto_init=True,
    trigger_mode=TRIGGER_MODE_AUTO,
    labels=['app'],
)

# macOS app tests
local_resource(
    'app-tests',
    cmd='cd app && swift test 2>&1',
    deps=['app/Sources', 'app/Tests'],
    auto_init=False,
    trigger_mode=TRIGGER_MODE_MANUAL,
    labels=['test'],
)

# macOS app run (manual, since it's a GUI app)
local_resource(
    'app-run',
    serve_cmd='cd app && swift run',
    auto_init=False,
    trigger_mode=TRIGGER_MODE_MANUAL,
    labels=['app'],
)

# E2E smoke test (manual)
local_resource(
    'e2e-test',
    cmd='bash tests/e2e/smoke_test.sh',
    auto_init=False,
    trigger_mode=TRIGGER_MODE_MANUAL,
    resource_deps=['server'],
    labels=['test'],
)

# Useful links
print("""
=============================================
  ScreenSpace Dev Environment
=============================================

  API Server:    http://localhost:8080
  Health Check:  http://localhost:8080/health
  MinIO Console: http://localhost:9001 (minioadmin/minioadmin)
  Postgres:      localhost:5432 (screenspace/devpassword)

  Admin:         admin@screenspace.dev / password
  User:          user@screenspace.dev / password

  To run the macOS app:
    Click 'app-run' in the Tilt UI, or:
    cd app && swift run

=============================================
""")
