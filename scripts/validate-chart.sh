#!/bin/bash

# Helm Chart Validation Script
# This script validates the Helm chart configuration before release

set -euo pipefail

CHART_DIR="deploy/charts/kite"
TEMP_DIR=$(mktemp -d)

echo "🔍 Validating Helm Chart..."

# Check if chart directory exists
if [ ! -d "$CHART_DIR" ]; then
    echo "❌ Chart directory '$CHART_DIR' not found"
    exit 1
fi

# Check if Chart.yaml exists
if [ ! -f "$CHART_DIR/Chart.yaml" ]; then
    echo "❌ Chart.yaml not found in '$CHART_DIR'"
    exit 1
fi

# Validate Chart.yaml structure
echo "📋 Checking Chart.yaml structure..."
if ! grep -q "^name:" "$CHART_DIR/Chart.yaml"; then
    echo "❌ Chart name not found in Chart.yaml"
    exit 1
fi

if ! grep -q "^version:" "$CHART_DIR/Chart.yaml"; then
    echo "❌ Chart version not found in Chart.yaml"
    exit 1
fi

echo "✅ Chart.yaml structure is valid"

# Lint the chart
echo "🔍 Linting Helm chart..."
if helm lint "$CHART_DIR"; then
    echo "✅ Chart linting passed"
else
    echo "❌ Chart linting failed"
    exit 1
fi

# Test chart packaging
echo "📦 Testing chart packaging..."
if helm package "$CHART_DIR" --destination "$TEMP_DIR"; then
    echo "✅ Chart packaging successful"
    PACKAGE_FILE=$(ls "$TEMP_DIR"/*.tgz)
    echo "📦 Package created: $(basename "$PACKAGE_FILE")"
else
    echo "❌ Chart packaging failed"
    exit 1
fi

# Test template rendering
echo "🔧 Testing template rendering..."
if helm template test-release "$CHART_DIR" > "$TEMP_DIR/rendered.yaml"; then
    echo "✅ Template rendering successful"
else
    echo "❌ Template rendering failed"
    exit 1
fi

# Default values ship sqlite on a persistent volume: the default render must
# contain the PVC and the /data mount.
if ! grep -E -q "kind:\s*PersistentVolumeClaim" "$TEMP_DIR/rendered.yaml"; then
    echo "❌ PersistentVolumeClaim not found in default rendered output"
    rm -rf "$TEMP_DIR"
    exit 1
fi
if ! grep -E -q "mountPath:\s*/data" "$TEMP_DIR/rendered.yaml"; then
    echo "❌ Default sqlite mountPath '/data' not found in rendered output"
    rm -rf "$TEMP_DIR"
    exit 1
fi

# Validate rendered YAML
# echo "📋 Validating rendered YAML..."
# if kubectl apply --dry-run=client -f "$TEMP_DIR/rendered.yaml" > /dev/null 2>&1; then
#     echo "✅ Rendered YAML is valid"
# else
#     echo "❌ Rendered YAML validation failed"
#     exit 1
# fi

# Test with different values (sqlite persistence)
echo "🔧 Testing with custom values (sqlite persistence)..."
cat > "$TEMP_DIR/test-values-sqlite.yaml" << EOF
replicaCount: 2
image:
    tag: "test"
service:
    type: LoadBalancer
db:
    type: sqlite
    sqlite:
        persistence:
            pvc:
                enabled: true
                size: 1Gi
EOF

if helm template test-release "$CHART_DIR" -f "$TEMP_DIR/test-values-sqlite.yaml" > "$TEMP_DIR/rendered-custom-sqlite.yaml"; then
        echo "✅ Custom values (sqlite) rendering successful"
else
        echo "❌ Custom values (sqlite) rendering failed"
        exit 1
fi


# Content checks for sqlite rendering
echo "📋 Verifying rendered content for sqlite persistence..."
RENDERED_SQLITE="$TEMP_DIR/rendered-custom-sqlite.yaml"
cat "$RENDERED_SQLITE"
fail() { echo "❌ $1"; rm -rf "$TEMP_DIR"; exit 1; }

# Ensure a PVC resource was generated for sqlite
if ! grep -E -q "kind:\s*PersistentVolumeClaim" "$RENDERED_SQLITE"; then
    fail "PersistentVolumeClaim not found in sqlite rendered output"
fi

# Ensure the PVC name or claim reference contains 'kite-storage'
if ! grep -E -q "kite-storage" "$RENDERED_SQLITE"; then
    fail "sqlite PVC name or reference not found in rendered output"
fi

# Ensure the mountPath for sqlite (default /data) is present in the Pod spec
if ! grep -E -q "mountPath:\s*/data" "$RENDERED_SQLITE"; then
    fail "Expected sqlite mountPath '/data' not found in rendered output"
fi

echo "✅ Sqlite rendered content looks correct"

# Test with an external postgres DSN provided
echo "🔧 Testing with custom values (external postgres dsn)..."
cat > "$TEMP_DIR/test-values-postgres.yaml" << EOF
replicaCount: 1
db:
    type: postgres
    dsn: "host=127.0.0.1 port=5432 user=test password=test dbname=kite sslmode=disable"
EOF

if helm template test-release "$CHART_DIR" -f "$TEMP_DIR/test-values-postgres.yaml" > "$TEMP_DIR/rendered-custom-postgres.yaml"; then
        echo "✅ Custom values (postgres) rendering successful"
else
        echo "❌ Custom values (postgres) rendering failed"
        exit 1
fi

# Content checks for external postgres rendering
echo "📋 Verifying rendered content for postgres DSN..."
RENDERED_PG="$TEMP_DIR/rendered-custom-postgres.yaml"

# External database settings are delivered through the generated Secret, and
# no sqlite PVC should be rendered for external database types.
if ! grep -E -q "^\s*DB_DSN:" "$RENDERED_PG"; then
    fail "DB_DSN not found in secret for external postgres dsn render"
fi
if grep -E -q "kind:\s*PersistentVolumeClaim" "$RENDERED_PG"; then
    fail "PersistentVolumeClaim unexpectedly present in external postgres rendered output"
fi

echo "✅ Postgres rendered content looks correct"

# Clean up
rm -rf "$TEMP_DIR"

echo "🎉 All validations passed! Chart is ready for release."
